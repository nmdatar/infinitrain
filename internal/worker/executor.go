package worker

import (
	"bytes"
	"context"
	"fmt"
	"infinitrain/pkg/job"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// JobExecutor implements the job.Executor interface
type JobExecutor struct {
	workingDir string
}

// NewJobExecutor creates a new job executor
func NewJobExecutor(workingDir string) *JobExecutor {
	return &JobExecutor{
		workingDir: workingDir,
	}
}

// Execute runs a job and returns the result
func (e *JobExecutor) Execute(ctx context.Context, j *job.Job) (*job.JobResult, error) {
	startTime := time.Now()

	// Create timeout context if job has timeout
	if j.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, j.Timeout)
		defer cancel()
	}

	var output string
	var err error
	var exitCode int

	// Execute based on job type
	switch j.Type {
	case job.JobTypeCommand:
		output, exitCode, err = e.executeCommand(ctx, j)
	case job.JobTypeScript:
		output, exitCode, err = e.executeScript(ctx, j)
	case job.JobTypeHTTP:
		output, exitCode, err = e.executeHTTP(ctx, j)
	case job.JobTypeFile:
		output, exitCode, err = e.executeFile(ctx, j)
	default:
		return nil, fmt.Errorf("unsupported job type: %s", j.Type)
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Determine final status
	status := job.JobStatusCompleted
	errorMessage := ""
	if err != nil {
		status = job.JobStatusFailed
		errorMessage = err.Error()
		if exitCode == 0 {
			exitCode = 1 // Default error exit code
		}
	}

	result := &job.JobResult{
		JobID:       j.ID,
		Status:      status,
		Output:      output,
		Error:       errorMessage,
		ExitCode:    exitCode,
		StartedAt:   startTime,
		CompletedAt: endTime,
		Duration:    duration,
	}

	return result, nil
}

// CanExecute checks if this executor can handle the given job type
func (e *JobExecutor) CanExecute(jobType job.JobType) bool {
	switch jobType {
	case job.JobTypeCommand, job.JobTypeScript, job.JobTypeHTTP, job.JobTypeFile:
		return true
	default:
		return false
	}
}

// Name returns the name of this executor
func (e *JobExecutor) Name() string {
	return "default-executor"
}

// executeCommand executes a shell command
func (e *JobExecutor) executeCommand(ctx context.Context, j *job.Job) (string, int, error) {
	// Parse command and arguments
	parts := strings.Fields(j.Command)
	if len(parts) == 0 {
		return "", 1, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = e.workingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range j.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n---STDERR---\n"
		}
		output += stderr.String()
	}

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return output, exitCode, err
}

// executeScript executes a script
func (e *JobExecutor) executeScript(ctx context.Context, j *job.Job) (string, int, error) {
	// Create temporary script file
	scriptFile := filepath.Join(e.workingDir, fmt.Sprintf("script_%s.sh", j.ID))

	// Write script content to file
	err := os.WriteFile(scriptFile, []byte(j.Script), 0755)
	if err != nil {
		return "", 1, fmt.Errorf("failed to write script file: %v", err)
	}

	// Clean up script file after execution
	defer func() {
		os.Remove(scriptFile)
	}()

	// Execute script
	cmd := exec.CommandContext(ctx, "/bin/bash", scriptFile)
	cmd.Dir = e.workingDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range j.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n---STDERR---\n"
		}
		output += stderr.String()
	}

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return output, exitCode, err
}

// executeHTTP executes an HTTP request
func (e *JobExecutor) executeHTTP(ctx context.Context, j *job.Job) (string, int, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, j.Method, j.URL, nil)
	if err != nil {
		return "", 1, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers from environment
	for key, value := range j.Environment {
		if strings.HasPrefix(key, "HTTP_HEADER_") {
			headerName := strings.TrimPrefix(key, "HTTP_HEADER_")
			req.Header.Set(headerName, value)
		}
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", 1, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 1, fmt.Errorf("failed to read response body: %v", err)
	}

	// Format output
	output := fmt.Sprintf("Status: %d %s\n", resp.StatusCode, resp.Status)
	if len(body) > 0 {
		output += fmt.Sprintf("Body: %s", string(body))
	}

	// Consider 2xx status codes as success
	exitCode := 0
	if resp.StatusCode >= 400 {
		exitCode = 1
		err = fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
	}

	return output, exitCode, err
}

// executeFile executes file operations
func (e *JobExecutor) executeFile(ctx context.Context, j *job.Job) (string, int, error) {
	// Determine operation from environment or default to "read"
	operation := "read"
	if op, exists := j.Environment["FILE_OPERATION"]; exists {
		operation = op
	}

	filePath := j.FilePath
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(e.workingDir, filePath)
	}

	switch operation {
	case "read":
		return e.readFile(filePath)
	case "stat":
		return e.statFile(filePath)
	case "list":
		return e.listDirectory(filePath)
	default:
		return "", 1, fmt.Errorf("unsupported file operation: %s", operation)
	}
}

// readFile reads a file and returns its content
func (e *JobExecutor) readFile(filePath string) (string, int, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", 1, fmt.Errorf("failed to read file: %v", err)
	}

	output := fmt.Sprintf("File: %s\nSize: %d bytes\nContent:\n%s",
		filePath, len(content), string(content))

	return output, 0, nil
}

// statFile gets file information
func (e *JobExecutor) statFile(filePath string) (string, int, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", 1, fmt.Errorf("failed to stat file: %v", err)
	}

	output := fmt.Sprintf("File: %s\nSize: %d bytes\nMode: %s\nModified: %s\nIsDir: %v",
		filePath, info.Size(), info.Mode(), info.ModTime(), info.IsDir())

	return output, 0, nil
}

// listDirectory lists directory contents
func (e *JobExecutor) listDirectory(dirPath string) (string, int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", 1, fmt.Errorf("failed to read directory: %v", err)
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Directory: %s\nEntries:\n", dirPath))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		output.WriteString(fmt.Sprintf("  %s (%d bytes) %s\n",
			entry.Name(), info.Size(), info.ModTime().Format("2006-01-02 15:04:05")))
	}

	return output.String(), 0, nil
}
