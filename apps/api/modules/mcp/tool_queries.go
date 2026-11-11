package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var errQueryMissing = errors.NotFound("saved query not found")

// listQueries is journal_list_saved_queries.
func (t *toolset) listQueries(ctx context.Context, req *sdk.CallToolRequest, _ queriesInput) (*sdk.CallToolResult, queriesOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, queriesOutput{}, err
	}
	records, err := t.Queries.List(ctx)
	if err != nil {
		return nil, queriesOutput{}, mcpkit.Failure(err, "")
	}
	out := queriesOutput{Queries: make([]queryOut, len(records))}
	for i, record := range records {
		out.Queries[i] = queryView(record)
	}
	return nil, out, nil
}

// saveQuery is journal_save_query.
func (t *toolset) saveQuery(ctx context.Context, req *sdk.CallToolRequest, in saveQueryInput) (*sdk.CallToolResult, queryOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, queryOutput{}, err
	}
	if in.Source != "" && in.Source != "server" && in.Source != "browser" {
		return nil, queryOutput{}, errSource
	}
	params := schemas.SavedQueryParams{
		App: strings.TrimSpace(in.App), Levels: in.Levels, Q: strings.TrimSpace(in.Q), Source: in.Source,
		RequestID: strings.TrimSpace(in.RequestID),
	}
	record, err := t.Queries.Create(ctx, in.Name, params)
	if err != nil {
		return nil, queryOutput{}, mcpkit.Failure(err, "")
	}
	return nil, queryOutput{Status: "created", Query: queryView(*record)}, nil
}

// queryView renders a saved query.
func queryView(q schemas.SavedQuery) queryOut {
	return queryOut{
		ID: q.ID, Name: q.Name, App: q.Params.App, Levels: q.Params.Levels, Q: q.Params.Q, Source: q.Params.Source,
		RequestID: q.Params.RequestID, CreatedAt: stamp(q.CreatedAt),
	}
}

// savedQuery finds one saved query by id.
func (t *toolset) savedQuery(ctx context.Context, id int64) (schemas.SavedQuery, error) {
	records, err := t.Queries.List(ctx)
	if err != nil {
		return schemas.SavedQuery{}, err
	}
	for _, record := range records {
		if record.ID == id {
			return record, nil
		}
	}
	return schemas.SavedQuery{}, errQueryMissing
}

// watchers counts the alert rules that watch a saved query, which keep it
// from being deleted.
func (t *toolset) watchers(ctx context.Context, queryID int64) (int, error) {
	rules, err := t.Alerts.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, rule := range rules {
		if rule.SavedQueryID == queryID {
			n++
		}
	}
	return n, nil
}

// deleteQuery is journal_delete_saved_query: the query goes once the user
// confirms.
func (t *toolset) deleteQuery(ctx context.Context, req *sdk.CallToolRequest, in queryRef) (*sdk.CallToolResult, deleteOutput, error) {
	if mcpkit.Retrying(req) {
		return t.resume(ctx, req, kindDeleteQuery, "saved query")
	}
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, deleteOutput{}, err
	}
	query, err := t.savedQuery(ctx, in.QueryID)
	if err != nil {
		return nil, deleteOutput{}, mcpkit.Failure(err, hintQuery)
	}
	n, err := t.watchers(ctx, query.ID)
	if err != nil {
		return nil, deleteOutput{}, mcpkit.Failure(err, "")
	}
	if n > 0 {
		return nil, deleteOutput{}, fmt.Errorf("%d alert rule(s) watch this saved query. Delete or move them first; an administrator lists them with journal_list_alerts", n)
	}
	question := fmt.Sprintf("Delete the saved query %q for everyone in Journal? This cannot be undone.", query.Name)
	return t.request(ctx, req, kindDeleteQuery, query.ID, question)
}
