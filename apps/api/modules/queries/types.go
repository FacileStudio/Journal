package queries

import "github.com/FacileStudio/Journal/apps/api/schemas"

// CreateRequest is the body of a saved query creation.
type CreateRequest struct {
	Name   string                   `json:"name"`
	Params schemas.SavedQueryParams `json:"params"`
}

// QueryResponse describes one saved query for the client.
type QueryResponse struct {
	ID        int64                    `json:"id"`
	Name      string                   `json:"name"`
	Params    schemas.SavedQueryParams `json:"params"`
	CreatedAt string                   `json:"created_at"`
}

// ListResponse is the list of every saved query.
type ListResponse struct {
	Queries []QueryResponse `json:"queries"`
}

// CreateResponse carries the query created by a CreateRequest.
type CreateResponse struct {
	Query QueryResponse `json:"query"`
}
