package auth

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Journal/apps/api/internal/authcontext"
	"github.com/FacileStudio/Journal/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// Handler serves the auth endpoints.
type Handler struct {
	service     *Service
	clearCookie func(http.ResponseWriter, *http.Request)
}

func newHandler(service *Service, clearCookie func(http.ResponseWriter, *http.Request)) *Handler {
	return &Handler{service: service, clearCookie: clearCookie}
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	user, token, err := h.service.Register(r.Context(), w, r, req.Email, req.Name, req.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, AuthResponse{Token: token, User: toUserResponse(*user)})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	user, token, err := h.service.Login(r.Context(), w, r, req.Email, req.Password)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, AuthResponse{Token: token, User: toUserResponse(*user)})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.From(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("not authenticated"))
		return
	}

	user, err := h.service.UserByID(r.Context(), identity.UserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, MeResponse{User: toUserResponse(*user)})
}

func (h *Handler) deleteMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.From(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("not authenticated"))
		return
	}

	if err := h.service.DeleteAccount(r.Context(), identity.UserID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	h.clearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func toUserResponse(user schemas.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		IsAdmin:   user.IsAdmin,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
	}
}
