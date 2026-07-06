package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type config struct {
	journalURL       string
	journalToken     string
	dockerSock       string
	discoverInterval time.Duration
}

func loadConfig() config {
	cfg := config{
		journalURL:       envOr("JOURNAL_URL", "http://journal-api:4010"),
		journalToken:     os.Getenv("JOURNAL_TOKEN"),
		dockerSock:       envOr("DOCKER_SOCK", "/var/run/docker.sock"),
		discoverInterval: 30 * time.Second,
	}
	if raw := os.Getenv("DISCOVER_INTERVAL"); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
			cfg.discoverInterval = time.Duration(secs) * time.Second
		}
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := loadConfig()
	if cfg.journalToken == "" {
		log.Error("JOURNAL_TOKEN is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	docker := newDockerClient(cfg.dockerSock)
	ship := newShipper(cfg.journalURL, cfg.journalToken, log)

	shipDone := make(chan struct{})
	go func() {
		defer close(shipDone)
		ship.run(ctx)
	}()

	hostname, _ := os.Hostname()
	log.Info("collector started",
		"journal_url", cfg.journalURL,
		"docker_sock", cfg.dockerSock,
		"discover_interval", cfg.discoverInterval.String(),
		"hostname", hostname,
	)

	runDiscovery(ctx, docker, ship, hostname, cfg.discoverInterval, log)
	<-shipDone

	flushCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ship.flush(flushCtx)
	log.Info("collector stopped")
}
