package job

import (
	"time"
)

// JobType represents the type of job to execute
type JobType string

const (
	JobTypeCommand JobType = "command"
	JobTypeScript  JobType = "script"
	JobTypeHTTP    JobType = "http"
	JobTypeFile    JobType = "file"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
	JobStatusRetrying  JobStatus = "retrying"
)

// Job represents a job to be executed
type Job struct {
	ID          string            `json:"id"`
	Type        JobType           `json:"type"`
	Command     string            `json:"command,omitempty"`
	Script      string            `json:"script,omitempty"`
	URL         string            `json:"url,omitempty"`
	Method      string            `json:"method,omitempty"`
	FilePath    string            `json:"file_path,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	Retries     int               `json:"retries"`
	Priority    int               `json:"priority"`
	Tags        []string          `json:"tags,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkerID    string            `json:"worker_id,omitempty"`
	Status      JobStatus         `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Output      string            `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	ExitCode    int               `json:"exit_code,omitempty"`
}

// JobResult represents the result of a job execution
type JobResult struct {
	JobID       string        `json:"job_id"`
	Status      JobStatus     `json:"status"`
	Output      string        `json:"output"`
	Error       string        `json:"error"`
	ExitCode    int           `json:"exit_code"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Duration    time.Duration `json:"duration"`
}

// JobRequest represents a request to create a new job
type JobRequest struct {
	Type        JobType           `json:"type"`
	Command     string            `json:"command,omitempty"`
	Script      string            `json:"script,omitempty"`
	URL         string            `json:"url,omitempty"`
	Method      string            `json:"method,omitempty"`
	FilePath    string            `json:"file_path,omitempty"`
	Timeout     string            `json:"timeout,omitempty"` // Will be parsed to time.Duration
	Retries     int               `json:"retries,omitempty"`
	Priority    int               `json:"priority,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// Validate validates a job request
func (jr *JobRequest) Validate() error {
	if jr.Type == "" {
		return NewValidationError("job type is required")
	}

	switch jr.Type {
	case JobTypeCommand:
		if jr.Command == "" {
			return NewValidationError("command is required for command jobs")
		}
	case JobTypeScript:
		if jr.Script == "" {
			return NewValidationError("script is required for script jobs")
		}
	case JobTypeHTTP:
		if jr.URL == "" {
			return NewValidationError("url is required for HTTP jobs")
		}
		if jr.Method == "" {
			jr.Method = "GET" // Default method
		}
	case JobTypeFile:
		if jr.FilePath == "" {
			return NewValidationError("file_path is required for file jobs")
		}
	default:
		return NewValidationError("unsupported job type: " + string(jr.Type))
	}

	return nil
}

// ToJob converts a JobRequest to a Job with generated ID and timestamps
func (jr *JobRequest) ToJob() (*Job, error) {
	if err := jr.Validate(); err != nil {
		return nil, err
	}

	job := &Job{
		ID:          GenerateJobID(),
		Type:        jr.Type,
		Command:     jr.Command,
		Script:      jr.Script,
		URL:         jr.URL,
		Method:      jr.Method,
		FilePath:    jr.FilePath,
		Retries:     jr.Retries,
		Priority:    jr.Priority,
		Tags:        jr.Tags,
		Environment: jr.Environment,
		Status:      JobStatusPending,
		CreatedAt:   time.Now(),
	}

	// Parse timeout
	if jr.Timeout != "" {
		timeout, err := time.ParseDuration(jr.Timeout)
		if err != nil {
			return nil, NewValidationError("invalid timeout format: " + jr.Timeout)
		}
		job.Timeout = timeout
	} else {
		job.Timeout = 5 * time.Minute // Default timeout
	}

	// Set default priority if not specified
	if job.Priority == 0 {
		job.Priority = 1
	}

	return job, nil
} 