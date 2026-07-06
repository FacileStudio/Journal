package alerts

import (
	"testing"
	"time"

	"github.com/FacileStudio/Journal/apps/api/schemas"
)

func TestShouldFire(t *testing.T) {
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	fired := func(ago time.Duration) *time.Time {
		ts := now.Add(-ago)
		return &ts
	}
	rule := func(enabled bool, threshold, windowMinutes int, lastFiredAt *time.Time) schemas.AlertRule {
		return schemas.AlertRule{Enabled: enabled, Threshold: threshold, WindowMinutes: windowMinutes, LastFiredAt: lastFiredAt}
	}

	cases := []struct {
		name  string
		rule  schemas.AlertRule
		count int64
		want  bool
	}{
		{"never fired above threshold", rule(true, 5, 15, nil), 5, true},
		{"never fired below threshold", rule(true, 5, 15, nil), 4, false},
		{"zero count", rule(true, 1, 15, nil), 0, false},
		{"disabled never fires", rule(false, 1, 15, nil), 100, false},
		{"fired within window stays armed off", rule(true, 5, 15, fired(5*time.Minute)), 100, false},
		{"fired just under a window ago", rule(true, 5, 15, fired(15*time.Minute-time.Second)), 100, false},
		{"re-arms at exactly one window", rule(true, 5, 15, fired(15*time.Minute)), 100, true},
		{"re-arms after window passed", rule(true, 5, 15, fired(16*time.Minute)), 100, true},
		{"re-armed but below threshold", rule(true, 5, 15, fired(time.Hour)), 4, false},
		{"large window still cooling", rule(true, 1, 1440, fired(23*time.Hour)), 10, false},
		{"large window re-armed", rule(true, 1, 1440, fired(25*time.Hour)), 10, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldFire(tc.rule, now, tc.count); got != tc.want {
				t.Fatalf("shouldFire(count=%d) = %v, want %v", tc.count, got, tc.want)
			}
		})
	}
}
