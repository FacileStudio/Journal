package schemas

import "time"

type SavedQueryParams struct {
	App       string   `json:"app,omitempty"`
	Levels    []string `json:"levels,omitempty"`
	Q         string   `json:"q,omitempty"`
	RequestID string   `json:"request_id,omitempty"`
}

type SavedQuery struct {
	ID        int64            `json:"id" gorm:"column:id;primaryKey"`
	Name      string           `json:"name" gorm:"column:name;uniqueIndex;not null"`
	Params    SavedQueryParams `json:"params" gorm:"column:params;type:jsonb;serializer:json;not null"`
	CreatedAt time.Time        `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (SavedQuery) TableName() string { return "saved_queries" }
