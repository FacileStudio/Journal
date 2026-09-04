package schemas

import "time"

// AntenneOutbox is a pooled event queued for later delivery.
type AntenneOutbox struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Channel   string    `gorm:"column:channel"`
	Payload   string    `gorm:"column:payload"`
	Attempts  int       `gorm:"column:attempts;not null;default:0"`
	LastError string    `gorm:"column:last_error"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AntenneOutbox) TableName() string { return "antenne_outbox" }

// AntenneProcessedEvent records a pooled event's idempotency key once it has
// been handled.
type AntenneProcessedEvent struct {
	IdempotencyKey string    `gorm:"column:idempotency_key;primaryKey"`
	ProcessedAt    time.Time `gorm:"column:processed_at;autoCreateTime"`
}

func (AntenneProcessedEvent) TableName() string { return "antenne_processed_events" }
