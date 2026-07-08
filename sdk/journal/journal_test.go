package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type capture struct {
	mu         sync.Mutex
	batches    [][]Entry
	failures   int
	rateLimits int
	requests   int
}

func (c *capture) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.requests++
		if c.failures > 0 {
			c.failures--
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if c.rateLimits > 0 {
			c.rateLimits--
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload struct {
			Entries []Entry `json:"entries"`
		}
		json.Unmarshal(body, &payload)
		c.batches = append(c.batches, payload.Entries)
		w.WriteHeader(http.StatusCreated)
	}
}

func (c *capture) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, b := range c.batches {
		n += len(b)
	}
	return n
}

func (c *capture) requestCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.requests
}

func (c *capture) entries() []Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	var all []Entry
	for _, b := range c.batches {
		all = append(all, b...)
	}
	return all
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for !cond() {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %s", what)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestClientShipsAndRetries(t *testing.T) {
	sink := &capture{failures: 1}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", App: "test", FlushInterval: 10 * time.Millisecond})
	client.Info("one", nil)
	client.Error("two", map[string]any{"k": "v"})

	waitFor(t, "entries shipped after retry", func() bool { return sink.total() >= 2 })
	client.Close()
	if sink.total() != 2 {
		t.Fatalf("expected exactly 2 entries, got %d", sink.total())
	}
	first := sink.entries()[0]
	if first.App != "test" || first.Level != "info" || first.Message != "one" || first.Ts == "" {
		t.Fatalf("unexpected entry: %+v", first)
	}
	if _, err := time.Parse(time.RFC3339, first.Ts); err != nil {
		t.Fatalf("Ts not parseable as RFC3339: %v", err)
	}
}

func TestTsParsesAsRFC3339(t *testing.T) {
	if _, err := time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("RFC3339Nano output must parse as RFC3339: %v", err)
	}
}

func TestClientDropsOldestOnOverflow(t *testing.T) {
	client := New(Config{URL: "http://127.0.0.1:1", Token: "t", BufferCap: 3, FlushInterval: time.Hour})
	for i := 0; i < 5; i++ {
		client.Info("m", nil)
	}
	if got := client.Dropped(); got != 2 {
		t.Fatalf("expected 2 dropped, got %d", got)
	}
	client.Close()
}

func TestCloseIsIdempotentAndLogAfterCloseIsSafe(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 10 * time.Millisecond})
	client.Info("one", nil)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.Close()
		}()
	}
	wg.Wait()
	client.Close()
	client.Info("after close", nil)
	if sink.total() != 1 {
		t.Fatalf("expected 1 entry drained by Close, got %d", sink.total())
	}
}

func TestCloseDrainsMoreThanOneServerBatch(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: time.Hour})
	for i := 0; i < maxServerBatch+50; i++ {
		client.Info("m", nil)
	}
	client.Close()
	if sink.total() != maxServerBatch+50 {
		t.Fatalf("expected %d entries drained, got %d", maxServerBatch+50, sink.total())
	}
}

