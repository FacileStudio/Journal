package antenne

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// handler handles HTTP requests for Antenne settings.
type handler struct {
	service *Service
	logger  *slog.Logger
}

// getSettings handles GET /api/settings/antenne.
func (h *handler) getSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, fromEnv, err := h.service.getSettings(ctx)
	if err != nil {
		h.logger.Error("antenne: failed to get settings", slog.Any("error", err))
		httpjson.WriteError(w, err)
		return
	}

	resp := PoolSettingsResponse{
		Settings:  *settings,
		Connected: h.service.isConnected(),
		FromEnv:   fromEnv,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("antenne: failed to encode response", slog.Any("error", err))
		httpjson.WriteError(w, err)
		return
	}
}

// updateSettings handles PUT /api/settings/antenne.
func (h *handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req UpdatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("antenne: failed to decode request", slog.Any("error", err))
		httpjson.WriteError(w, errors.Invalid("invalid request body"))
		return
	}

	settings, connectErr, err := h.service.updateSettings(ctx, &req)
	if err != nil {
		h.logger.Error("antenne: failed to update settings", slog.Any("error", err))
		httpjson.WriteError(w, err)
		return
	}

	resp := PoolSettingsResponse{
		Settings:  *settings,
		Connected: h.service.isConnected(),
	}
	if connectErr != "" {
		resp.ConnectError = connectErr
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("antenne: failed to encode response", slog.Any("error", err))
		httpjson.WriteError(w, err)
		return
	}
}