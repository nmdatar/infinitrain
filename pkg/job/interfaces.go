package job

import (
	"context"
)

// Executor defines the interface for executing jobs
type Executor interface {
	// Execute runs a job and returns the result
	Execute(ctx context.Context, job *Job) (*JobResult, error)
	
	// CanExecute checks if this executor can handle the given job type
	CanExecute(jobType JobType) bool
	
	// Name returns the name of this executor
	Name() string
}

// Queue defines the interface for job queue operations
type Queue interface {
	// Enqueue adds a job to the queue
	Enqueue(ctx context.Context, job *Job) error
	
	// Dequeue removes and returns the next job from the queue
	Dequeue(ctx context.Context) (*Job, error)
	
	// Peek returns the next job without removing it from the queue
	Peek(ctx context.Context) (*Job, error)
	
	// Size returns the number of jobs in the queue
	Size(ctx context.Context) (int, error)
	
	// IsEmpty returns true if the queue is empty
	IsEmpty(ctx context.Context) (bool, error)
}

// Store defines the interface for job storage and retrieval
type Store interface {
	// Create stores a new job
	Create(ctx context.Context, job *Job) error
	
	// Get retrieves a job by ID
	Get(ctx context.Context, jobID string) (*Job, error)
	
	// Update updates an existing job
	Update(ctx context.Context, job *Job) error
	
	// Delete removes a job from storage
	Delete(ctx context.Context, jobID string) error
	
	// List returns jobs with optional filtering
	List(ctx context.Context, filters ...Filter) ([]*Job, error)
	
	// UpdateStatus updates the status of a job
	UpdateStatus(ctx context.Context, jobID string, status JobStatus) error
}

// Scheduler defines the interface for job scheduling
type Scheduler interface {
	// Schedule schedules a job for execution
	Schedule(ctx context.Context, job *Job) error
	
	// Cancel cancels a scheduled job
	Cancel(ctx context.Context, jobID string) error
	
	// GetNextJob returns the next job to be executed
	GetNextJob(ctx context.Context) (*Job, error)
	
	// MarkCompleted marks a job as completed
	MarkCompleted(ctx context.Context, jobID string, result *JobResult) error
	
	// MarkFailed marks a job as failed
	MarkFailed(ctx context.Context, jobID string, err error) error
}

// Worker defines the interface for worker nodes
type Worker interface {
	// ID returns the unique identifier for this worker
	ID() string
	
	// Start starts the worker
	Start(ctx context.Context) error
	
	// Stop stops the worker gracefully
	Stop(ctx context.Context) error
	
	// IsHealthy returns true if the worker is healthy
	IsHealthy() bool
	
	// GetCapacity returns the maximum number of concurrent jobs this worker can handle
	GetCapacity() int
	
	// GetCurrentLoad returns the current number of jobs being executed
	GetCurrentLoad() int
	
	// CanAcceptJob returns true if the worker can accept a new job
	CanAcceptJob() bool
}

// WorkerRegistry defines the interface for managing workers
type WorkerRegistry interface {
	// Register adds a worker to the registry
	Register(ctx context.Context, worker Worker) error
	
	// Unregister removes a worker from the registry
	Unregister(ctx context.Context, workerID string) error
	
	// GetWorker returns a worker by ID
	GetWorker(ctx context.Context, workerID string) (Worker, error)
	
	// ListWorkers returns all registered workers
	ListWorkers(ctx context.Context) ([]Worker, error)
	
	// GetAvailableWorkers returns workers that can accept new jobs
	GetAvailableWorkers(ctx context.Context) ([]Worker, error)
	
	// Heartbeat updates the last seen time for a worker
	Heartbeat(ctx context.Context, workerID string) error
}

// Filter defines filtering criteria for job queries
type Filter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, in, contains
	Value    interface{} `json:"value"`
}

// JobManager combines all job-related operations
type JobManager interface {
	// Submit submits a new job
	Submit(ctx context.Context, request *JobRequest) (*Job, error)
	
	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, jobID string) (*Job, error)
	
	// ListJobs lists jobs with optional filtering
	ListJobs(ctx context.Context, filters ...Filter) ([]*Job, error)
	
	// CancelJob cancels a running or pending job
	CancelJob(ctx context.Context, jobID string) error
	
	// GetJobResult gets the result of a completed job
	GetJobResult(ctx context.Context, jobID string) (*JobResult, error)
} 