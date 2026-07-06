package schemas

import "time"

type APIKey struct {
	ID        int64      `json:"id" gorm:"column:id;primaryKey"`
	App       string     `json:"app" gorm:"column:app;not null"`
	Prefix    string     `json:"prefix" gorm:"column:prefix;not null"`
	KeyHash   string     `json:"-" gorm:"column:key_hash;uniqueIndex;not null"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	RevokedAt *time.Time `json:"revoked_at" gorm:"column:revoked_at"`
}

func (APIKey) TableName() string { return "api_keys" }
