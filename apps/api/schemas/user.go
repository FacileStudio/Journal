package schemas

import "time"

type User struct {
	ID           int64     `json:"id" gorm:"column:id;primaryKey"`
	Email        string    `json:"email" gorm:"column:email;uniqueIndex;not null"`
	Name         string    `json:"name" gorm:"column:name"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	IsAdmin      bool      `json:"is_admin" gorm:"column:is_admin;default:false"`
	CreatedAt    time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }
