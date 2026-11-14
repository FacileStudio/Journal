// Package plans reads the Journal entitlements a workspace's plan grants (its
// daily entry cap and its retention) from the forfait values Registre serves
// in the directory snapshot, and enforces the daily cap at ingest.
//
// Until a snapshot carrying entitlements arrives, and for every account the
// snapshot holds no Journal value for, the free defaults apply. That is the
// opposite of forfait's own fail-open default on purpose: an ingest quota that
// opens up while Registre is unreachable would let one account fill the disk.
// The instance workspaces (plan_exempt) are never limited.
package plans

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/forfait"

	"gorm.io/gorm"
)

// The entitlement keys, as Ardoise will name them, and the free defaults.
const (
	FeatureEntriesDaily  = "journal.entries_daily"
	FeatureRetentionDays = "journal.retention_days"
	DefaultEntriesDaily  = 10_000
	DefaultRetentionDays = 7
	cacheFor             = time.Minute
	syncEvery            = 5 * time.Minute
)

// Limits is what one workspace's plan allows. Exempt workspaces are not
// limited at all; otherwise -1 means unlimited.
type Limits struct {
	Exempt        bool
	EntriesDaily  int64
	RetentionDays int64
}

// Defaults is the free plan.
func Defaults() Limits {
	return Limits{EntriesDaily: DefaultEntriesDaily, RetentionDays: DefaultRetentionDays}
}

type cached struct {
	limits Limits
	until  time.Time
}

// Service resolves workspace limits and charges the daily cap.
type Service struct {
	orm           *gorm.DB
	issuer        string
	store         *forfait.Store
	authoritative atomic.Bool
	now           func() time.Time

	mu    sync.Mutex
	cache map[int64]cached
}

// New builds the plan service. issuer is the OIDC issuer the accounts'
// Registre identities are filed under.
func New(orm *gorm.DB, issuer string, logger *slog.Logger) *Service {
	store := forfait.NewStore(
		forfait.Feature{Key: FeatureEntriesDaily, Kind: forfait.Limit, Default: json.RawMessage(strconv.Itoa(DefaultEntriesDaily))},
		forfait.Feature{Key: FeatureRetentionDays, Kind: forfait.Limit, Default: json.RawMessage(strconv.Itoa(DefaultRetentionDays))},
	)
	store.Issuer, store.Logger = issuer, logger
	return &Service{orm: orm, issuer: strings.TrimSuffix(issuer, "/"), store: store, now: time.Now, cache: map[int64]cached{}}
}

// Apply feeds one directory snapshot to the entitlement store.
func (s *Service) Apply(snap directory.Snapshot) {
	s.store.Apply(snap)
	if snap.EntitlementsRevision > 0 {
		s.authoritative.Store(true)
	}
	s.mu.Lock()
	s.cache = map[int64]cached{}
	s.mu.Unlock()
}

// Run pulls the directory snapshot at once and then every five minutes,
// until ctx ends.
func (s *Service) Run(ctx context.Context, client directory.Client, logger *slog.Logger) {
	ticker := time.NewTicker(syncEvery)
	defer ticker.Stop()
	for {
		snap, err := client.Snapshot(ctx)
		switch {
		case err == nil:
			s.Apply(snap)
		case ctx.Err() == nil:
			logger.Warn("plans: directory snapshot failed, keeping the last limits", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
