package scheduler

import (
	"context"
	"infinitrain/pkg/job"
	"sync"
	"time"
)

// MemoryStore is a simple in-memory implementation of the job.Store interface
type MemoryStore struct {
	jobs   map[string]*job.Job
	mutex  sync.RWMutex
}

// NewMemoryStore creates a new in-memory job store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*job.Job),
	}
}

// Create stores a new job
func (s *MemoryStore) Create(ctx context.Context, j *job.Job) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if job already exists
	if _, exists := s.jobs[j.ID]; exists {
		return job.NewValidationError("job already exists: " + j.ID)
	}

	// Create a copy to avoid mutations
	jobCopy := *j
	s.jobs[j.ID] = &jobCopy

	return nil
}

// Get retrieves a job by ID
func (s *MemoryStore) Get(ctx context.Context, jobID string) (*job.Job, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	j, exists := s.jobs[jobID]
	if !exists {
		return nil, job.NewJobNotFoundError(jobID)
	}

	// Return a copy to avoid mutations
	jobCopy := *j
	return &jobCopy, nil
}

// Update updates an existing job
func (s *MemoryStore) Update(ctx context.Context, j *job.Job) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.jobs[j.ID]; !exists {
		return job.NewJobNotFoundError(j.ID)
	}

	// Create a copy to avoid mutations
	jobCopy := *j
	s.jobs[j.ID] = &jobCopy

	return nil
}

// Delete removes a job from storage
func (s *MemoryStore) Delete(ctx context.Context, jobID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.jobs[jobID]; !exists {
		return job.NewJobNotFoundError(jobID)
	}

	delete(s.jobs, jobID)
	return nil
}

// List returns jobs with optional filtering
func (s *MemoryStore) List(ctx context.Context, filters ...job.Filter) ([]*job.Job, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []*job.Job

	for _, j := range s.jobs {
		if s.matchesFilters(j, filters) {
			// Return a copy to avoid mutations
			jobCopy := *j
			result = append(result, &jobCopy)
		}
	}

	return result, nil
}

// UpdateStatus updates the status of a job
func (s *MemoryStore) UpdateStatus(ctx context.Context, jobID string, status job.JobStatus) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	j, exists := s.jobs[jobID]
	if !exists {
		return job.NewJobNotFoundError(jobID)
	}

	// Update the status and timestamps
	if err := j.UpdateStatus(status); err != nil {
		return err
	}

	return nil
}

// matchesFilters checks if a job matches the given filters
func (s *MemoryStore) matchesFilters(j *job.Job, filters []job.Filter) bool {
	for _, filter := range filters {
		if !s.matchesFilter(j, filter) {
			return false
		}
	}
	return true
}

// matchesFilter checks if a job matches a single filter
func (s *MemoryStore) matchesFilter(j *job.Job, filter job.Filter) bool {
	var fieldValue interface{}

	// Extract field value from job
	switch filter.Field {
	case "id":
		fieldValue = j.ID
	case "type":
		fieldValue = string(j.Type)
	case "status":
		fieldValue = string(j.Status)
	case "worker_id":
		fieldValue = j.WorkerID
	case "priority":
		fieldValue = j.Priority
	case "created_at":
		fieldValue = j.CreatedAt
	case "started_at":
		if j.StartedAt != nil {
			fieldValue = *j.StartedAt
		} else {
			fieldValue = nil
		}
	case "completed_at":
		if j.CompletedAt != nil {
			fieldValue = *j.CompletedAt
		} else {
			fieldValue = nil
		}
	default:
		return false // Unknown field
	}

	// Apply operator
	switch filter.Operator {
	case "eq":
		return fieldValue == filter.Value
	case "ne":
		return fieldValue != filter.Value
	case "gt":
		return s.compareValues(fieldValue, filter.Value) > 0
	case "lt":
		return s.compareValues(fieldValue, filter.Value) < 0
	case "gte":
		return s.compareValues(fieldValue, filter.Value) >= 0
	case "lte":
		return s.compareValues(fieldValue, filter.Value) <= 0
	case "in":
		if slice, ok := filter.Value.([]interface{}); ok {
			for _, v := range slice {
				if fieldValue == v {
					return true
				}
			}
		}
		return false
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := filter.Value.(string); ok {
				return contains(str, substr)
			}
		}
		return false
	default:
		return false // Unknown operator
	}
}

// compareValues compares two values for ordering operations
func (s *MemoryStore) compareValues(a, b interface{}) int {
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case string:
		if vb, ok := b.(string); ok {
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case time.Time:
		if vb, ok := b.(time.Time); ok {
			if va.Before(vb) {
				return -1
			} else if va.After(vb) {
				return 1
			}
			return 0
		}
	}
	return 0
}

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && 
		   (str == substr || 
		    (len(substr) > 0 && findSubstring(str, substr)))
}

// Simple substring search (case-insensitive)
func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLowerCase(str[i+j]) != toLowerCase(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Simple case conversion for ASCII characters
func toLowerCase(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}

// GetJobsByStatus is a convenience method to get jobs by status
func (s *MemoryStore) GetJobsByStatus(ctx context.Context, status job.JobStatus) ([]*job.Job, error) {
	return s.List(ctx, job.Filter{
		Field:    "status",
		Operator: "eq",
		Value:    string(status),
	})
}

// GetJobsByWorker is a convenience method to get jobs by worker ID
func (s *MemoryStore) GetJobsByWorker(ctx context.Context, workerID string) ([]*job.Job, error) {
	return s.List(ctx, job.Filter{
		Field:    "worker_id",
		Operator: "eq",
		Value:    workerID,
	})
}

// Count returns the total number of jobs in the store
func (s *MemoryStore) Count(ctx context.Context) int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.jobs)
}

// Clear removes all jobs from the store (useful for testing)
func (s *MemoryStore) Clear(ctx context.Context) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.jobs = make(map[string]*job.Job)
} 