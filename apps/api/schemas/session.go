package schemas

import "time"

type Session struct {
	Token     string    `json:"-" gorm:"column:token;primaryKey"`
	UserID    int64     `json:"user_id" gorm:"column:user_id;index;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"column:expires_at;index"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Session) TableName() string { return "sessions" }
