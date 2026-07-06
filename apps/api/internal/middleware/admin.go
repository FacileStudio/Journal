package middleware

import (
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
)

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		identity, ok := authcontext.From(request.Context())
		if !ok || !identity.IsAdmin {
			httpjson.WriteError(w, errors.Forbidden("admin access required"))
			return
		}
		next.ServeHTTP(w, request)
	})
}
