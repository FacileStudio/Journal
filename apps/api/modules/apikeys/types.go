package apikeys

type CreateRequest struct {
	App string `json:"app"`
}

type KeyResponse struct {
	ID        int64   `json:"id"`
	App       string  `json:"app"`
	Prefix    string  `json:"prefix"`
	CreatedAt string  `json:"created_at"`
	RevokedAt *string `json:"revoked_at"`
}

type ListResponse struct {
	Keys []KeyResponse `json:"keys"`
}

type CreateResponse struct {
	Key   KeyResponse `json:"key"`
	Token string      `json:"token"`
}
