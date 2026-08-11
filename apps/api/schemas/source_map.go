package schemas

import "time"

// SourceMap is one uploaded map, keyed by the bundle file it explains.
//
// File is the *basename* of the generated file — "BxYz1234.js", not a path or
// a URL. A stack frame arrives as a full URL whose origin and prefix depend on
// how the app is served, and Vite hashes every filename, so the basename is
// both stable across deployments and unique within a release. Matching on
// anything longer means a map that was uploaded correctly still fails to
// resolve.
//
// Content is the map's JSON, stored as text rather than on disk: it keeps the
// one-datastore property, rides along with whatever backs up the database, and
// avoids giving the API a writable volume it otherwise does not need.
type SourceMap struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey"`
	App       string    `json:"app" gorm:"column:app;not null;index:idx_source_maps_lookup,unique,priority:1"`
	Release   string    `json:"release" gorm:"column:release;not null;index:idx_source_maps_lookup,unique,priority:2"`
	File      string    `json:"file" gorm:"column:file;not null;index:idx_source_maps_lookup,unique,priority:3"`
	Content   string    `json:"-" gorm:"column:content;type:text;not null"`
	Bytes     int       `json:"bytes" gorm:"column:bytes;not null;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (SourceMap) TableName() string { return "source_maps" }
