package main

import (
	"testing"
)

func TestCreateJobsInDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	payload := &RequestPayload{
		VideoId: "vid123",
		Input: struct {
			Key    string
			Bucket string
		}{
			Key:    "input.mp4",
			Bucket: "input-bucket",
		},
		Output: struct {
			BasePath string
			Bucket   string
		}{
			BasePath: "outputs/",
			Bucket:   "output-bucket",
		},
		CallbackURL: "http://callback",
		Profiles: []struct {
			Resolution string
			Crf        int
		}{
			{Resolution: "720", Crf: 23},
			{Resolution: "1080", Crf: 20},
		},
	}

	err := CreateJobsInDB(db, payload)
	if err != nil {
		t.Fatalf("CreateJobsInDB failed: %v", err)
	}

	jobs, err := GetPendingJobs(db)
	if err != nil {
		t.Fatalf("GetPendingJobs failed: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
	for _, job := range jobs {
		if job.VideoID != "vid123" {
			t.Errorf("Expected VideoID 'vid123', got %s", job.VideoID)
		}
		if job.InputKey != "input.mp4" {
			t.Errorf("Expected InputKey 'input.mp4', got %s", job.InputKey)
		}
		if job.OutputBucket != "output-bucket" {
			t.Errorf("Expected OutputBucket 'output-bucket', got %s", job.OutputBucket)
		}
		if job.CallbackURL != "http://callback" {
			t.Errorf("Expected CallbackURL 'http://callback', got %s", job.CallbackURL)
		}
		if job.Resolution != 720 && job.Resolution != 1080 {
			t.Errorf("Expected Resolution 720 or 1080, got %d", job.Resolution)
		}
	}
}
