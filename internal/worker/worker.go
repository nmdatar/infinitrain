package worker

import (
	"context"
	"fmt"
	"infinitrain/internal/config"
	"infinitrain/pkg/job"
	"sync"
	"time"
)

// Worker represents a worker node that can execute jobs
type Worker struct {
	id             string
	config         *config.WorkerConfig
	executor       job.Executor
	currentJobs    map[string]*job.Job
	currentJobsMux sync.RWMutex
	isRunning      bool
	isHealthy      bool
	lastHeartbeat  time.Time
	heartbeatMux   sync.RWMutex
}

// NewWorker creates a new worker instance
func NewWorker(cfg *config.WorkerConfig, executor job.Executor) *Worker {
	return &Worker{
		id:            cfg.ID,
		config:        cfg,
		executor:      executor,
		currentJobs:   make(map[string]*job.Job),
		isHealthy:     true,
		lastHeartbeat: time.Now(),
	}
}

// ID returns the unique identifier for this worker
func (w *Worker) ID() string {
	return w.id
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	w.isRunning = true

	// Create working directory if it doesn't exist
	if err := w.ensureWorkingDirectory(); err != nil {
		return fmt.Errorf("failed to create working directory: %v", err)
	}

	fmt.Printf("Worker %s started\n", w.id)

	// Start heartbeat routine
	go w.heartbeatLoop(ctx)

	// Start job polling routine
	go w.jobPollingLoop(ctx)

	return nil
}

// Stop stops the worker gracefully
func (w *Worker) Stop(ctx context.Context) error {
	w.isRunning = false

	// Wait for current jobs to complete or timeout
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			fmt.Printf("Worker %s stopped with timeout, cancelling remaining jobs\n", w.id)
			return nil
		case <-ticker.C:
			if w.GetCurrentLoad() == 0 {
				fmt.Printf("Worker %s stopped gracefully\n", w.id)
				return nil
			}
		case <-ctx.Done():
			fmt.Printf("Worker %s stopped due to context cancellation\n", w.id)
			return ctx.Err()
		}
	}
}

// IsHealthy returns true if the worker is healthy
func (w *Worker) IsHealthy() bool {
	w.heartbeatMux.RLock()
	defer w.heartbeatMux.RUnlock()
	return w.isHealthy && w.isRunning
}

// GetCapacity returns the maximum number of concurrent jobs this worker can handle
func (w *Worker) GetCapacity() int {
	return w.config.MaxConcurrentJobs
}

// GetCurrentLoad returns the current number of jobs being executed
func (w *Worker) GetCurrentLoad() int {
	w.currentJobsMux.RLock()
	defer w.currentJobsMux.RUnlock()
	return len(w.currentJobs)
}

// CanAcceptJob returns true if the worker can accept a new job
func (w *Worker) CanAcceptJob() bool {
	return w.IsHealthy() && w.GetCurrentLoad() < w.GetCapacity()
}

// ExecuteJob executes a job
func (w *Worker) ExecuteJob(ctx context.Context, j *job.Job) (*job.JobResult, error) {
	if !w.CanAcceptJob() {
		return nil, fmt.Errorf("worker %s cannot accept job: at capacity or unhealthy", w.id)
	}

	// Add job to current jobs
	w.currentJobsMux.Lock()
	w.currentJobs[j.ID] = j
	w.currentJobsMux.Unlock()

	// Remove job from current jobs when done
	defer func() {
		w.currentJobsMux.Lock()
		delete(w.currentJobs, j.ID)
		w.currentJobsMux.Unlock()
	}()

	// Update job status to running
	j.WorkerID = w.id
	if err := j.UpdateStatus(job.JobStatusRunning); err != nil {
		return nil, fmt.Errorf("failed to update job status: %v", err)
	}

	fmt.Printf("Worker %s executing job %s (%s)\n", w.id, j.ID, j.Type)

	// Execute the job
	result, err := w.executor.Execute(ctx, j)
	if err != nil {
		fmt.Printf("Worker %s failed to execute job %s: %v\n", w.id, j.ID, err)
		return result, err
	}

	fmt.Printf("Worker %s completed job %s with status %s\n", w.id, j.ID, result.Status)
	return result, nil
}

// GetCurrentJobs returns the jobs currently being executed
func (w *Worker) GetCurrentJobs() []*job.Job {
	w.currentJobsMux.RLock()
	defer w.currentJobsMux.RUnlock()

	jobs := make([]*job.Job, 0, len(w.currentJobs))
	for _, j := range w.currentJobs {
		jobs = append(jobs, j)
	}

	return jobs
}

// UpdateHeartbeat updates the last heartbeat time
func (w *Worker) UpdateHeartbeat() {
	w.heartbeatMux.Lock()
	defer w.heartbeatMux.Unlock()
	w.lastHeartbeat = time.Now()
}

// GetLastHeartbeat returns the last heartbeat time
func (w *Worker) GetLastHeartbeat() time.Time {
	w.heartbeatMux.RLock()
	defer w.heartbeatMux.RUnlock()
	return w.lastHeartbeat
}

// SetHealthy sets the health status of the worker
func (w *Worker) SetHealthy(healthy bool) {
	w.heartbeatMux.Lock()
	defer w.heartbeatMux.Unlock()
	w.isHealthy = healthy
}

// heartbeatLoop sends periodic heartbeats to the scheduler
func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !w.isRunning {
				return
			}

			w.sendHeartbeat()
		}
	}
}

// jobPollingLoop polls for new jobs from the scheduler
func (w *Worker) jobPollingLoop(ctx context.Context) {
	ticker := time.NewTicker(w.config.JobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !w.isRunning {
				return
			}

			w.pollForJobs(ctx)
		}
	}
}

// sendHeartbeat sends a heartbeat to the scheduler
func (w *Worker) sendHeartbeat() {
	// TODO: Implement HTTP client to send heartbeat to scheduler
	// For now, just update local heartbeat
	w.UpdateHeartbeat()
	fmt.Printf("Worker %s sent heartbeat\n", w.id)
}

// pollForJobs polls the scheduler for new jobs
func (w *Worker) pollForJobs(ctx context.Context) {
	if !w.CanAcceptJob() {
		return // Skip polling if we can't accept jobs
	}

	// TODO: Implement HTTP client to poll scheduler for jobs
	// For now, this is a placeholder
	fmt.Printf("Worker %s polling for jobs (capacity: %d/%d)\n",
		w.id, w.GetCurrentLoad(), w.GetCapacity())
}

// ensureWorkingDirectory creates the working directory if it doesn't exist
func (w *Worker) ensureWorkingDirectory() error {
	return ensureDirectory(w.config.WorkingDirectory)
}

// GetInfo returns worker information
func (w *Worker) GetInfo() map[string]interface{} {
	return map[string]interface{}{
		"id":             w.ID(),
		"healthy":        w.IsHealthy(),
		"capacity":       w.GetCapacity(),
		"current_load":   w.GetCurrentLoad(),
		"can_accept":     w.CanAcceptJob(),
		"last_heartbeat": w.GetLastHeartbeat(),
		"current_jobs":   len(w.currentJobs),
		"working_dir":    w.config.WorkingDirectory,
	}
}
