package schemas

import "time"

// SavedQueryParams is the filter a saved query captures.
type SavedQueryParams struct {
	App       string   `json:"app,omitempty"`
	Levels    []string `json:"levels,omitempty"`
	Q         string   `json:"q,omitempty"`
	Source    string   `json:"source,omitempty"`
	RequestID string   `json:"request_id,omitempty"`
}

// SavedQuery is a named, reusable log filter.
type SavedQuery struct {
	ID          int64            `json:"id" gorm:"column:id;primaryKey"`
	WorkspaceID int64            `json:"workspace_id" gorm:"column:workspace_id;not null;uniqueIndex:idx_saved_queries_workspace_name,priority:1"`
	Name        string           `json:"name" gorm:"column:name;not null;uniqueIndex:idx_saved_queries_workspace_name,priority:2"`
	Params      SavedQueryParams `json:"params" gorm:"column:params;type:jsonb;serializer:json;not null"`
	CreatedAt   time.Time        `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (SavedQuery) TableName() string { return "saved_queries" }
