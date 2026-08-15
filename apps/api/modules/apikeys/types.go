package apikeys

// CreateRequest is the body of an API key creation.
type CreateRequest struct {
	App            string   `json:"app"`
	Kind           string   `json:"kind"`
	AllowedOrigins []string `json:"allowed_origins"`
	DailyQuota     int      `json:"daily_quota"`
}

// KeyResponse describes one key for the client, without the secret.
type KeyResponse struct {
	ID             int64    `json:"id"`
	App            string   `json:"app"`
	Kind           string   `json:"kind"`
	Prefix         string   `json:"prefix"`
	AllowedOrigins []string `json:"allowed_origins"`
	DailyQuota     int      `json:"daily_quota"`
	UsedToday      int64    `json:"used_today"`
	CreatedAt      string   `json:"created_at"`
	RevokedAt      *string  `json:"revoked_at"`
}

// ListResponse is the list of every key.
type ListResponse struct {
	Keys []KeyResponse `json:"keys"`
}

// CreateResponse carries the key plus the one-time secret token, shown only
// at creation.
type CreateResponse struct {
	Key   KeyResponse `json:"key"`
	Token string      `json:"token"`
}
