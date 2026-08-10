package apikeys

type CreateRequest struct {
	App            string   `json:"app"`
	Kind           string   `json:"kind"`
	AllowedOrigins []string `json:"allowed_origins"`
	DailyQuota     int      `json:"daily_quota"`
}

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

type ListResponse struct {
	Keys []KeyResponse `json:"keys"`
}

type CreateResponse struct {
	Key   KeyResponse `json:"key"`
	Token string      `json:"token"`
}
