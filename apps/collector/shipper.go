package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	maxBuffer  = 5000
	flushCount = 200
	flushEvery = 2 * time.Second
)

type shipper struct {
	url    string
	token  string
	log    *slog.Logger
	client *http.Client
	kick   chan struct{}

	mu      sync.Mutex
	buf     []entry
	dropped int
}

func newShipper(baseURL, token string, log *slog.Logger) *shipper {
	return &shipper{
		url:    strings.TrimRight(baseURL, "/") + "/ingest",
		token:  token,
		log:    log,
		client: &http.Client{Timeout: 15 * time.Second},
		kick:   make(chan struct{}, 1),
	}
}

func (s *shipper) add(e entry) {
	s.mu.Lock()
	s.buf = append(s.buf, e)
	s.capLocked()
	n := len(s.buf)
	s.mu.Unlock()
	if n >= flushCount {
		select {
		case s.kick <- struct{}{}:
		default:
		}
	}
}

func (s *shipper) capLocked() {
	if over := len(s.buf) - maxBuffer; over > 0 {
		s.buf = append(s.buf[:0], s.buf[over:]...)
		s.dropped += over
	}
}

func (s *shipper) run(ctx context.Context) {
	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.kick:
		}
		s.flush(ctx)
	}
}

func (s *shipper) flush(ctx context.Context) {
	s.mu.Lock()
	if s.dropped > 0 {
		s.log.Warn("buffer full, dropped oldest entries", "dropped", s.dropped)
		s.dropped = 0
	}
	if len(s.buf) == 0 {
		s.mu.Unlock()
		return
	}
	batch := s.buf
	s.buf = nil
	s.mu.Unlock()

	status, err := s.post(ctx, batch)
	if err == nil {
		return
	}
	if status >= 400 && status < 500 && status != http.StatusTooManyRequests {
		s.log.Error("ingest rejected batch, dropping it", "status", status, "entries", len(batch))
		return
	}
	s.log.Warn("ingest failed, batch kept for retry", "error", err, "entries", len(batch))
	s.mu.Lock()
	s.buf = append(batch, s.buf...)
	s.capLocked()
	s.mu.Unlock()
}

func (s *shipper) post(ctx context.Context, batch []entry) (int, error) {
	payload, err := json.Marshal(map[string][]entry{"entries": batch})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.token)
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, fmt.Errorf("ingest returned status %d", resp.StatusCode)
}
