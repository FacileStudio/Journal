package schemas

import "time"

// AlertProviderType is the type of delivery target for an alert.
type AlertProviderType string

const (
	// AlertProviderWebhook delivers via HTTP POST.
	AlertProviderWebhook AlertProviderType = "webhook"
	// AlertProviderAntenne delivers via Antenne WebSocket.
	AlertProviderAntenne AlertProviderType = "antenne"
)

// AlertRule is one configured alert: a saved query, a threshold, a provider
// and its destination to hit when the count crosses it.
type AlertRule struct {
	ID            int64             `json:"id" gorm:"column:id;primaryKey"`
	Name          string            `json:"name" gorm:"column:name;not null"`
	SavedQueryID  int64             `json:"saved_query_id" gorm:"column:saved_query_id;not null"`
	Provider      AlertProviderType `json:"provider" gorm:"column:provider;type:text;not null;default:'webhook'"`
	Threshold     int               `json:"threshold" gorm:"column:threshold;not null"`
	WindowMinutes int               `json:"window_minutes" gorm:"column:window_minutes;not null"`
	WebhookURL    string            `json:"webhook_url" gorm:"column:webhook_url;type:text;not null"`
	WebhookHeader *string           `json:"webhook_header" gorm:"column:webhook_header;type:text"`
	WebhookSecret *string           `json:"-" gorm:"column:webhook_secret;type:text"`
	Enabled       bool              `json:"enabled" gorm:"column:enabled;not null;default:true"`
	LastFiredAt   *time.Time        `json:"last_fired_at" gorm:"column:last_fired_at"`
	CreatedAt     time.Time         `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (AlertRule) TableName() string { return "alert_rules" }
