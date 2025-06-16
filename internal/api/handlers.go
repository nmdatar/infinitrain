package api

import (
	"encoding/json"
	"fmt"
	"infinitrain/internal/config"
	"infinitrain/internal/scheduler"
	"infinitrain/pkg/job"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Server holds the API server dependencies
type Server struct {
	config  *config.Config
	store   job.Store
	manager job.JobManager
	workers job.WorkerRegistry
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, store job.Store, manager job.JobManager, workers job.WorkerRegistry) *Server {
	return &Server{
		config:  cfg,
		store:   store,
		manager: manager,
		workers: workers,
	}
}

// SetupRoutes configures the HTTP routes
func (s *Server) SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Job endpoints
	api.HandleFunc("/jobs", s.handleSubmitJob).Methods("POST")
	api.HandleFunc("/jobs", s.handleListJobs).Methods("GET")
	api.HandleFunc("/jobs/{id}", s.handleGetJob).Methods("GET")
	api.HandleFunc("/jobs/{id}", s.handleCancelJob).Methods("DELETE")

	// Worker endpoints
	api.HandleFunc("/workers", s.handleListWorkers).Methods("GET")
	api.HandleFunc("/workers/{id}/heartbeat", s.handleWorkerHeartbeat).Methods("POST")

	// System endpoints
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

	// Middleware
	r.Use(s.loggingMiddleware)
	r.Use(s.corsMiddleware)

	return r
}

// Job Handlers

func (s *Server) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	var request job.JobRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	j, err := s.manager.Submit(r.Context(), &request)
	if err != nil {
		if job.IsValidationError(err) {
			s.writeError(w, http.StatusBadRequest, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "failed to submit job: "+err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusCreated, j)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	var filters []job.Filter

	if status := r.URL.Query().Get("status"); status != "" {
		filters = append(filters, job.Filter{
			Field:    "status",
			Operator: "eq",
			Value:    status,
		})
	}

	if workerID := r.URL.Query().Get("worker_id"); workerID != "" {
		filters = append(filters, job.Filter{
			Field:    "worker_id",
			Operator: "eq",
			Value:    workerID,
		})
	}

	// Parse limit
	limit := 100 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	jobs, err := s.manager.ListJobs(r.Context(), filters...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list jobs: "+err.Error())
		return
	}

	// Apply limit
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}

	response := map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	j, err := s.manager.GetJob(r.Context(), jobID)
	if err != nil {
		if job.IsJobNotFoundError(err) {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "failed to get job: "+err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, j)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	err := s.manager.CancelJob(r.Context(), jobID)
	if err != nil {
		if job.IsJobNotFoundError(err) {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "failed to cancel job: "+err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "job cancelled"})
}

// Worker Handlers

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	workers, err := s.workers.ListWorkers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to list workers: "+err.Error())
		return
	}

	// Convert to response format
	var workerInfo []map[string]interface{}
	for _, worker := range workers {
		workerInfo = append(workerInfo, map[string]interface{}{
			"id":           worker.ID(),
			"healthy":      worker.IsHealthy(),
			"capacity":     worker.GetCapacity(),
			"current_load": worker.GetCurrentLoad(),
			"can_accept":   worker.CanAcceptJob(),
		})
	}

	response := map[string]interface{}{
		"workers": workerInfo,
		"count":   len(workerInfo),
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workerID := vars["id"]

	err := s.workers.Heartbeat(r.Context(), workerID)
	if err != nil {
		if job.IsWorkerNotFoundError(err) {
			s.writeError(w, http.StatusNotFound, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, "failed to update heartbeat: "+err.Error())
		}
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "heartbeat updated"})
}

// System Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Basic health check
	workers, err := s.workers.ListWorkers(r.Context())
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable, "failed to check workers: "+err.Error())
		return
	}

	healthyWorkers := 0
	for _, worker := range workers {
		if worker.IsHealthy() {
			healthyWorkers++
		}
	}

	health := map[string]interface{}{
		"status":          "healthy",
		"total_workers":   len(workers),
		"healthy_workers": healthyWorkers,
		"timestamp":       scheduler.Now(),
	}

	s.writeJSON(w, http.StatusOK, health)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get job counts by status
	statuses := []job.JobStatus{
		job.JobStatusPending,
		job.JobStatusQueued,
		job.JobStatusRunning,
		job.JobStatusCompleted,
		job.JobStatusFailed,
		job.JobStatusCancelled,
	}

	jobCounts := make(map[string]int)
	totalJobs := 0

	for _, status := range statuses {
		jobs, err := s.store.List(r.Context(), job.Filter{
			Field:    "status",
			Operator: "eq",
			Value:    string(status),
		})
		if err == nil {
			count := len(jobs)
			jobCounts[string(status)] = count
			totalJobs += count
		}
	}

	// Get worker metrics
	workers, _ := s.workers.ListWorkers(r.Context())
	totalCapacity := 0
	totalLoad := 0
	healthyWorkers := 0

	for _, worker := range workers {
		totalCapacity += worker.GetCapacity()
		totalLoad += worker.GetCurrentLoad()
		if worker.IsHealthy() {
			healthyWorkers++
		}
	}

	metrics := map[string]interface{}{
		"jobs": map[string]interface{}{
			"total":     totalJobs,
			"by_status": jobCounts,
		},
		"workers": map[string]interface{}{
			"total":          len(workers),
			"healthy":        healthyWorkers,
			"total_capacity": totalCapacity,
			"total_load":     totalLoad,
			"utilization":    calculateUtilization(totalLoad, totalCapacity),
		},
		"timestamp": scheduler.Now(),
	}

	s.writeJSON(w, http.StatusOK, metrics)
}

// Helper methods

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

// Middleware

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("[%s] %s %s\n", scheduler.Now().Format("2006-01-02 15:04:05"), r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func calculateUtilization(load, capacity int) float64 {
	if capacity == 0 {
		return 0.0
	}
	return float64(load) / float64(capacity) * 100.0
}
