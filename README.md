# InfiniTrain - Distributed Job Training System

A lightweight, high-performance distributed job execution system built in Go. InfiniTrain enables you to distribute and execute jobs across multiple worker nodes with fault tolerance, monitoring, and simple management.

## 🎯 Project Goals

**Primary Goal**: Build a production-ready distributed job execution system that can:
- Execute jobs across multiple worker nodes
- Handle job failures gracefully with automatic retry
- Provide real-time job status and monitoring
- Scale horizontally by adding more worker nodes
- Deploy easily with Docker Compose

**MVP Timeline**: 4-day rapid development cycle focusing on core functionality over advanced features.

## 🏗️ Architecture Overview

```
┌─────────────────┐                           ┌─────────────────┐
│   Job Clients   │                           │   CLI Tool      │
└─────────┬───────┘                           └─────────┬───────┘
          │                                             │
          └─────────────────┬───────────────────────────┘
                            │
               ┌─────────────▼──────────────┐
               │      Job Scheduler         │
               │    (REST API Server)       │
               └─────────────┬──────────────┘
                             │
                    ┌────────▼────────┐
                    │   Redis Queue   │
                    │   (Job Store)   │
                    └────────┬────────┘
                             │
     ┌───────────────────────┼───────────────────────┐
     │                       │                       │
┌────▼─────┐           ┌─────▼─────┐           ┌─────▼─────┐
│ Worker 1 │           │ Worker 2  │           │ Worker N  │
│ Node     │           │ Node      │           │ Node      │
└──────────┘           └───────────┘           └───────────┘
```

### Core Components

- **Job Scheduler**: Central coordinator that receives jobs and distributes them to available workers
- **Worker Nodes**: Distributed execution units that pull and execute jobs
- **Redis Queue**: Simple and reliable job queue and state storage
- **REST API**: HTTP interface for job submission, monitoring, and management
- **CLI Tool**: Command-line interface for job management

## 🚀 Features

### Current Status: 🚧 In Development

- [ ] **Job Submission**: Submit jobs via REST API with JSON specification
- [ ] **Distributed Execution**: Execute jobs across multiple worker nodes
- [ ] **Job Types**: Support for command execution, script running, and HTTP requests
- [ ] **Status Tracking**: Real-time job status and progress monitoring
- [ ] **Worker Management**: Automatic worker registration and health monitoring
- [ ] **Fault Tolerance**: Automatic job retry and worker failure detection
- [ ] **CLI Tool**: Command-line tool for job and worker management
- [ ] **Docker Deployment**: Easy deployment with Docker Compose

### Planned Features (Post-MVP)

- [ ] Job dependencies and workflows (DAG support)
- [ ] Advanced scheduling algorithms
- [ ] Resource-aware job placement
- [ ] Job templates and batch processing
- [ ] Advanced monitoring and metrics
- [ ] Authentication and authorization
- [ ] Kubernetes deployment

## 📋 Job Types Supported

1. **Command Jobs**: Execute shell commands
   ```json
   {
     "type": "command",
     "command": "echo 'Hello World'"
   }
   ```

2. **Script Jobs**: Run bash/shell scripts
   ```json
   {
     "type": "script",
     "script": "#!/bin/bash\necho 'Running script'\ndate"
   }
   ```

3. **HTTP Jobs**: Make HTTP requests
   ```json
   {
     "type": "http",
     "url": "https://api.example.com/webhook",
     "method": "POST"
   }
   ```

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **Queue & Storage**: Redis
- **Communication**: HTTP REST API
- **Containerization**: Docker & Docker Compose
- **Testing**: Go built-in testing + manual integration tests

## 🏃‍♂️ Quick Start

> **Note**: This section will be updated as the project is built

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Redis (or use Docker Compose setup)

### Development Setup

```bash
# Clone the repository
git clone <repository-url>
cd infinitrain

# Initialize Go modules
go mod init infinitrain
go mod tidy

# Start Redis (using Docker)
docker run -d -p 6379:6379 redis:latest

# Run the scheduler
go run cmd/scheduler/main.go

# Run a worker (in another terminal)
go run cmd/worker/main.go

# Submit a test job
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"type":"command","command":"echo Hello InfiniTrain"}'
```

### Production Deployment

```bash
# Deploy with Docker Compose
docker-compose up -d

# Scale workers
docker-compose up -d --scale worker=3

# View logs
docker-compose logs -f
```

## 📊 API Documentation

### Job Submission
```http
POST /api/v1/jobs
Content-Type: application/json

{
  "id": "job-123",
  "type": "command",
  "command": "echo 'Hello World'",
  "timeout": "5m",
  "retries": 3,
  "priority": 1,
  "tags": ["example", "test"]
}
```

### Job Status
```http
GET /api/v1/jobs/{job-id}
```

### List Jobs
```http
GET /api/v1/jobs
```

### Worker Status
```http
GET /api/v1/workers
```

### System Health
```http
GET /api/v1/health
```

## 🧪 Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
make test-integration

# Load testing
make load-test
```

## 📈 Performance Goals (MVP)

- **Concurrent Jobs**: 100+ simultaneous jobs
- **Worker Nodes**: Support for 5+ workers
- **Job Latency**: < 1 minute for simple command jobs
- **Failure Detection**: < 30 seconds for worker failures
- **Success Rate**: 95%+ job completion rate

## 🤝 Contributing

This project is currently in rapid MVP development. Contribution guidelines will be added after the initial 4-day development cycle.

## 📜 License

MIT License - see LICENSE file for details

## 🗓️ Development Timeline

- **Day 1**: Foundation & Basic Job System ⏳
- **Day 2**: Distribution & Communication
- **Day 3**: Reliability & Management
- **Day 4**: Polish & Deployment

---

**Status**: 🚧 Active Development - MVP Phase
**Last Updated**: [Current Date]
**Next Milestone**: Day 1 - Single Node Job Execution
