// Package workspace is Journal's tenancy: who belongs to which workspace, and
// the Scope every read of tenant data filters on.
package workspace

import (
	"slices"

	"gorm.io/gorm"
)

// Scope is the set of workspaces one request may read. The zero Scope holds
// none, so a query built from a forgotten Scope matches nothing.
type Scope struct {
	ids []int64
}

// Of builds a Scope over ids, ignoring non-positive ones and duplicates.
func Of(ids ...int64) Scope {
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return Scope{ids: out}
}

// IDs returns a copy of the workspace ids in the scope, in ascending order.
func (s Scope) IDs() []int64 {
	return slices.Clone(s.ids)
}

// Empty reports whether the scope holds no workspace.
func (s Scope) Empty() bool {
	return len(s.ids) == 0
}

// Contains reports whether id is one of the scope's workspaces.
func (s Scope) Contains(id int64) bool {
	return slices.Contains(s.ids, id)
}

// Where narrows query to rows whose workspace_id column is in the scope. An
// empty scope narrows it to nothing. column defaults to workspace_id; a join
// passes the qualified name.
func (s Scope) Where(query *gorm.DB, column ...string) *gorm.DB {
	name := "workspace_id"
	if len(column) > 0 {
		name = column[0]
	}
	if s.Empty() {
		return query.Where("1 = 0")
	}
	return query.Where(name+" IN ?", s.ids)
}