func TestMetaSanitization(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 10 * time.Millisecond})
	meta := map[string]any{
		"err":    errors.New("disk full"),
		"chan":   make(chan int),
		"fn":     func() {},
		"cmplx":  complex(1, 2),
		"nan":    math.NaN(),
		"inf":    math.Inf(1),
		"dur":    1500 * time.Millisecond,
		"nested": map[string]any{"inner": errors.New("inner boom")},
		"list":   []any{errors.New("in list"), 7},
		"ok":     42,
		"str":    "plain",
	}
	client.Error("boom", meta)
	meta["ok"] = "mutated after Log"

	waitFor(t, "sanitized entry shipped", func() bool { return sink.total() >= 1 })
	client.Close()
	got := sink.entries()[0].Meta
	if got["err"] != "disk full" {
		t.Fatalf("error not stringified: %#v", got["err"])
	}
	if s, ok := got["chan"].(string); !ok || s == "" {
		t.Fatalf("chan not stringified: %#v", got["chan"])
	}
	if s, ok := got["fn"].(string); !ok || s == "" {
		t.Fatalf("func not stringified: %#v", got["fn"])
	}
	if got["cmplx"] != "(1+2i)" {
		t.Fatalf("complex not stringified: %#v", got["cmplx"])
	}
	if got["nan"] != "NaN" || got["inf"] != "+Inf" {
		t.Fatalf("non-finite floats not stringified: %#v %#v", got["nan"], got["inf"])
	}
	if got["dur"] != "1.5s" {
		t.Fatalf("Stringer not stringified: %#v", got["dur"])
	}
	nested, ok := got["nested"].(map[string]any)
	if !ok || nested["inner"] != "inner boom" {
		t.Fatalf("nested error not stringified: %#v", got["nested"])
	}
	list, ok := got["list"].([]any)
	if !ok || list[0] != "in list" || list[1] != float64(7) {
		t.Fatalf("list not sanitized: %#v", got["list"])
	}
	if got["ok"] != float64(42) || got["str"] != "plain" {
		t.Fatalf("plain values altered: %#v %#v", got["ok"], got["str"])
	}
}

func TestNoLossNoDuplicationUnderRetryAndOverflow(t *testing.T) {
	sink := &capture{failures: 3}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	total := 200
	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 5 * time.Millisecond, BufferCap: 50, MaxBatch: 10})
	for i := 0; i < total; i++ {
		client.Info(fmt.Sprintf("m%d", i), nil)
	}
	waitFor(t, "retries to settle", func() bool { return sink.total()+client.Dropped() >= total })
	client.Close()

	seen := map[string]int{}
	for _, entry := range sink.entries() {
		seen[entry.Message]++
		if seen[entry.Message] > 1 {
			t.Fatalf("duplicate entry shipped: %s", entry.Message)
		}
	}
	if got := sink.total() + client.Dropped(); got != total {
		t.Fatalf("shipped(%d) + dropped(%d) = %d, want %d", sink.total(), client.Dropped(), got, total)
	}
}

func TestRetryAfterBackoff(t *testing.T) {
	sink := &capture{rateLimits: 1}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 5 * time.Millisecond})
	client.Info("one", nil)
	waitFor(t, "first 429 response", func() bool { return sink.requestCount() >= 1 })
	time.Sleep(150 * time.Millisecond)
	if got := sink.requestCount(); got != 1 {
		t.Fatalf("expected backoff after 429, got %d requests", got)
	}
	client.Close()
	if sink.total() != 1 {
		t.Fatalf("expected entry shipped on Close despite backoff, got %d", sink.total())
	}
}

func TestConcurrentLogFlushClose(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: time.Millisecond, MaxBatch: 8})
	var loggers sync.WaitGroup
	for g := 0; g < 8; g++ {
		loggers.Add(1)
		go func(g int) {
			defer loggers.Done()
			for i := 0; i < 100; i++ {
				meta := map[string]any{"g": g, "i": i}
				client.Log("info", "m", meta)
				meta["i"] = -1
			}
		}(g)
	}
	loggers.Wait()

	var stragglers sync.WaitGroup
	for g := 0; g < 4; g++ {
		stragglers.Add(1)
		go func() {
			defer stragglers.Done()
			for i := 0; i < 50; i++ {
				client.Info("straggler", nil)
			}
		}()
	}
	var closers sync.WaitGroup
	for i := 0; i < 4; i++ {
		closers.Add(1)
		go func() {
			defer closers.Done()
			client.Close()
		}()
	}
	stragglers.Wait()
	closers.Wait()
	if got := sink.total(); got < 800 {
		t.Fatalf("expected at least the 800 pre-Close entries, got %d", got)
	}
}

