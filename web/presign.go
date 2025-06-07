package main

import (
	"context"
	"fmt"
	"time"

	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
)

// GeneratePresignedURL returns a presigned URL for the given S3 key using AppContext.
func GeneratePresignedURL(appCtx *AppContext, s3Key string) (string, error) {
	cfg := appCtx.Config
	if cfg == nil {
		return "", fmt.Errorf("missing config in AppContext")
	}
	if appCtx.S3Client == nil {
		return "", fmt.Errorf("missing S3 client in AppContext")
	}

	presignClient := s3sdk.NewPresignClient(appCtx.S3Client)
	presignParams := &s3sdk.PutObjectInput{
		Bucket: &cfg.S3UserUploadBucketName,
		Key:    &s3Key,
	}
	presignDuration := time.Duration(cfg.S3PresignDuration) * time.Second
	presignedReq, err := presignClient.PresignPutObject(context.Background(), presignParams, func(opts *s3sdk.PresignOptions) {
		opts.Expires = presignDuration
	})
	if err != nil {
		return "", err
	}

	return presignedReq.URL, nil
}
