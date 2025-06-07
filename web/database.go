package main

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDatabase() error {
	cfg := LoadConfig()
	DataSourceName := "host=" + cfg.DatabaseHost +
		" user=" + cfg.DatabaseUser +
		" password=" + cfg.DatabasePassword +
		" dbname=" + cfg.DatabaseName +
		" port=" + cfg.DatabasePort +
		" sslmode=" + cfg.DatabaseSSLMode +
		" TimeZone=" + cfg.DatabaseTimeZone
	db, err := gorm.Open(postgres.Open(DataSourceName), &gorm.Config{})
	if err != nil {
		return err
	}
	DB = db
	return nil
}

type VideoVisibility string

type VideoStatus string

const (
	VisibilityPrivate  VideoVisibility = "private"
	VisibilityUnlisted VideoVisibility = "unlisted"
	VisibilityPublic   VideoVisibility = "public"

	StatusPending VideoStatus = "pending"
	StatusSuccess VideoStatus = "success"
	StatusFailure VideoStatus = "failure"
)

type Video struct {
	ID          string          `json:"id" gorm:"primaryKey"`
	S3Key       string          `json:"s3Key"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Tags        []string        `json:"tags" gorm:"type:text[]"`
	Visibility  VideoVisibility `json:"visibility" gorm:"type:video_visibility;default:'private'"`
	Resolutions []int           `json:"resolutions" gorm:"type:int[]"`
	Status      VideoStatus     `json:"status" gorm:"type:video_status;default:'pending'"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}
