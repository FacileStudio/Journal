package sourcemaps

import (
	"context"
	"path"
	"regexp"
	"strings"
	"sync"

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

type Service struct {
	orm *gorm.DB

	mu       sync.Mutex
	consumer map[string]*sourcemap.Consumer
	order    []string
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm, consumer: map[string]*sourcemap.Consumer{}}
}

var fileNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,200}$`)

// Store saves one map, ignoring a re-upload of one already held.
//
// Idempotent because the uploader runs at application boot: a restart, a
// rollback and a scaled-up replica all re-offer the same maps, and none of
// those is an error worth surfacing.
func (s *Service) Store(ctx context.Context, app, release, file, content string) (bool, error) {
	release = strings.TrimSpace(release)
	file = path.Base(strings.TrimSpace(file))
	if release == "" {
		return false, errors.Invalid("release is required")
	}
	if !fileNamePattern.MatchString(file) {
		return false, errors.Invalid("file must be a bare file name matching ^[A-Za-z0-9._-]{1,200}$")
	}
	// Parse before storing: a map that cannot be read is worse than no map,
	// because it looks uploaded and silently resolves nothing.
	if _, err := sourcemap.Parse("", []byte(content)); err != nil {
		return false, errors.Invalid("the map is not a valid source map: " + err.Error())
	}

	row := schemas.SourceMap{App: app, Release: release, File: file, Content: content, Bytes: len(content)}
	result := s.orm.WithContext(ctx).
		Where("app = ? AND release = ? AND file = ?", app, release, file).
		FirstOrCreate(&row)
	if result.Error != nil {
		return false, errors.Internal("failed to store the source map", result.Error)
	}
	return result.RowsAffected > 0, nil
}

// Files lists what is already held for a release, so an uploader sends only
// what is missing.
func (s *Service) Files(ctx context.Context, app, release string) ([]string, error) {
	files := []string{}
	err := s.orm.WithContext(ctx).Model(&schemas.SourceMap{}).
		Where("app = ? AND release = ?", app, release).
		Order("file").Pluck("file", &files).Error
	if err != nil {
		return nil, errors.Internal("failed to list source maps", err)
	}
	return files, nil
}

// Releases summarises what is stored, for the dashboard.
func (s *Service) Releases(ctx context.Context) ([]ReleaseSummary, error) {
	var rows []struct {
		App       string
		Release   string
		Files     int
		Bytes     int64
		CreatedAt string
	}
	err := s.orm.WithContext(ctx).Model(&schemas.SourceMap{}).
		Select("app, release, count(*) as files, sum(bytes) as bytes, to_char(max(created_at) at time zone 'utc', 'YYYY-MM-DD\"T\"HH24:MI:SS\"Z\"') as created_at").
		Group("app, release").Order("max(created_at) desc").Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to summarise source maps", err)
	}

	out := make([]ReleaseSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, ReleaseSummary{App: row.App, Release: row.Release, Files: row.Files, Bytes: row.Bytes, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

// Delete drops one release's maps. Rolling a release back and forward should
// not leave a stale map claiming to explain the new build.
func (s *Service) Delete(ctx context.Context, app, release string) (int64, error) {
	result := s.orm.WithContext(ctx).
		Where("app = ? AND release = ?", app, release).
		Delete(&schemas.SourceMap{})
	if result.Error != nil {
		return 0, errors.Internal("failed to delete the source maps", result.Error)
	}
	s.evictRelease(app, release)
	return result.RowsAffected, nil
}

// consumerFor returns a parsed map, or nil when none is held. A miss is the
// normal case — most apps upload nothing — so it must be cheap and quiet.
func (s *Service) consumerFor(ctx context.Context, app, release, file string) *sourcemap.Consumer {
	key := app + "\x00" + release + "\x00" + file

	s.mu.Lock()
	if cached, ok := s.consumer[key]; ok {
		s.mu.Unlock()
		return cached
	}
	s.mu.Unlock()

	var row schemas.SourceMap
	err := s.orm.WithContext(ctx).
		Where("app = ? AND release = ? AND file = ?", app, release, file).
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

func (s *Service) evictRelease(app, release string) {
	prefix := app + "\x00" + release + "\x00"
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
