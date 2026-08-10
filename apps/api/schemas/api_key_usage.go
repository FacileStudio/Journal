package schemas

import "time"

// APIKeyUsage counts the entries one key accepted on one UTC day.
//
// It exists for the browser endpoint, where the credential is public by
// construction: rate limiting bounds the burst from one IP, and this bounds
// the total from everyone. The row is the enforcement point, not a report —
// the counter is incremented and read in the same statement before anything
// is written to log_entries.
type APIKeyUsage struct {
	APIKeyID int64     `json:"api_key_id" gorm:"column:api_key_id;primaryKey"`
	Day      time.Time `json:"day" gorm:"column:day;type:date;primaryKey"`
	Count    int64     `json:"count" gorm:"column:count;not null;default:0"`
}

func (APIKeyUsage) TableName() string { return "api_key_usage" }
