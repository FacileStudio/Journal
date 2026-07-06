package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"
)

type tailer struct {
	docker *dockerClient
	ship   *shipper
	log    *slog.Logger
	id     string
	app    string
	tty    bool
	lastTS time.Time
}

func (t *tailer) run(ctx context.Context) {
	since := time.Now()
	for {
		body, err := t.docker.streamLogs(ctx, t.id, sinceParam(since))
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			t.log.Warn("log stream connect failed", "container", shortID(t.id), "error", err)
		} else {
			t.consume(body)
			body.Close()
			if ctx.Err() != nil {
				return
			}
			t.log.Info("log stream ended, reconnecting", "container", shortID(t.id))
		}
		if !t.lastTS.IsZero() {
			since = t.lastTS.Add(time.Nanosecond)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (t *tailer) consume(body io.Reader) {
	var feed func(p []byte)
	var flush func()
	if t.tty {
		lb := &lineBuffer{stream: streamStdout, emit: t.handleLine}
		feed, flush = lb.feed, lb.flush
	} else {
		d := newDemuxer(t.handleLine)
		feed, flush = d.feed, d.flush
	}
	buf := make([]byte, 32*1024)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			feed(buf[:n])
		}
		if err != nil {
			flush()
			return
		}
	}
}

func (t *tailer) handleLine(stream byte, raw []byte) {
	ts, message := splitTimestamp(string(raw))
	if !ts.IsZero() {
		t.lastTS = ts
	}
	t.ship.add(mapLine(t.app, t.id, stream, ts, message))
}

func sinceParam(t time.Time) string {
	return fmt.Sprintf("%d.%09d", t.Unix(), t.Nanosecond())
}
