package mcp

import (
	"testing"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/mcp/mcptest"
)

// TestConformance runs tronc's conformance kit on the endpoint, with both
// deletions as the sensitive tools.
func TestConformance(t *testing.T) {
	f := newFixture(t)
	mcptest.Run(t, mcptest.Config{
		URL:   f.resource,
		Token: f.token,
		Confirm: []mcptest.ConfirmCase{
			{Tool: toolDeleteQuery, Setup: f.freshQuery},
			{Tool: toolDeleteAlert, Setup: f.freshAlert},
		},
	})
}

// freshQuery stores a saved query and reports 1 once it is gone.
func (f *fixture) freshQuery(tb testing.TB) (map[string]any, func() int) {
	q := f.query(tb, "query "+tb.Name())
	return map[string]any{"query_id": q.ID}, func() int { return 1 - f.count(&schemas.SavedQuery{}, q.ID) }
}

// freshAlert stores an alert rule and reports 1 once it is gone.
func (f *fixture) freshAlert(tb testing.TB) (map[string]any, func() int) {
	r := f.alert(tb, "alert "+tb.Name(), f.query(tb, "watched "+tb.Name()).ID)
	return map[string]any{"alert_id": r.ID}, func() int { return 1 - f.count(&schemas.AlertRule{}, r.ID) }
}
