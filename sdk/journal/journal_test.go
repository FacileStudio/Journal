package journal

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type capture struct {
	mu       sync.Mutex
	batches  [][]Entry
	failures int
}

func (c *capture) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.failures > 0 {
			c.failures--
			w.WriteHeader(http.StatusInternalServerError)
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

func TestClientShipsAndRetries(t *testing.T) {
	sink := &capture{failures: 1}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", App: "test", FlushInterval: 10 * time.Millisecond})
	client.Info("one", nil)
	client.Error("two", map[string]any{"k": "v"})

	deadline := time.After(2 * time.Second)
	for sink.total() < 2 {
		select {
		case <-deadline:
			t.Fatalf("entries not shipped after retry, got %d", sink.total())
		case <-time.After(10 * time.Millisecond):
		}
	}
	client.Close()
	if sink.total() != 2 {
		t.Fatalf("expected exactly 2 entries, got %d", sink.total())
	}
	first := sink.batches[len(sink.batches)-1][0]
	if first.App != "test" || first.Level != "info" || first.Message != "one" || first.Ts == "" {
		t.Fatalf("unexpected entry: %+v", first)
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

func TestHandlerLevelsGroupsAndAttrs(t *testing.T) {
	sink := &capture{}
	server := httptest.NewServer(sink.handler())
	defer server.Close()

	client := New(Config{URL: server.URL, Token: "t", FlushInterval: 10 * time.Millisecond})
	logger := slog.New(NewHandler(client, nil)).With("service", "api").WithGroup("req")
	logger.Warn("slow", slog.String("path", "/x"), slog.Group("db", slog.Int("ms", 42)))
	logger.Debug("hidden")

	deadline := time.After(2 * time.Second)
	for sink.total() < 1 {
		select {
		case <-deadline:
			t.Fatal("entry not shipped")
		case <-time.After(10 * time.Millisecond):
		}
	}
	client.Close()
	if sink.total() != 1 {
		t.Fatalf("debug should be filtered without next handler, got %d entries", sink.total())
	}
	entry := sink.batches[0][0]
	if entry.Level != "warn" || entry.Message != "slow" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.Meta["service"] != "api" || entry.Meta["req.path"] != "/x" || entry.Meta["req.db.ms"] != float64(42) {
		t.Fatalf("unexpected meta: %+v", entry.Meta)
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
