package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// JobStatus type and constants
type JobStatus int

const (
	JobStatusEncodingPending    JobStatus = 0
	JobStatusEncodingRunning    JobStatus = 1
	JobStatusEncodingFailed     JobStatus = 2
	JobStatusEncodingSuccess    JobStatus = 3
	JobStatusCallbackPending    JobStatus = 4
	JobStatusCallbackInProgress JobStatus = 5
	JobStatusCallbackFailed     JobStatus = 6
	JobStatusCallbackSuccess    JobStatus = 7
)

// Struct for job
type Job struct {
	ID               string
	VideoID          string
	InputKey         string
	InputBucket      string
	OutputPath       string
	OutputBucket     string
	Resolution       int
	Crf              int
	CallbackURL      string
	CreatedAt        string
	UpdatedAt        string
	Status           JobStatus
	FailedCount      int
	CallbackFailures int // new field for tracking callback failures
}

// Initialize DB and schema
func InitDB(filepath string) *sql.DB {
	log.Printf("Initializing database at %s", filepath)
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		video_id TEXT NOT NULL,
		input_key TEXT NOT NULL,
		input_bucket TEXT NOT NULL,
		output_path TEXT NOT NULL,
		output_bucket TEXT NOT NULL,
		resolution INTEGER,
		crf INTEGER,
		callback_url TEXT NOT NULL,
		status INTEGER NOT NULL,
		failed_count INTEGER DEFAULT 0,
		callback_failures INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	return db
}

// Insert new job
func InsertJob(db *sql.DB, job Job) error {
	_, err := db.Exec(`INSERT INTO jobs (id, video_id, input_key, input_bucket, output_path, output_bucket, resolution, crf, callback_url, status, failed_count, callback_failures) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.VideoID, job.InputKey, job.InputBucket, job.OutputPath, job.OutputBucket, job.Resolution, job.Crf, job.CallbackURL, int(job.Status), job.FailedCount, job.CallbackFailures)
	return err
}

// Update job status
func UpdateJobStatus(db *sql.DB, jobID string, status JobStatus) error {
	_, err := db.Exec(`UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, int(status), jobID)
	return err
}

// Update job failed count
func IncrementJobFailedCount(db *sql.DB, jobID string) error {
	_, err := db.Exec(`UPDATE jobs SET failed_count = failed_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, jobID)
	return err
}

// Fetch pending jobs (for worker loop)
func GetPendingJobs(db *sql.DB) ([]Job, error) {
	rows, err := db.Query(`SELECT id, video_id, input_key, input_bucket, output_path, output_bucket, resolution, crf, callback_url, status, failed_count FROM jobs WHERE status = ?`, int(JobStatusEncodingPending))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		var status int
		err := rows.Scan(&job.ID, &job.VideoID, &job.InputKey, &job.InputBucket, &job.OutputPath, &job.OutputBucket, &job.Resolution, &job.Crf, &job.CallbackURL, &status, &job.FailedCount)
		if err != nil {
			return nil, err
		}
		job.Status = JobStatus(status)
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Fetch jobs with status 'pending' or 'failed'
func GetPendingOrFailedJobs(db *sql.DB) ([]Job, error) {
	log.Printf("Fetching jobs with status 'pending' or 'failed' from database...")
	rows, err := db.Query(`SELECT id, video_id, input_key, input_bucket, output_path, output_bucket, resolution, crf, callback_url, status, failed_count FROM jobs WHERE status = ? OR status = ?`, int(JobStatusEncodingPending), int(JobStatusEncodingFailed))
	if err != nil {
		log.Printf("Error querying jobs: %v", err)
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		var status int
		err := rows.Scan(&job.ID, &job.VideoID, &job.InputKey, &job.InputBucket, &job.OutputPath, &job.OutputBucket, &job.Resolution, &job.Crf, &job.CallbackURL, &status, &job.FailedCount)
		if err != nil {
			log.Printf("Error scanning job row: %v", err)
			return nil, err
		}
		job.Status = JobStatus(status)
		jobs = append(jobs, job)
	}
	log.Printf("Fetched %d jobs with status 'pending' or 'failed'", len(jobs))
	return jobs, nil
}

// Atomically claim a job (set to in_progress if still pending/failed)
func ClaimJob(db *sql.DB, jobID string) (bool, error) {
	res, err := db.Exec(`UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND (status = ? OR status = ?)`, int(JobStatusEncodingPending+1), jobID, int(JobStatusEncodingPending), int(JobStatusEncodingFailed))
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// Reset jobs stuck in 'in_progress' state on startup
func ResetInProgressJobs(db *sql.DB) error {
	// Set jobs with failed_count > 0 to 'failed', others to 'pending'
	_, err := db.Exec(`
		UPDATE jobs
		SET status = CASE WHEN failed_count > 0 THEN ? ELSE ? END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE status = ?
	`, int(JobStatusEncodingFailed), int(JobStatusEncodingPending), int(JobStatusEncodingRunning))
	return err
}

// Fetch jobs with status 'callback pending'
func GetCallbackPendingJobs(db *sql.DB) ([]Job, error) {
	rows, err := db.Query(`SELECT id, video_id, input_key, input_bucket, output_path, output_bucket, resolution, crf, callback_url, status, failed_count, callback_failures FROM jobs WHERE status = ?`, int(JobStatusCallbackPending))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var job Job
		var status int
		err := rows.Scan(&job.ID, &job.VideoID, &job.InputKey, &job.InputBucket, &job.OutputPath, &job.OutputBucket, &job.Resolution, &job.Crf, &job.CallbackURL, &status, &job.FailedCount, &job.CallbackFailures)
		if err != nil {
			return nil, err
		}
		job.Status = JobStatus(status)
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Increment callback_failures for a job
func IncrementCallbackFailures(db *sql.DB, jobID string) error {
	_, err := db.Exec(`UPDATE jobs SET callback_failures = callback_failures + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, jobID)
	return err
}
