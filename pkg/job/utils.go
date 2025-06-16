package job

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateJobID generates a unique job ID
func GenerateJobID() string {
	// Generate timestamp prefix
	timestamp := time.Now().Unix()
	
	// Generate random suffix
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomHex := hex.EncodeToString(randomBytes)
	
	return fmt.Sprintf("job-%d-%s", timestamp, randomHex)
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) error {
	return ValidationError{Message: message}
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(ValidationError)
	return ok
}

// JobNotFoundError represents a job not found error
type JobNotFoundError struct {
	JobID string
}

func (e JobNotFoundError) Error() string {
	return fmt.Sprintf("job not found: %s", e.JobID)
}

// NewJobNotFoundError creates a new job not found error
func NewJobNotFoundError(jobID string) error {
	return JobNotFoundError{JobID: jobID}
}

// IsJobNotFoundError checks if an error is a job not found error
func IsJobNotFoundError(err error) bool {
	_, ok := err.(JobNotFoundError)
	return ok
}

// WorkerNotFoundError represents a worker not found error
type WorkerNotFoundError struct {
	WorkerID string
}

func (e WorkerNotFoundError) Error() string {
	return fmt.Sprintf("worker not found: %s", e.WorkerID)
}

// NewWorkerNotFoundError creates a new worker not found error
func NewWorkerNotFoundError(workerID string) error {
	return WorkerNotFoundError{WorkerID: workerID}
}

// IsWorkerNotFoundError checks if an error is a worker not found error
func IsWorkerNotFoundError(err error) bool {
	_, ok := err.(WorkerNotFoundError)
	return ok
}

// ExecutionError represents a job execution error
type ExecutionError struct {
	JobID   string
	Message string
	Cause   error
}

func (e ExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("execution error for job %s: %s: %v", e.JobID, e.Message, e.Cause)
	}
	return fmt.Sprintf("execution error for job %s: %s", e.JobID, e.Message)
}

func (e ExecutionError) Unwrap() error {
	return e.Cause
}

// NewExecutionError creates a new execution error
func NewExecutionError(jobID, message string, cause error) error {
	return ExecutionError{
		JobID:   jobID,
		Message: message,
		Cause:   cause,
	}
}

// IsExecutionError checks if an error is an execution error
func IsExecutionError(err error) bool {
	_, ok := err.(ExecutionError)
	return ok
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	JobID   string
	Timeout time.Duration
}

func (e TimeoutError) Error() string {
	return fmt.Sprintf("job %s timed out after %v", e.JobID, e.Timeout)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(jobID string, timeout time.Duration) error {
	return TimeoutError{
		JobID:   jobID,
		Timeout: timeout,
	}
}

// IsTimeoutError checks if an error is a timeout error
func IsTimeoutError(err error) bool {
	_, ok := err.(TimeoutError)
	return ok
}

// Helper functions for job status transitions
func (j *Job) CanTransitionTo(newStatus JobStatus) bool {
	switch j.Status {
	case JobStatusPending:
		return newStatus == JobStatusQueued || newStatus == JobStatusCancelled
	case JobStatusQueued:
		return newStatus == JobStatusRunning || newStatus == JobStatusCancelled
	case JobStatusRunning:
		return newStatus == JobStatusCompleted || newStatus == JobStatusFailed || 
			   newStatus == JobStatusCancelled || newStatus == JobStatusRetrying
	case JobStatusRetrying:
		return newStatus == JobStatusQueued || newStatus == JobStatusFailed || newStatus == JobStatusCancelled
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return false // Terminal states
	default:
		return false
	}
}

// UpdateStatus safely updates the job status if the transition is valid
func (j *Job) UpdateStatus(newStatus JobStatus) error {
	if !j.CanTransitionTo(newStatus) {
		return NewValidationError(fmt.Sprintf("cannot transition from %s to %s", j.Status, newStatus))
	}
	
	j.Status = newStatus
	
	// Update timestamps based on status
	now := time.Now()
	switch newStatus {
	case JobStatusRunning:
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		if j.CompletedAt == nil {
			j.CompletedAt = &now
		}
	}
	
	return nil
}

// GetDuration returns the duration of the job execution
func (j *Job) GetDuration() time.Duration {
	if j.StartedAt == nil {
		return 0
	}
	
	endTime := time.Now()
	if j.CompletedAt != nil {
		endTime = *j.CompletedAt
	}
	
	return endTime.Sub(*j.StartedAt)
}

// IsTerminal returns true if the job is in a terminal state
func (j *Job) IsTerminal() bool {
	return j.Status == JobStatusCompleted || j.Status == JobStatusFailed || j.Status == JobStatusCancelled
}

// IsRunning returns true if the job is currently running
func (j *Job) IsRunning() bool {
	return j.Status == JobStatusRunning
}

// IsPending returns true if the job is pending or queued
func (j *Job) IsPending() bool {
	return j.Status == JobStatusPending || j.Status == JobStatusQueued
} 