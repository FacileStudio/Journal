package workspace

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// QueryParam is the query parameter a request names its workspaces in: a
// comma-separated list of ids, or nothing for every workspace of the caller.
const QueryParam = "workspace"

// Middleware runs after porte's RequireAuth. It loads the caller's
// workspaces, creating a personal one on the first request of a new account,
// and stores the Access the handlers read. A request naming a workspace the
// caller does not belong to answers 404.
func Middleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := kit.FromContext(r.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("not authenticated"))
				return
			}
			access, err := service.Access(r.Context(), identity, r.URL.Query().Get(QueryParam))
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithAccess(r.Context(), access)))
		})
	}
}

// Access resolves what identity may read, narrowed to the comma-separated
// workspace ids in selection.
func (s *Service) Access(ctx context.Context, identity kit.Identity, selection string) (Access, error) {
	requested, err := ParseIDs(selection)
	if err != nil {
		return Access{}, err
	}
	memberships, err := s.Memberships(ctx, identity.UserID, identity.Name)
	if err != nil {
		return Access{}, err
	}
	access := Access{UserID: identity.UserID, Memberships: memberships}
	if access.Scope, err = access.Select(requested); err != nil {
		return Access{}, err
	}
	return access, nil
}

// ParseIDs reads a comma-separated list of workspace ids. "all" and an empty
// string both mean no selection.
func ParseIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "all" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > 100 {
		return nil, errors.Invalid("at most 100 workspaces at a time")
	}
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			return nil, errors.Invalid("workspace must be a comma-separated list of workspace ids")
		}
		ids = append(ids, id)
	}
	return ids, nil
}
