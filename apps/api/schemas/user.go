package schemas

import "time"

// User is the local identity row: credentials plus the OIDC source that
// created or last synced the account.
type User struct {
	ID           int64     `json:"id" gorm:"column:id;primaryKey"`
	Email        string    `json:"email" gorm:"column:email;uniqueIndex;not null"`
	Name         string    `json:"name" gorm:"column:name"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	IsAdmin      bool      `json:"is_admin" gorm:"column:is_admin;default:false"`
	AvatarURL    string    `json:"avatar_url" gorm:"column:avatar_url"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }
