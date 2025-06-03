package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Profile struct {
	Resolution string `json:"resolution"`
	Crf        int    `json:"crf"`
}

type Input struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

type Output struct {
	Bucket   string `json:"bucket"`
	BasePath string `json:"basePath"`
}

type RequestPayload struct {
	VideoId     string    `json:"videoId"`
	Input       Input     `json:"input"`
	Output      Output    `json:"output"`
	Profiles    []Profile `json:"profiles"`
	CallbackURL string    `json:"callbackUrl"`
}

type OutputResult struct {
	Resolution string `json:"resolution"`
	Key        string `json:"key,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

type ResponsePayload struct {
	Status  string         `json:"status"`
	Outputs []OutputResult `json:"outputs"`
}

type AppContext struct {
	Config   Config
	DB       *sql.DB
	S3Client *s3.Client
}

func ensureDirectoryExistence(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		os.MkdirAll(dirPath, 0755)
		log.Printf("Directory created at %s\n", dirPath)
	}
}

func ProcessVideoHandler(ctx *AppContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqPayload RequestPayload
		var respPayload ResponsePayload
		respPayload.Status = "error"

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&reqPayload); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			respPayload.Status = "error"
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON: " + err.Error()})
			return
		}

		// Validate required fields
		if reqPayload.Input.Bucket == "" || reqPayload.Input.Key == "" ||
			reqPayload.Output.Bucket == "" || reqPayload.Output.BasePath == "" ||
			len(reqPayload.Profiles) == 0 || reqPayload.CallbackURL == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			respPayload.Status = "error"
			json.NewEncoder(w).Encode(map[string]string{"error": "Missing required fields (including callbackUrl)"})
			return
		}

		// Validate callbackUrl format
		parsedUrl, err := url.ParseRequestURI(reqPayload.CallbackURL)
		if err != nil || parsedUrl.Scheme == "" || parsedUrl.Host == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			respPayload.Status = "error"
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid callbackUrl format"})
			return
		}

		// Create the job(s) in SQLite, passing callback URL
		if err := CreateJobsInDB(ctx.DB, &reqPayload); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			respPayload.Status = "error"
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create job: " + err.Error()})
			return
		}

		// Return success indicating job accepted (no jobId)
		respPayload.Status = "accepted"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respPayload)
	}
}

func StartWorkerPool(ctx *AppContext) {
	for i := 0; i < ctx.Config.EncoderWorkerCount; i++ {
		go func(workerID int) {
			log.Printf("Worker %d started", workerID)
			for {
				jobs, err := GetPendingOrFailedJobs(ctx.DB)
				if err != nil {
					log.Printf("Worker %d: error fetching jobs: %v", workerID, err)
					continue
				}
				claimed := false
				for _, job := range jobs {
					ok, err := ClaimJob(ctx.DB, job.ID)
					if err != nil {
						log.Printf("Worker %d: error claiming job %s: %v", workerID, job.ID, err)
						continue
					}
					if ok {
						log.Printf("Worker %d: claimed job %s", workerID, job.ID)
						ProcessVideoJob(ctx, &job)
						claimed = true
						break // Only process one job per loop per worker
					}
				}
				if !claimed {
					// No jobs claimed, sleep before next poll
					time.Sleep(2 * time.Second)
				}
			}
		}(i + 1)
	}
}

// StartCallbackWorkerPool starts workers for callback jobs
func StartCallbackWorkerPool(ctx *AppContext) {
	for i := 0; i < 1; i++ { // one callback worker should be more than enough, add more if needed
		go func(workerID int) {
			log.Printf("Callback Worker %d started", workerID)
			for {
				jobs, err := GetCallbackPendingJobs(ctx.DB)
				if err != nil {
					log.Printf("Callback Worker %d: error fetching jobs: %v", workerID, err)
					continue
				}
				claimed := false
				for _, job := range jobs {
					if job.CallbackFailures >= ctx.Config.MaxCallbackFailures {
						UpdateJobStatus(ctx.DB, job.ID, JobStatusCallbackFailed)
						continue
					}
					ok, err := ClaimCallbackJob(ctx.DB, job.ID)
					if err != nil {
						log.Printf("Callback Worker %d: error claiming job %s: %v", workerID, job.ID, err)
						continue
					}
					if ok {
						log.Printf("Callback Worker %d: claimed callback job %s", workerID, job.ID)
						ProcessCallbackJob(ctx, &job)
						claimed = true
						break // Only process one job per loop per worker
					}
				}
				if !claimed {
					// No jobs claimed, sleep before next poll
					time.Sleep(2 * time.Second)
				}
			}
		}(i + 1)
	}
}

// ClaimCallbackJob atomically claims a callback job
func ClaimCallbackJob(db *sql.DB, jobID string) (bool, error) {
	res, err := db.Exec(`UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = ?`, int(JobStatusCallbackInProgress), jobID, int(JobStatusCallbackPending))
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// ProcessCallbackJob attempts the callback and updates job state
func ProcessCallbackJob(ctx *AppContext, job *Job) {
	// Example: POST to callback URL (add real payload as needed)
	resp, err := http.Post(job.CallbackURL, "application/json", nil)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		UpdateJobStatus(ctx.DB, job.ID, JobStatusCallbackSuccess)
		return
	}
	IncrementCallbackFailures(ctx.DB, job.ID)
	if job.CallbackFailures+1 >= ctx.Config.MaxCallbackFailures {
		UpdateJobStatus(ctx.DB, job.ID, JobStatusCallbackFailed)
	} else {
		UpdateJobStatus(ctx.DB, job.ID, JobStatusCallbackPending)
	}
}

func StartServer(ctx *AppContext) {
	ensureDirectoryExistence(ctx.Config.LocalRawVideoPath)
	ensureDirectoryExistence(ctx.Config.LocalProcessedVideoPath)

	http.HandleFunc("/process-video", ProcessVideoHandler(ctx))
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Server running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func main() {
	cfg := LoadConfig()

	// Start the database
	db := InitDB(cfg.DBFilePath)
	defer db.Close()

	// Reset jobs stuck in 'in_progress' state
	err := ResetInProgressJobs(db)
	if err != nil {
		log.Fatalf("Failed to reset in-progress jobs: %v", err)
	}

	// Initialize S3 client
	s3Client, err := NewS3Client(cfg.S3)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// Create app context
	ctx := &AppContext{
		Config:   cfg,
		DB:       db,
		S3Client: s3Client,
	}
	StartWorkerPool(ctx)
	StartCallbackWorkerPool(ctx)
	StartServer(ctx)
}
