package main

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

func runDiscovery(ctx context.Context, docker *dockerClient, ship *shipper, selfHost string, interval time.Duration, log *slog.Logger) {
	tails := map[string]context.CancelFunc{}
	var wg sync.WaitGroup

	sweep := func() {
		listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		containers, err := docker.listContainers(listCtx)
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				log.Warn("container discovery failed", "error", err)
			}
			return
		}
		seen := map[string]bool{}
		for _, c := range containers {
			if skipContainer(c, selfHost) {
				continue
			}
			seen[c.ID] = true
			if _, ok := tails[c.ID]; ok {
				continue
			}
			inspectCtx, cancelInspect := context.WithTimeout(ctx, 10*time.Second)
			tty, err := docker.inspectTTY(inspectCtx, c.ID)
			cancelInspect()
			if err != nil {
				if ctx.Err() == nil {
					log.Warn("container inspect failed", "container", shortID(c.ID), "error", err)
				}
				continue
			}
			app := appName(c)
			tailCtx, cancelTail := context.WithCancel(ctx)
			tails[c.ID] = cancelTail
			t := &tailer{
				docker: docker,
				ship:   ship,
				log:    log,
				id:     c.ID,
				app:    app,
				tty:    tty,
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				t.run(tailCtx)
			}()
			log.Info("tailing container", "container", shortID(c.ID), "app", app, "tty", tty)
		}
		for id, cancelTail := range tails {
			if !seen[id] {
				cancelTail()
				delete(tails, id)
				log.Info("stopped tailing container", "container", shortID(id))
			}
		}
	}

	sweep()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			for _, cancelTail := range tails {
				cancelTail()
			}
			wg.Wait()
			return
		case <-ticker.C:
			sweep()
		}
	}
}

func skipContainer(c containerSummary, selfHost string) bool {
	if selfHost != "" && strings.HasPrefix(c.ID, selfHost) {
		return true
	}
	return c.Labels["journal.ignore"] == "true"
}

func appName(c containerSummary) string {
	if v := c.Labels["journal.app"]; v != "" {
		return v
	}
	if len(c.Names) > 0 && c.Names[0] != "" {
		return strings.TrimPrefix(c.Names[0], "/")
	}
	return shortID(c.ID)
}
