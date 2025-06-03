package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type S3Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
}

type Config struct {
	DBFilePath              string
	S3                      S3Config
	LocalRawVideoPath       string
	LocalProcessedVideoPath string
	EncoderWorkerCount      int
	MaxEncodingFailures     int
	MaxCallbackFailures     int
}

func LoadConfig() Config {
	encoderWorkerCount := 2 // default
	if v := os.Getenv("WORKER_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			encoderWorkerCount = n
		}
	}

	dbFilePath := os.Getenv("DB_PATH")
	if dbFilePath == "" {
		log.Fatal("DB_PATH environment variable must be set")
	}

	maxCallbackFailures := 3 // default
	if v := os.Getenv("MAX_CALLBACK_FAILURES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxCallbackFailures = n
		}
	}

	maxEncodingFailures := 3 // default
	if v := os.Getenv("MAX_ENCODING_FAILURES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxEncodingFailures = n
		}
	}

	return Config{
		DBFilePath: dbFilePath,
		S3: S3Config{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          os.Getenv("S3_REGION"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
		},
		LocalRawVideoPath:       filepath.Join("app", "data", "tmp", "raw-videos"),
		LocalProcessedVideoPath: filepath.Join("app", "data", "tmp", "processed-videos"),
		EncoderWorkerCount:      encoderWorkerCount,
		MaxCallbackFailures:     maxCallbackFailures,
		MaxEncodingFailures:     maxEncodingFailures,
	}
}
