package journal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const maxServerBatch = 1000

// Config configures a Client. URL and Token are required; everything else
// has sensible defaults.
type Config struct {
	// URL is the Journal API base URL, e.g. http://journal-api:4010.
	URL string
	// Token is a per-app API key (journal_<app>_...) or the legacy INGEST_TOKEN.
	Token string
	// App is the source app name. Leave empty when Token is a per-app key —
	// the server fills it from the key's scope.
	App string
	// FlushInterval is how often buffered entries are shipped. Default 2s.
	FlushInterval time.Duration
	// MaxBatch triggers an early flush when the buffer reaches this size. Default 200.
	MaxBatch int
	// BufferCap bounds the buffer; the oldest entries are dropped beyond it. Default 5000.
	BufferCap int
	// HTTPClient overrides the default client (10s timeout).
	HTTPClient *http.Client
}

// Entry is one log entry as accepted by POST /ingest.
type Entry struct {
	App     string         `json:"app,omitempty"`
	Level   string         `json:"level,omitempty"`
	Message string         `json:"message"`
	Ts      string         `json:"ts,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// Client ships log entries to Journal in the background. Shipping is
// best-effort and never blocks or panics: batches are retried on 429/5xx and
// network errors, dropped on any other 4xx, and the oldest entries are
// dropped if the buffer overflows while Journal is unreachable.
type Client struct {
	cfg  Config
	http *http.Client

	mu      sync.Mutex
	buf     []Entry
	dropped int

	kick chan struct{}
	stop chan struct{}
	done chan struct{}
}

// New starts a Client and its background shipper goroutine. Call Close to
// flush and stop it.
func New(cfg Config) *Client {
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 2 * time.Second
	}
	if cfg.MaxBatch <= 0 {
		cfg.MaxBatch = 200
	}
	if cfg.MaxBatch > maxServerBatch {
		cfg.MaxBatch = maxServerBatch
	}
	if cfg.BufferCap <= 0 {
		cfg.BufferCap = 5000
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	c := &Client{
		cfg:  cfg,
		http: httpClient,
		kick: make(chan struct{}, 1),
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go c.run()
	return c
}

// Log buffers one entry. Level must be debug, info, warn or error; anything
// else is stored as-is and normalized by the server.
func (c *Client) Log(level, message string, meta map[string]any) {
	entry := Entry{
		App:     c.cfg.App,
		Level:   level,
		Message: message,
		Ts:      time.Now().UTC().Format(time.RFC3339Nano),
		Meta:    meta,
	}
	c.mu.Lock()
	if len(c.buf) >= c.cfg.BufferCap {
		c.buf = c.buf[1:]
		c.dropped++
	}
	c.buf = append(c.buf, entry)
	full := len(c.buf) >= c.cfg.MaxBatch
	c.mu.Unlock()
	if full {
		select {
		case c.kick <- struct{}{}:
		default:
		}
	}
}

// Debug logs at debug level.
func (c *Client) Debug(message string, meta map[string]any) { c.Log("debug", message, meta) }

// Info logs at info level.
func (c *Client) Info(message string, meta map[string]any) { c.Log("info", message, meta) }

// Warn logs at warn level.
func (c *Client) Warn(message string, meta map[string]any) { c.Log("warn", message, meta) }

// Error logs at error level.
func (c *Client) Error(message string, meta map[string]any) { c.Log("error", message, meta) }

// Dropped reports how many entries were discarded due to buffer overflow.
func (c *Client) Dropped() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropped
}

// Close flushes buffered entries once and stops the shipper goroutine.
func (c *Client) Close() {
	close(c.stop)
	<-c.done
}

func (c *Client) run() {
	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.kick:
			c.flush()
		case <-c.stop:
			c.flush()
			close(c.done)
			return
		}
	}
}

func (c *Client) flush() {
	c.mu.Lock()
	if len(c.buf) == 0 {
		c.mu.Unlock()
		return
	}
	size := len(c.buf)
	if size > maxServerBatch {
		size = maxServerBatch
	}
	batch := make([]Entry, size)
	copy(batch, c.buf[:size])
	c.mu.Unlock()

	retryable := c.ship(batch)
	if retryable {
		return
	}
	c.mu.Lock()
	c.buf = c.buf[size:]
	c.mu.Unlock()
}

func (c *Client) ship(entries []Entry) bool {
	body, err := json.Marshal(map[string][]Entry{"entries": entries})
	if err != nil {
		return false
	}
	request, err := http.NewRequest(http.MethodPost, c.cfg.URL+"/ingest", bytes.NewReader(body))
	if err != nil {
		return false
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	response, err := c.http.Do(request)
	if err != nil {
		return true
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 {
		return true
	}
	return false
}
