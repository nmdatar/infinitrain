package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Worker    WorkerConfig    `yaml:"worker"`
	Logging   LoggingConfig   `yaml:"logging"`
	Redis     RedisConfig     `yaml:"redis"`
}

// SchedulerConfig holds scheduler-specific configuration
type SchedulerConfig struct {
	Port                int           `yaml:"port"`
	Host                string        `yaml:"host"`
	RedisURL            string        `yaml:"redis_url"`
	MaxConcurrentJobs   int           `yaml:"max_concurrent_jobs"`
	JobTimeout          time.Duration `yaml:"job_timeout"`
	WorkerTimeout       time.Duration `yaml:"worker_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	ID                  string        `yaml:"id"`
	SchedulerURL        string        `yaml:"scheduler_url"`
	MaxConcurrentJobs   int           `yaml:"max_concurrent_jobs"`
	HeartbeatInterval   time.Duration `yaml:"heartbeat_interval"`
	JobPollInterval     time.Duration `yaml:"job_poll_interval"`
	WorkingDirectory    string        `yaml:"working_directory"`
	LogLevel            string        `yaml:"log_level"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	URL      string `yaml:"url"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	config := &Config{
		Scheduler: SchedulerConfig{
			Port:                getEnvInt("SCHEDULER_PORT", 8080),
			Host:                getEnvString("SCHEDULER_HOST", "0.0.0.0"),
			RedisURL:            getEnvString("REDIS_URL", "redis://localhost:6379"),
			MaxConcurrentJobs:   getEnvInt("SCHEDULER_MAX_CONCURRENT_JOBS", 100),
			JobTimeout:          getEnvDuration("SCHEDULER_JOB_TIMEOUT", 30*time.Minute),
			WorkerTimeout:       getEnvDuration("SCHEDULER_WORKER_TIMEOUT", 60*time.Second),
			HealthCheckInterval: getEnvDuration("SCHEDULER_HEALTH_CHECK_INTERVAL", 30*time.Second),
		},
		Worker: WorkerConfig{
			ID:                getEnvString("WORKER_ID", generateWorkerID()),
			SchedulerURL:      getEnvString("SCHEDULER_URL", "http://localhost:8080"),
			MaxConcurrentJobs: getEnvInt("WORKER_MAX_CONCURRENT_JOBS", 5),
			HeartbeatInterval: getEnvDuration("WORKER_HEARTBEAT_INTERVAL", 30*time.Second),
			JobPollInterval:   getEnvDuration("WORKER_JOB_POLL_INTERVAL", 5*time.Second),
			WorkingDirectory:  getEnvString("WORKER_WORKING_DIRECTORY", "/tmp/infinitrain"),
			LogLevel:          getEnvString("WORKER_LOG_LEVEL", "info"),
		},
		Logging: LoggingConfig{
			Level:  getEnvString("LOG_LEVEL", "info"),
			Format: getEnvString("LOG_FORMAT", "json"),
			Output: getEnvString("LOG_OUTPUT", "stdout"),
		},
		Redis: RedisConfig{
			URL:      getEnvString("REDIS_URL", "redis://localhost:6379"),
			Password: getEnvString("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			PoolSize: getEnvInt("REDIS_POOL_SIZE", 10),
		},
	}

	return config
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Scheduler.Port <= 0 || c.Scheduler.Port > 65535 {
		return fmt.Errorf("invalid scheduler port: %d", c.Scheduler.Port)
	}

	if c.Scheduler.RedisURL == "" {
		return fmt.Errorf("redis URL cannot be empty")
	}

	if c.Worker.SchedulerURL == "" {
		return fmt.Errorf("scheduler URL cannot be empty")
	}

	if c.Worker.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("worker max concurrent jobs must be positive")
	}

	if c.Scheduler.MaxConcurrentJobs <= 0 {
		return fmt.Errorf("scheduler max concurrent jobs must be positive")
	}

	return nil
}

// GetSchedulerAddress returns the full scheduler address
func (c *Config) GetSchedulerAddress() string {
	return fmt.Sprintf("%s:%d", c.Scheduler.Host, c.Scheduler.Port)
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func generateWorkerID() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("worker-%s-%d", hostname, time.Now().Unix())
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return getEnvString("ENVIRONMENT", "development") == "production"
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return !c.IsProduction()
} 