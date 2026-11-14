package workspaces

import (
	"net/http"

	"github.com/FacileStudio/Journal/apps/api/internal/workspace"
	"github.com/FacileStudio/tronc/httpjson"
)

// ListResponse is every workspace the caller belongs to.
type ListResponse struct {
	Workspaces []workspace.Membership `json:"workspaces"`
}

func list(w http.ResponseWriter, r *http.Request) {
	memberships := workspace.FromContext(r.Context()).Memberships
	if memberships == nil {
		memberships = []workspace.Membership{}
	}
	httpjson.WriteJSON(w, http.StatusOK, ListResponse{Workspaces: memberships})
}
