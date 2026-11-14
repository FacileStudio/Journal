package sourcemaps

import (
	"context"
	"encoding/json"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"github.com/go-sourcemap/sourcemap"
	"gorm.io/gorm"
)

// maxCachedConsumers bounds the parsed-map cache. Parsing a map is the
// expensive part of resolving a stack, and a page of errors from one release
// hits the same handful of chunks, so a small cache removes nearly all of it.
// Bounded because a map is megabytes once parsed and releases accumulate.
const maxCachedConsumers = 32

// Service stores source maps and resolves minified traces against a bounded
// in-memory consumer cache.
type Service struct {
	orm *gorm.DB

	mu       sync.Mutex
	consumer map[string]*sourcemap.Consumer
	order    []string
}

// NewService wires a source-map service onto the database.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm, consumer: map[string]*sourcemap.Consumer{}}
}

var fileNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,200}$`)

// isEmptyMap reports a well-formed v3 map that maps nothing, which a bundler
// emits routinely and which is not a reason to refuse an upload.
func isEmptyMap(content string) bool {
	var probe struct {
		Version  int    `json:"version"`
		Mappings string `json:"mappings"`
	}
	if json.Unmarshal([]byte(content), &probe) != nil {
		return false
	}
	return probe.Version == 3 && strings.TrimSpace(probe.Mappings) == ""
}

// Store saves one map, ignoring a re-upload of one already held.
//
// Idempotent because the uploader runs at application boot: a restart, a
// rollback and a scaled-up replica all re-offer the same maps, and none of
// those is an error worth surfacing. The map is parsed before it is stored:
// one that cannot be read is worse than no map, because it looks uploaded and
// silently resolves nothing. An empty-mappings map is the one exception — Vite
// emits them for assets that compile to nothing worth mapping, they are
// perfectly valid v3 documents, and failing the upload over one would fail the
// deploy of a build whose real maps are fine — so it is accepted and nothing
// is stored.
func (s *Service) Store(ctx context.Context, workspaceID int64, app, release, file, content string) (bool, error) {
	release = strings.TrimSpace(release)
	file = path.Base(strings.TrimSpace(file))
	if release == "" {
		return false, errors.Invalid("release is required")
	}
	if !fileNamePattern.MatchString(file) {
		return false, errors.Invalid("file must be a bare file name matching ^[A-Za-z0-9._-]{1,200}$")
	}
	if _, err := sourcemap.Parse("", []byte(content)); err != nil {
		if isEmptyMap(content) {
			return false, nil
		}
		return false, errors.Invalid("the map is not a valid source map: " + err.Error())
	}

	row := schemas.SourceMap{WorkspaceID: workspaceID, App: app, Release: release, File: file, Content: content, Bytes: len(content)}
	result := s.orm.WithContext(ctx).
		Where("workspace_id = ? AND app = ? AND release = ? AND file = ?", workspaceID, app, release, file).
		FirstOrCreate(&row)
	if result.Error != nil {
		return false, errors.Internal("failed to store the source map", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// Files lists what is already held for a release, so an uploader sends only
// what is missing.
func (s *Service) Files(ctx context.Context, workspaceID int64, app, release string) ([]string, error) {
	files := []string{}
	err := s.orm.WithContext(ctx).Model(&schemas.SourceMap{}).
		Where("workspace_id = ? AND app = ? AND release = ?", workspaceID, app, release).
		Order("file").Pluck("file", &files).Error
	if err != nil {
		return nil, errors.Internal("failed to list source maps", err)
	}
	return files, nil
}

// Releases summarises what is stored, for the dashboard.
func (s *Service) Releases(ctx context.Context, scope workspace.Scope) ([]ReleaseSummary, error) {
	var rows []struct {
		WorkspaceID int64
		App         string
		Release     string
		Files       int
		Bytes       int64
		CreatedAt   string
	}
	err := scope.Where(s.orm.WithContext(ctx).Model(&schemas.SourceMap{})).
		Select("workspace_id, app, release, count(*) as files, sum(bytes) as bytes, to_char(max(created_at) at time zone 'utc', 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') as created_at").
		Group("workspace_id, app, release").Order("max(created_at) desc").Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to summarise source maps", err)
	}

	out := make([]ReleaseSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, ReleaseSummary{WorkspaceID: row.WorkspaceID, App: row.App, Release: row.Release, Files: row.Files, Bytes: row.Bytes, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

// Delete drops one release's maps. Rolling a release back and forward should
// not leave a stale map claiming to explain the new build.
func (s *Service) Delete(ctx context.Context, scope workspace.Scope, workspaceID int64, app, release string) (int64, error) {
	if !scope.Contains(workspaceID) {
		return 0, errors.NotFound("no source maps for this release")
	}
	result := s.orm.WithContext(ctx).
		Where("workspace_id = ? AND app = ? AND release = ?", workspaceID, app, release).
		Delete(&schemas.SourceMap{})
	if result.Error != nil {
		return 0, errors.Internal("failed to delete the source maps", result.Error)
	}
	s.evictRelease(workspaceID, app, release)
	return result.RowsAffected, nil
}

// consumerFor returns a parsed map, or nil when none is held. A miss is the
// normal case — most apps upload nothing — so it must be cheap and quiet.
func (s *Service) consumerFor(ctx context.Context, workspaceID int64, app, release, file string) *sourcemap.Consumer {
	key := cacheKey(workspaceID, app, release) + file

	s.mu.Lock()
	if cached, ok := s.consumer[key]; ok {
		s.mu.Unlock()
		return cached
	}
	s.mu.Unlock()

	var row schemas.SourceMap
	err := s.orm.WithContext(ctx).
		Where("workspace_id = ? AND app = ? AND release = ? AND file = ?", workspaceID, app, release, file).
		First(&row).Error
	if err != nil {
		return nil
	}
	consumer, err := sourcemap.Parse("", []byte(row.Content))
	if err != nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.order) >= maxCachedConsumers {
		delete(s.consumer, s.order[0])
		s.order = s.order[1:]
	}
	s.consumer[key] = consumer
	s.order = append(s.order, key)
	return consumer
}

func cacheKey(workspaceID int64, app, release string) string {
	return strconv.FormatInt(workspaceID, 10) + "\x00" + app + "\x00" + release + "\x00"
}

func (s *Service) evictRelease(workspaceID int64, app, release string) {
	prefix := cacheKey(workspaceID, app, release)
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.order[:0]
	for _, key := range s.order {
		if strings.HasPrefix(key, prefix) {
			delete(s.consumer, key)
			continue
		}
		kept = append(kept, key)
	}
	s.order = kept
}
