// filepath: /workspaces/ScalrCMS-/video-processing-service/src/processor.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

// ConvertVideo converts a video to 360p using ffmpeg.
// rawVideoName: the input file path
// processedVideoName: the output file path
func ConvertVideo(rawVideoName string, processedVideoName string, resolution int, crf int) error {
	fmt.Printf("Converting video from %s to %s\n", rawVideoName, processedVideoName)

	cmd := exec.Command(
		"ffmpeg",
		"-i", rawVideoName,
		"-vf", fmt.Sprintf("scale=-2:%d", resolution),
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", fmt.Sprintf("%d", crf),
		processedVideoName,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	fmt.Printf("Spawned FFMPEG with command: %v\n", cmd.Args)

	// Print ffmpeg stderr output
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				fmt.Printf("FFmpeg stderr: %s", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("an error occurred: %w", err)
	}

	fmt.Println("Processing finished successfully")
	return nil
}

// ProcessVideoJob processes a video job (now takes Job struct)
func ProcessVideoJob(ctx *AppContext, job *Job) {
	inputFilePath := filepath.Join(ctx.Config.LocalRawVideoPath, filepath.Base(job.InputKey))
	outputBasePath := filepath.Join(ctx.Config.LocalProcessedVideoPath, filepath.Base(job.OutputPath))

	// Ensure output directory exists before processing
	if err := os.MkdirAll(outputBasePath, 0755); err != nil {
		log.Printf("Failed to create output directory: %v", err)
		UpdateJobStatus(ctx.DB, job.ID, JobStatusEncodingFailed)
		return
	}

	// Download the input file (Bucket is now stored in Job)
	if err := DownloadFile(context.Background(), ctx.S3Client, job.InputBucket, job.InputKey, inputFilePath); err != nil {
		log.Printf("Failed to download file: %v", err)
		UpdateJobStatus(ctx.DB, job.ID, JobStatusEncodingFailed)
		IncrementJobFailedCount(ctx.DB, job.ID)
		return
	}

	outputFileName := fmt.Sprintf("%dp.mp4", job.Resolution)
	outputFilePath := filepath.Join(outputBasePath, outputFileName)
	outputKey := filepath.Join(job.OutputPath, outputFileName)

	err := ConvertVideo(inputFilePath, outputFilePath, job.Resolution, job.Crf)
	if err == nil {
		err = UploadFile(context.Background(), ctx.S3Client, job.OutputBucket, outputFilePath, outputKey)
	}

	if err == nil {
		UpdateJobStatus(ctx.DB, job.ID, JobStatusEncodingSuccess)
	} else {
		log.Printf("Failed to process video for job %s: %v", job.ID, err)
		UpdateJobStatus(ctx.DB, job.ID, JobStatusEncodingFailed)
		IncrementJobFailedCount(ctx.DB, job.ID)
	}

	// Cleanup
	if err := os.Remove(inputFilePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove input file: %v", err)
	}
	if err := os.RemoveAll(outputBasePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove output directory: %v", err)
	}
}

// CreateJobsInDB inserts new jobs into the database for each profile in the request payload
func CreateJobsInDB(db *sql.DB, reqPayload *RequestPayload) error {
	for _, profile := range reqPayload.Profiles {
		job := Job{
			ID:               uuid.New().String(),
			VideoID:          reqPayload.VideoId,
			InputKey:         reqPayload.Input.Key,
			InputBucket:      reqPayload.Input.Bucket,
			OutputPath:       reqPayload.Output.BasePath,
			OutputBucket:     reqPayload.Output.Bucket,
			Resolution:       0, // Default resolution, will be set later if valid
			Crf:              profile.Crf,
			CallbackURL:      reqPayload.CallbackURL,
			Status:           JobStatusEncodingPending,
			FailedCount:      0,
			CallbackFailures: 0,
		}
		if res, err := strconv.Atoi(profile.Resolution); err == nil {
			job.Resolution = res
		}
		if err := InsertJob(db, job); err != nil {
			return err
		}
	}
	return nil
}
