package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/internal/errors"
	"github.com/FacileStudio/Journal/apps/api/internal/httpjson"
)

func RequireToken(expected string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
			if expected == "" || token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
				httpjson.WriteError(w, errors.Unauthorized("invalid ingest token"))
				return
			}
			next.ServeHTTP(w, request)
		})
	}
}
