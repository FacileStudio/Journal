package queries

import "github.com/FacileStudio/Journal/apps/api/schemas"

type CreateRequest struct {
	Name   string                   `json:"name"`
	Params schemas.SavedQueryParams `json:"params"`
}

type QueryResponse struct {
	ID        int64                    `json:"id"`
	Name      string                   `json:"name"`
	Params    schemas.SavedQueryParams `json:"params"`
	CreatedAt string                   `json:"created_at"`
}

type ListResponse struct {
	Queries []QueryResponse `json:"queries"`
}

type CreateResponse struct {
	Query QueryResponse `json:"query"`
}
