package schemas

import "time"

// Ingest key kinds. A secret key is a server credential and authenticates
// POST /ingest; a public key is pasted into a JavaScript bundle and
// authenticates POST /ingest/browser only.
//
// The two are never interchangeable, and that is the whole point of the
// column: a public key is readable by anyone who opens devtools, so it must
// not be able to reach the endpoint that accepts a thousand entries per
// request from any origin with no quota.
const (
	KeyKindSecret = "secret"
	KeyKindPublic = "public"
)

type APIKey struct {
	ID      int64  `json:"id" gorm:"column:id;primaryKey"`
	App     string `json:"app" gorm:"column:app;not null"`
	Kind    string `json:"kind" gorm:"column:kind;not null;default:secret"`
	Prefix  string `json:"prefix" gorm:"column:prefix;not null"`
	KeyHash string `json:"-" gorm:"column:key_hash;uniqueIndex;not null"`

	// AllowedOrigins is the exact-match origin allowlist a public key is
	// checked against, and it is empty on a secret key. An empty list on a
	// public key rejects every request: there is no "unset means any" here,
	// because that reading turns a leaked key into an open relay.
	AllowedOrigins []string `json:"allowed_origins,omitempty" gorm:"column:allowed_origins;type:jsonb;serializer:json"`

	// DailyQuota caps entries accepted per UTC day. Zero means unlimited and
	// is only legal on a secret key.
	DailyQuota int `json:"daily_quota" gorm:"column:daily_quota;not null;default:0"`

	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	RevokedAt *time.Time `json:"revoked_at" gorm:"column:revoked_at"`
}

func (APIKey) TableName() string { return "api_keys" }
