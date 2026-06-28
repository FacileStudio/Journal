package schemas

import "time"

type LogEntry struct {
	ID         int64          `json:"id" gorm:"column:id;primaryKey"`
	App        string         `json:"app" gorm:"column:app;index;not null"`
	Level      string         `json:"level" gorm:"column:level;index;default:info"`
	Message    string         `json:"message" gorm:"column:message;type:text"`
	Meta       map[string]any `json:"meta,omitempty" gorm:"column:meta;type:jsonb;serializer:json"`
	CreatedAt  time.Time      `json:"created_at" gorm:"column:created_at;index"`
	ReceivedAt time.Time      `json:"received_at" gorm:"column:received_at;autoCreateTime"`
}

func (LogEntry) TableName() string { return "log_entries" }
