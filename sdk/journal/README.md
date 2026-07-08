# journal — Go SDK

Stdlib-only client for shipping logs to Journal from any Go app.

```sh
go get github.com/FacileStudio/Journal/sdk/journal@main
```

## Direct use

```go
client := journal.New(journal.Config{
	URL:   os.Getenv("JOURNAL_URL"),
	Token: os.Getenv("JOURNAL_TOKEN"),
})
defer client.Close()

client.Info("started", map[string]any{"version": "1.2.3"})
client.Error("upload failed", map[string]any{"request_id": rid, "file_id": 42})
```

## slog tee (recommended)

Wrap your existing handler once at startup — everything the app already logs
flows to Journal unchanged, levels and attrs included:

```go
client := journal.New(journal.Config{URL: url, Token: token})
defer client.Close()
slog.SetDefault(slog.New(journal.NewHandler(client, slog.Default().Handler())))
```

Attrs become `meta` (groups flattened to dotted keys), so `slog.String("request_id", rid)`
lights up the dashboard's request_id pivot.

## Behavior

- Batched: flushes every 2s or at 200 entries, never blocks the caller.
- Best-effort: retries on 429/5xx and network errors, drops on other 4xx,
  drops oldest beyond a 5000-entry buffer when Journal is unreachable —
  Journal being down never takes your app down.
- With a per-app key (`journal_<app>_…`) leave `Config.App` empty; the server
  fills it from the key's scope. The legacy shared token needs `App` set.
- `Close()` flushes once on shutdown.