func TestClientTrimsTrailingSlash(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(func() http.HandlerFunc {
		inner := sink.handler()
		return func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "//") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			inner(w, r)
		}
	}())
	defer server.Close()

	client := New(Config{URL: server.URL + "/", Token: "t", FlushInterval: 10 * time.Millisecond})
	client.Info("one", nil)
	waitFor(t, "entry shipped with trimmed URL", func() bool { return sink.total() >= 1 })
	client.Close()
}

func TestHandlerLevelsGroupsAndAttrs(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 10 * time.Millisecond})
	logger := slog.New(NewHandler(client, nil)).With("service", "api").WithGroup("req")
	logger.Warn("slow", slog.String("path", "/x"), slog.Group("db", slog.Int("ms", 42)))
	logger.Debug("hidden")

	waitFor(t, "handler entry shipped", func() bool { return sink.total() >= 1 })
	client.Close()
	if sink.total() != 1 {
		t.Fatalf("debug should be filtered without next handler, got %d entries", sink.total())
	}
	entry := sink.entries()[0]
	if entry.Level != "warn" || entry.Message != "slow" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.Meta["service"] != "api" || entry.Meta["req.path"] != "/x" || entry.Meta["req.db.ms"] != float64(42) {
		t.Fatalf("unexpected meta: %+v", entry.Meta)
	}
	if _, err := time.Parse(time.RFC3339, entry.Ts); err != nil {
		t.Fatalf("handler Ts not parseable as RFC3339: %v", err)
	}
}

type stringLogValuer struct{}

func (stringLogValuer) LogValue() slog.Value { return slog.StringValue("resolved") }

func TestHandlerContract(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 10 * time.Millisecond})
	base := NewHandler(client, nil)
	child := base.WithAttrs([]slog.Attr{slog.String("a", "1")})
	grand := child.WithGroup("g").WithAttrs([]slog.Attr{slog.String("b", "2")})

	slog.New(base).Info("base")
	slog.New(child).Info("child")
	slog.New(grand).Info("grand", slog.Int("c", 3))
	slog.New(base).Info("inline", slog.Group("", slog.Int("x", 1)))
	slog.New(base).Info("emptygroup", slog.Group("empty"))
	slog.New(base).Info("valuer", slog.Any("v", stringLogValuer{}))

	waitFor(t, "contract entries shipped", func() bool { return sink.total() >= 6 })
	client.Close()

	byMessage := map[string]Entry{}
	for _, entry := range sink.entries() {
		byMessage[entry.Message] = entry
	}
	if byMessage["base"].Meta != nil {
		t.Fatalf("base handler polluted by derived handlers: %+v", byMessage["base"].Meta)
	}
	childMeta := byMessage["child"].Meta
	if childMeta["a"] != "1" || len(childMeta) != 1 {
		t.Fatalf("child meta wrong: %+v", childMeta)
	}
	grandMeta := byMessage["grand"].Meta
	if grandMeta["a"] != "1" || grandMeta["g.b"] != "2" || grandMeta["g.c"] != float64(3) {
		t.Fatalf("grand meta wrong: %+v", grandMeta)
	}
	if byMessage["inline"].Meta["x"] != float64(1) {
		t.Fatalf("empty-key group not inlined: %+v", byMessage["inline"].Meta)
	}
	if byMessage["emptygroup"].Meta != nil {
		t.Fatalf("empty group not elided: %+v", byMessage["emptygroup"].Meta)
	}
	if byMessage["valuer"].Meta["v"] != "resolved" {
		t.Fatalf("LogValuer not resolved: %+v", byMessage["valuer"].Meta)
	}
}

func TestLevelName(t *testing.T) {
	cases := map[slog.Level]string{
		slog.LevelDebug:     "debug",
		slog.LevelInfo:      "info",
		slog.LevelWarn:      "warn",
		slog.LevelError:     "error",
		slog.LevelError + 4: "error",
	}
	for level, want := range cases {
		if got := levelName(level); got != want {
			t.Fatalf("level %v: got %s want %s", level, got, want)
		}
	}
}
