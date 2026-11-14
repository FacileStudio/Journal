package plans

import (
	"context"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"
)

// Limits returns what workspaceID's plan allows. A personal workspace reads
// its owner's personal entitlements, the most generous over the owner's
// Registre identities; any other workspace that is not exempt reads the free
// defaults.
func (s *Service) Limits(ctx context.Context, workspaceID int64) (Limits, error) {
	if hit, ok := s.cached(workspaceID); ok {
		return hit, nil
	}
	var ws schemas.Workspace
	if err := s.orm.WithContext(ctx).Where("id = ?", workspaceID).Take(&ws).Error; err != nil {
		return Limits{}, errors.Internal("failed to load the workspace plan", err)
	}
	limits := Defaults()
	switch {
	case ws.PlanExempt:
		limits = Limits{Exempt: true, EntriesDaily: -1, RetentionDays: -1}
	case ws.PersonalOf != nil && s.authoritative.Load():
		subjects, err := s.subjects(ctx, *ws.PersonalOf)
		if err != nil {
			return Limits{}, err
		}
		limits = s.best(subjects)
	}
	s.remember(workspaceID, limits)
	return limits, nil
}

func (s *Service) subjects(ctx context.Context, userID int64) ([]string, error) {
	var subjects []string
	err := s.orm.WithContext(ctx).Table("porte_identities").
		Where("user_id = ? AND rtrim(provider, '/') = ?", userID, s.issuer).
		Pluck("subject", &subjects).Error
	if err != nil {
		return nil, errors.Internal("failed to read the account's identities", err)
	}
	return subjects, nil
}

func (s *Service) best(subjects []string) Limits {
	if len(subjects) == 0 {
		return Defaults()
	}
	var limits Limits
	for _, subject := range subjects {
		scope := forfait.Scope{Subject: strings.TrimSpace(subject)}
		limits.EntriesDaily = generous(limits.EntriesDaily, s.store.Limit(scope, FeatureEntriesDaily))
		limits.RetentionDays = generous(limits.RetentionDays, s.store.Limit(scope, FeatureRetentionDays))
	}
	return limits
}

func generous(a, b int64) int64 {
	if a == -1 || b == -1 {
		return -1
	}
	return max(a, b)
}

func (s *Service) cached(workspaceID int64) (Limits, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hit, ok := s.cache[workspaceID]
	if !ok || s.now().After(hit.until) {
		return Limits{}, false
	}
	return hit.limits, true
}

func (s *Service) remember(workspaceID int64, limits Limits) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache[workspaceID] = cached{limits: limits, until: s.now().Add(cacheFor)}
}
