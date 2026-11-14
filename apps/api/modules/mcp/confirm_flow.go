package mcp

import (
	"context"

	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// request asks the person before a deletion: the input_required question
// when the client can ask, otherwise the confirm link.
func (t *toolset) request(ctx context.Context, req *sdk.CallToolRequest, kind string, id int64, question string) (*sdk.CallToolResult, deleteOutput, error) {
	result, outcome, err := t.confirm.Request(ctx, req, kind, target(id), question)
	if err != nil {
		return nil, deleteOutput{}, mcpkit.Failure(err, "")
	}
	if result != nil {
		return result, deleteOutput{}, nil
	}
	return nil, deleteOutput{Status: outcome.Status, ID: id, ConfirmURL: outcome.ConfirmURL, Message: outcome.Message}, nil
}

// resume handles the retry that carries the person's answer to a deletion.
func (t *toolset) resume(ctx context.Context, req *sdk.CallToolRequest, kind, noun string) (*sdk.CallToolResult, deleteOutput, error) {
	if _, err := t.callerOf(ctx, req); err != nil {
		return nil, deleteOutput{}, err
	}
	outcome, err := t.confirm.Resume(ctx, req, kind)
	if err != nil {
		return nil, deleteOutput{}, mcpkit.Failure(err, "Call the tool again to ask the user afresh.")
	}
	if outcome.Status == mcpkit.StatusDeclined {
		return nil, deleteOutput{Status: mcpkit.StatusDeclined, Message: "The user declined. Nothing happened, and the " + noun + " is kept."}, nil
	}
	id, _ := idOf(outcome.Action)
	return nil, deleteOutput{Status: statusDeleted, ID: id, Message: "The user confirmed and the " + noun + " is deleted."}, nil
}
