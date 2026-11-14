package workspace

import (
	"context"

	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

var (
	errUnknown  = errors.NotFound("workspace not found")
	errNotOwner = errors.Forbidden("only an owner of this workspace can do this")
	errNoTarget = errors.Invalid("workspace_id is required: you belong to several workspaces")
)

// Access is what one authenticated request may do: the workspaces the caller
// belongs to, and the Scope of the ones this request reads.
type Access struct {
	UserID      int64
	Memberships []Membership
	Scope       Scope
}

type accessKey struct{}

// WithAccess stores access in ctx.
func WithAccess(ctx context.Context, access Access) context.Context {
	return context.WithValue(ctx, accessKey{}, access)
}

// FromContext returns the Access the middleware stored, or the zero Access,
// whose empty Scope reads nothing.
func FromContext(ctx context.Context) Access {
	access, _ := ctx.Value(accessKey{}).(Access)
	return access
}

// ScopeFrom is FromContext(ctx).Scope.
func ScopeFrom(ctx context.Context) Scope {
	return FromContext(ctx).Scope
}

func (a Access) role(id int64) (string, bool) {
	for _, m := range a.Memberships {
		if m.ID == id {
			return m.Role, true
		}
	}
	return "", false
}

// All is the Scope of every workspace the caller belongs to.
func (a Access) All() Scope {
	ids := make([]int64, 0, len(a.Memberships))
	for _, m := range a.Memberships {
		ids = append(ids, m.ID)
	}
	return Of(ids...)
}

// Owned is the part of the request's Scope the caller owns.
func (a Access) Owned() Scope {
	ids := make([]int64, 0, len(a.Memberships))
	for _, m := range a.Memberships {
		if m.Role == schemas.RoleOwner && a.Scope.Contains(m.ID) {
			ids = append(ids, m.ID)
		}
	}
	return Of(ids...)
}

// Select narrows the caller's workspaces to requested, answering 404 for any
// id the caller does not belong to, so a probe cannot tell a foreign
// workspace from a missing one. No id selects every workspace.
func (a Access) Select(requested []int64) (Scope, error) {
	if len(requested) == 0 {
		return a.All(), nil
	}
	for _, id := range requested {
		if _, ok := a.role(id); !ok {
			return Scope{}, errUnknown
		}
	}
	return Of(requested...), nil
}

// Target picks the one workspace a write goes to. requested 0 means the
// caller's only eligible workspace; with several, the caller must name one.
// owner demands the owner role.
func (a Access) Target(requested int64, owner bool) (int64, error) {
	if requested != 0 {
		role, ok := a.role(requested)
		if !ok {
			return 0, errUnknown
		}
		if owner && role != schemas.RoleOwner {
			return 0, errNotOwner
		}
		return requested, nil
	}
	var eligible []int64
	for _, m := range a.Memberships {
		if a.Scope.Contains(m.ID) && (!owner || m.Role == schemas.RoleOwner) {
			eligible = append(eligible, m.ID)
		}
	}
	switch len(eligible) {
	case 0:
		return 0, errNotOwner
	case 1:
		return eligible[0], nil
	default:
		return 0, errNoTarget
	}
}
