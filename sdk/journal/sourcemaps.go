package journal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SourceMapResult reports what an upload run did.
type SourceMapResult struct {
	// Found is how many .map files were on disk, Uploaded how many the server
	// did not already hold, Skipped the rest, and Failed the ones it refused.
	Found    int
	Uploaded int
	Skipped  int
	Failed   int
}

// UploadSourceMaps ships every .map in dir to Journal for one release.
//
// It is meant to be called once at application startup, from the process that
// serves the build the maps belong to. That placement is deliberate: the maps
// travel inside the image alongside the bundle they explain, so there is no CI
// step to remember, no build secret, and no way for the two to drift apart. Go
// stamps no VCS data into a container build — every Dockerfile here excludes
// .git — so reading the release out of the built client is also the only
// reliable way to name it.
//
// It asks the server what it already holds and sends only the difference, so a
// restart, a rollback or a second replica costs one request. Failures are
// returned, never fatal: an app that cannot upload its maps must still boot,
// because unreadable stack traces are a worse day, not a broken one.
func UploadSourceMaps(ctx context.Context, cfg Config, dir, release string) (SourceMapResult, error) {
	var result SourceMapResult

	release = strings.TrimSpace(release)
	if release == "" {
		return result, fmt.Errorf("journal: release is required to upload source maps")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if base == "" || strings.TrimSpace(cfg.Token) == "" {
		return result, fmt.Errorf("journal: URL and Token are required to upload source maps")
	}

	client := cfg.HTTPClient
	if client == nil {
		// Generous next to the SDK's own 10s: a map is megabytes, and this
		// runs once at boot rather than on a request path.
		client = &http.Client{Timeout: 60 * time.Second}
	}

	maps, err := findSourceMaps(dir)
	if err != nil {
		return result, err
	}
	result.Found = len(maps)
	if result.Found == 0 {
		return result, nil
	}

	held, err := heldFiles(ctx, client, base, cfg.Token, release)
	if err != nil {
		return result, err
	}

	var firstFailure error
	for _, path := range maps {
		// The server keys on the bundle's basename, because a stack frame
		// carries a URL whose origin and prefix depend on how the app is
		// served while the file itself does not.
		name := strings.TrimSuffix(filepath.Base(path), ".map")
		if held[name] {
			result.Skipped++
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return result, fmt.Errorf("journal: reading %s: %w", path, err)
		}
		if err := uploadOne(ctx, client, base, cfg.Token, release, name, content); err != nil {
			// One rejected map must not cost the rest of the build. A
			// bundler emits maps of wildly different shapes and a single
			// odd one is not a reason to leave every other trace
			// unreadable, so the run continues and reports.
			result.Failed++
			if firstFailure == nil {
				firstFailure = err
			}
			continue
		}
		result.Uploaded++
	}

	if firstFailure != nil {
		return result, fmt.Errorf("journal: %d of %d source maps were refused, first: %w", result.Failed, result.Found, firstFailure)
	}
	return result, nil
}

func findSourceMaps(dir string) ([]string, error) {
	if dir == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// A deployment that ships no maps is the normal case, not a
			// failure: most apps have not adopted this.
			return nil, nil
		}
		return nil, fmt.Errorf("journal: source map directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("journal: %s is not a directory", dir)
	}

	var maps []string
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".map") {
			maps = append(maps, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("journal: walking %s: %w", dir, err)
	}
	return maps, nil
}

func heldFiles(ctx context.Context, client *http.Client, base, token, release string) (map[string]bool, error) {
	endpoint := base + "/sourcemaps?release=" + url.QueryEscape(release)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("journal: listing source maps: %w", err)
	}
	defer drain(response)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("journal: listing source maps: %s", statusOf(response))
	}
	var payload struct {
		Files []string `json:"files"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("journal: listing source maps: %w", err)
	}

	held := make(map[string]bool, len(payload.Files))
	for _, file := range payload.Files {
		held[file] = true
	}
	return held, nil
}

func uploadOne(ctx context.Context, client *http.Client, base, token, release, file string, content []byte) error {
	body, err := json.Marshal(map[string]string{
		"release": release,
		"file":    file,
		"map":     string(content),
	})
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/sourcemaps", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("journal: uploading %s: %w", file, err)
	}
	defer drain(response)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("journal: uploading %s: %s", file, statusOf(response))
	}
	return nil
}

// drain reads and closes a response body so the connection is reusable.
func drain(response *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	_ = response.Body.Close()
}

func statusOf(response *http.Response) string {
	message, _ := io.ReadAll(io.LimitReader(response.Body, 512))
	trimmed := strings.TrimSpace(string(message))
	if trimmed == "" {
		return response.Status
	}
	return response.Status + ": " + trimmed
}
