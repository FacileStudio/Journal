package schemas

// AppSetting is the singleton row of app-level configuration.
type AppSetting struct {
	ID             int    `gorm:"primaryKey"`
	AntenneURL     string `gorm:"column:antenne_url;not null;default:''" json:"antenne_url"`
	AntenneSecret  string `gorm:"column:antenne_secret;not null;default:''" json:"-"`
	AntenneEnabled bool   `gorm:"column:antenne_enabled;not null;default:false" json:"antenne_enabled"`
}

func (AppSetting) TableName() string { return "app_settings" }