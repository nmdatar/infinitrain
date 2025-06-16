package job

import (
	"testing"
	"time"
)

func TestJobRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request JobRequest
		wantErr bool
	}{
		{
			name: "valid command job",
			request: JobRequest{
				Type:    JobTypeCommand,
				Command: "echo 'hello'",
			},
			wantErr: false,
		},
		{
			name: "valid script job",
			request: JobRequest{
				Type:   JobTypeScript,
				Script: "#!/bin/bash\necho 'hello'",
			},
			wantErr: false,
		},
		{
			name: "valid HTTP job",
			request: JobRequest{
				Type: JobTypeHTTP,
				URL:  "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "empty type",
			request: JobRequest{
				Command: "echo 'hello'",
			},
			wantErr: true,
		},
		{
			name: "command job without command",
			request: JobRequest{
				Type: JobTypeCommand,
			},
			wantErr: true,
		},
		{
			name: "script job without script",
			request: JobRequest{
				Type: JobTypeScript,
			},
			wantErr: true,
		},
		{
			name: "HTTP job without URL",
			request: JobRequest{
				Type: JobTypeHTTP,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("JobRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobRequest_ToJob(t *testing.T) {
	request := JobRequest{
		Type:     JobTypeCommand,
		Command:  "echo 'hello'",
		Timeout:  "5m",
		Priority: 2,
		Retries:  3,
		Tags:     []string{"test", "example"},
	}

	job, err := request.ToJob()
	if err != nil {
		t.Fatalf("JobRequest.ToJob() error = %v", err)
	}

	if job.ID == "" {
		t.Error("Expected job ID to be generated")
	}

	if job.Type != JobTypeCommand {
		t.Errorf("Expected job type %v, got %v", JobTypeCommand, job.Type)
	}

	if job.Command != "echo 'hello'" {
		t.Errorf("Expected command 'echo 'hello'', got %v", job.Command)
	}

	if job.Status != JobStatusPending {
		t.Errorf("Expected status %v, got %v", JobStatusPending, job.Status)
	}

	if job.Timeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", job.Timeout)
	}

	if job.Priority != 2 {
		t.Errorf("Expected priority 2, got %v", job.Priority)
	}

	if job.Retries != 3 {
		t.Errorf("Expected retries 3, got %v", job.Retries)
	}

	if len(job.Tags) != 2 || job.Tags[0] != "test" || job.Tags[1] != "example" {
		t.Errorf("Expected tags [test, example], got %v", job.Tags)
	}
}

func TestJob_UpdateStatus(t *testing.T) {
	job := &Job{
		ID:     "test-job",
		Status: JobStatusPending,
	}

	// Test valid transition
	err := job.UpdateStatus(JobStatusQueued)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if job.Status != JobStatusQueued {
		t.Errorf("Expected status %v, got %v", JobStatusQueued, job.Status)
	}

	// Test invalid transition
	err = job.UpdateStatus(JobStatusCompleted)
	if err == nil {
		t.Error("Expected error for invalid transition")
	}

	// Test terminal state
	job.Status = JobStatusCompleted
	err = job.UpdateStatus(JobStatusRunning)
	if err == nil {
		t.Error("Expected error when transitioning from terminal state")
	}
}

func TestJob_StatusMethods(t *testing.T) {
	job := &Job{
		ID:     "test-job",
		Status: JobStatusRunning,
	}

	if !job.IsRunning() {
		t.Error("Expected job to be running")
	}

	if job.IsTerminal() {
		t.Error("Expected job not to be terminal")
	}

	if job.IsPending() {
		t.Error("Expected job not to be pending")
	}

	job.Status = JobStatusCompleted
	if !job.IsTerminal() {
		t.Error("Expected completed job to be terminal")
	}

	job.Status = JobStatusPending
	if !job.IsPending() {
		t.Error("Expected pending job to be pending")
	}
}

func TestGenerateJobID(t *testing.T) {
	id1 := GenerateJobID()
	id2 := GenerateJobID()

	if id1 == "" {
		t.Error("Expected non-empty job ID")
	}

	if id1 == id2 {
		t.Error("Expected unique job IDs")
	}

	// Basic format check
	if len(id1) < 10 {
		t.Error("Expected job ID to have reasonable length")
	}
}
