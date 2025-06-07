package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type InitializeVideoResponse struct {
	Success   bool   `json:"success"`
	VideoID   string `json:"videoId"`
	UploadURL string `json:"uploadUrl"`
	Error     string `json:"error,omitempty"`
}

func InitializeVideoHandler(c *gin.Context) {
	appCtx := GetAppContext(c)
	if appCtx == nil {
		log.Printf("AppContext missing in handler")
		c.JSON(http.StatusInternalServerError, InitializeVideoResponse{
			Success:   false,
			VideoID:   "",
			UploadURL: "",
			Error:     "Internal server error",
		})
		return
	}

	// Artificial delay for demonstration/testing
	time.Sleep(1 * time.Second)

	videoID := uuid.New().String()
	s3Key := "videos/" + videoID

	log.Printf("Initializing new video: id=%s, s3Key=%s", videoID, s3Key)

	video := Video{
		ID:          videoID,
		S3Key:       s3Key,
		Title:       "",
		Description: "",
		Tags:        []string{},
		Visibility:  VisibilityPrivate,
		Resolutions: []int{},
		Status:      StatusPending,
	}
	if err := appCtx.DB.Create(&video).Error; err != nil {
		log.Printf("Failed to create video record: %v", err)
		c.JSON(http.StatusInternalServerError, InitializeVideoResponse{
			Success:   false,
			VideoID:   videoID,
			UploadURL: "",
			Error:     "Failed to create video record",
		})
		return
	}

	uploadURL, err := GeneratePresignedURL(appCtx, s3Key)
	if err != nil {
		log.Printf("Failed to generate presigned URL for s3Key=%s: %v", s3Key, err)
		c.JSON(http.StatusInternalServerError, InitializeVideoResponse{
			Success:   false,
			VideoID:   videoID,
			UploadURL: "",
			Error:     "Failed to generate presigned URL",
		})
		return
	}

	log.Printf("Successfully initialized video: id=%s, uploadURL=%s", videoID, uploadURL)

	c.JSON(http.StatusOK, InitializeVideoResponse{
		Success:   true,
		VideoID:   videoID,
		UploadURL: uploadURL,
	})
}
