package mcp

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/porte/kit"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const hintWorkspace = "Call journal_list_workspaces for the ids of your workspaces."

var errNotOwner = stderrors.New("alert rules are for workspace owners only, and this account owns no workspace here. " +
	"Saved queries and logs stay available")

func (t *toolset) accessFor(ctx context.Context, account kit.Identity, selected int64) (workspace.Access, error) {
	raw := ""
	if selected > 0 {
		raw = strconv.FormatInt(selected, 10)
	}
	access, err := t.Workspaces.Access(ctx, account, raw)
	if err != nil {
		return workspace.Access{}, mcpkit.Failure(err, hintWorkspace)
	}
	return access, nil
}

func (t *toolset) accessOf(ctx context.Context, req *sdk.CallToolRequest, selected int64) (workspace.Access, error) {
	account, err := t.callerOf(ctx, req)
	if err != nil {
		return workspace.Access{}, err
	}
	return t.accessFor(ctx, account, selected)
}

func (t *toolset) ownerOf(ctx context.Context, req *sdk.CallToolRequest, selected int64) (workspace.Access, error) {
	access, err := t.accessOf(ctx, req, selected)
	if err != nil {
		return workspace.Access{}, err
	}
	if access.Owned().Empty() {
		return workspace.Access{}, errNotOwner
	}
	return access, nil
}

func (t *toolset) accessOfUser(ctx context.Context, userID string) (workspace.Access, error) {
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return workspace.Access{}, errNoCaller
	}
	account, err := t.Accounts.Identity(ctx, id)
	if err != nil {
		return workspace.Access{}, errNoCaller
	}
	return t.accessFor(ctx, account, 0)
}

func (t *toolset) listWorkspaces(ctx context.Context, req *sdk.CallToolRequest, _ workspacesInput) (*sdk.CallToolResult, workspacesOutput, error) {
	access, err := t.accessOf(ctx, req, 0)
	if err != nil {
		return nil, workspacesOutput{}, err
	}
	out := workspacesOutput{Workspaces: make([]workspaceOut, len(access.Memberships))}
	for i, m := range access.Memberships {
		out.Workspaces[i] = workspaceOut{ID: m.ID, Name: m.Name, Role: m.Role, Personal: m.Personal}
	}
	return nil, out, nil
}
