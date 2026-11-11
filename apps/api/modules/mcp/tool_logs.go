package mcp

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Journal/apps/api/modules/logs"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultPage     = 50
	maxPage         = 200
	maxNeighbours   = 50
	maxCountWindow  = 90 * 24 * time.Hour
	emptyPageHint   = "No entry matches. Widen since (such as 7d), drop a filter, or check the app name with journal_list_apps."
	trimmedPageHint = "The page was cut to stay under the result size limit; pass next_cursor for the rest, or narrow the filter."
)

var errCountWindow = stderrors.New("journal_count_logs covers at most 90 days at a time; move since closer to until")

// cursor is where the next page of a search starts: before this entry.
type cursor struct {
	Ts string `json:"ts"`
	ID int64  `json:"id"`
}

// search is journal_search_logs.
func (t *toolset) search(ctx context.Context, req *sdk.CallToolRequest, in searchInput) (*sdk.CallToolResult, searchOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, searchOutput{}, err
	}
	wide, err := mcpkit.Detailed(in.ResponseFormat)
	if err != nil {
		return nil, searchOutput{}, err
	}
	params, err := t.pageParams(in)
	if err != nil {
		return nil, searchOutput{}, err
	}
	records, err := t.Logs.List(ctx, params)
	if err != nil {
		return nil, searchOutput{}, mcpkit.Failure(err, "")
	}
	look := concise
	if wide {
		look = detailed
	}
	return nil, page(records, params, look), nil
}

// pageParams turns a search input into the explorer's list parameters.
func (t *toolset) pageParams(in searchInput) (logs.ListParams, error) {
	filter, err := filterOf(in.filterInput, time.Now())
	if err != nil {
		return logs.ListParams{}, err
	}
	params := logs.ListParams{Params: filter, Limit: min(max(in.Limit, 0), maxPage)}
	if params.Limit == 0 {
		params.Limit = defaultPage
	}
	var at cursor
	if err := mcpkit.DecodeCursor(in.Cursor, &at); err != nil {
		return logs.ListParams{}, err
	}
	if in.Cursor != "" {
		ts, err := time.Parse(time.RFC3339Nano, at.Ts)
		if err != nil {
			return logs.ListParams{}, mcpkit.ErrCursor
		}
		params.BeforeTs, params.BeforeID = &ts, &at.ID
	}
	return params, nil
}

// page renders one page of entries under the result budget, with the cursor
// of the next one when the list was full or had to be cut.
func page(records []schemas.LogEntry, params logs.ListParams, look shape) searchOutput {
	out := searchOutput{Since: stamp(*params.Since), Until: stamp(*params.Until), Entries: make([]entryOut, len(records))}
	for i, record := range records {
		out.Entries[i] = render(record, look)
	}
	all := out.Entries
	n := mcpkit.FitCount(len(all), func(n int) int { out.Entries = all[:n]; return mcpkit.EncodedSize(out) })
	out.Entries = all[:max(n, min(1, len(all)))]
	kept := len(out.Entries)
	switch {
	case len(all) == 0:
		out.Note = emptyPageHint
	case kept < len(all):
		out.Note = trimmedPageHint
		out.NextCursor = next(records[kept-1])
	case len(records) == params.Limit:
		out.NextCursor = next(records[kept-1])
	}
	return out
}

// next is the cursor of the page after entry.
func next(entry schemas.LogEntry) string {
	return mcpkit.EncodeCursor(cursor{Ts: entry.CreatedAt.UTC().Format(time.RFC3339Nano), ID: entry.ID})
}

// read is journal_read_log.
func (t *toolset) read(ctx context.Context, req *sdk.CallToolRequest, in readInput) (*sdk.CallToolResult, readOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, readOutput{}, err
	}
	entry, err := t.Logs.Entry(ctx, in.EntryID)
	if err != nil {
		return nil, readOutput{}, mcpkit.Failure(err, hintEntry)
	}
	out := readOutput{Entry: render(entry, full), ReceivedAt: stamp(entry.ReceivedAt), SessionID: metaText(entry.Meta, "session_id")}
	before, after := min(max(in.Before, 0), maxNeighbours), min(max(in.After, 0), maxNeighbours)
	if before+after > 0 {
		around, err := t.Logs.Context(ctx, entry.ID, before, after)
		if err != nil {
			return nil, readOutput{}, mcpkit.Failure(err, hintEntry)
		}
		out.Newer, out.Older = neighbours(around, entry.ID)
	}
	if in.ResolveStack && t.Stacks != nil {
		stack := t.Stacks.Resolve(ctx, entry.App, metaText(entry.Meta, "release"), metaText(entry.Meta, "stack"))
		out.Stack = &stack
	}
	return nil, fit(out), nil
}

// neighbours splits a context page, newest first, into the entries logged
// after the anchor, nearest first, and those before it, nearest first.
func neighbours(around []schemas.LogEntry, anchorID int64) ([]entryOut, []entryOut) {
	var newer, older []entryOut
	seen := false
	for _, e := range around {
		switch {
		case e.ID == anchorID:
			seen = true
		case seen:
			older = append(older, render(e, concise))
		default:
			newer = append([]entryOut{render(e, concise)}, newer...)
		}
	}
	return newer, older
}

// fit drops the farthest neighbours until the result is under the budget.
func fit(out readOutput) readOutput {
	for mcpkit.EncodedSize(out) > mcpkit.ResultBudget && len(out.Newer)+len(out.Older) > 0 {
		if len(out.Older) >= len(out.Newer) {
			out.Older = out.Older[:len(out.Older)-1]
		} else {
			out.Newer = out.Newer[:len(out.Newer)-1]
		}
	}
	return out
}
