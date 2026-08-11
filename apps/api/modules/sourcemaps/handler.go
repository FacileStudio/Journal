package sourcemaps

import (
	"net/http"
	"strings"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// maxMapBytes caps one upload. A map for a large chunk runs to a few megabytes;
// well past that is a mistake, and the point of a cap is to answer it with a
// 413 rather than an out-of-memory.
const maxMapBytes = 24 << 20

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler { return &Handler{service: service} }

// scopedApp returns the app the credential is scoped to.
//
// The legacy INGEST_TOKEN is unscoped, and an unscoped credential must not be
// able to write maps for an app it cannot name: it would let one shipper's
// token overwrite another app's resolution. Uploading needs a per-app key.
func scopedApp(r *http.Request) (string, error) {
	scope, _ := authcontext.IngestScopeFrom(r.Context())
	if scope.App == "" {
		return "", errors.Forbidden("uploading source maps needs a per-app API key, not the shared ingest token")
	}
	return scope.App, nil
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	app, err := scopedApp(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	var req UploadRequest
	if err := httpjson.DecodeJSONLimit(w, r, &req, maxMapBytes); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if strings.TrimSpace(req.Map) == "" {
		httpjson.WriteError(w, errors.Invalid("map is required"))
		return
	}

	stored, err := h.service.Store(r.Context(), app, req.Release, req.File, req.Map)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, UploadResponse{Stored: stored})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	app, err := scopedApp(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	release := strings.TrimSpace(r.URL.Query().Get("release"))
	if release == "" {
		httpjson.WriteError(w, errors.Invalid("release is required"))
		return
	}

	files, err := h.service.Files(r.Context(), app, release)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Release: release, Files: files})
}

func (h *Handler) releases(w http.ResponseWriter, r *http.Request) {
	releases, err := h.service.Releases(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, ReleasesResponse{Releases: releases})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	app := strings.TrimSpace(r.URL.Query().Get("app"))
	release := strings.TrimSpace(r.URL.Query().Get("release"))
	if app == "" || release == "" {
		httpjson.WriteError(w, errors.Invalid("app and release are required"))
		return
	}

	if _, err := h.service.Delete(r.Context(), app, release); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusNoContent, nil)
}
