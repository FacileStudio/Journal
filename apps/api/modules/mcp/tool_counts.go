package mcp

import (
	"context"
	"time"

	"github.com/FacileStudio/Journal/apps/api/modules/logs"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// count is journal_count_logs.
func (t *toolset) count(ctx context.Context, req *sdk.CallToolRequest, in countInput) (*sdk.CallToolResult, countOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, countOutput{}, err
	}
	filter, err := filterOf(in.filterInput, time.Now())
	if err != nil {
		return nil, countOutput{}, err
	}
	if filter.Until.Sub(*filter.Since) > maxCountWindow {
		return nil, countOutput{}, errCountWindow
	}
	width := logs.BucketSeconds(int64(filter.Until.Sub(*filter.Since) / time.Second))
	buckets, err := t.Logs.Histogram(ctx, logs.ListParams{Params: filter}, width)
	if err != nil {
		return nil, countOutput{}, mcpkit.Failure(err, "")
	}
	out := countOutput{
		Since: stamp(*filter.Since), Until: stamp(*filter.Until), BucketSeconds: width,
		ByLevel: map[string]int64{}, Buckets: make([]bucketOut, len(buckets)),
	}
	for i, bucket := range buckets {
		out.Buckets[i] = bucketOut{Start: bucket.Ts, Counts: bucket.Counts}
		for level, n := range bucket.Counts {
			out.ByLevel[level] += n
			out.Total += n
		}
	}
	return nil, out, nil
}

// apps is journal_list_apps.
func (t *toolset) apps(ctx context.Context, req *sdk.CallToolRequest, _ appsInput) (*sdk.CallToolResult, appsOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, appsOutput{}, err
	}
	summaries, err := t.Logs.Apps(ctx)
	if err != nil {
		return nil, appsOutput{}, mcpkit.Failure(err, "")
	}
	out := appsOutput{Apps: make([]appOut, len(summaries))}
	for i, s := range summaries {
		out.Apps[i] = appOut{Name: s.Name, Entries: s.Count, LastSeen: s.LastSeen}
	}
	return nil, out, nil
}
