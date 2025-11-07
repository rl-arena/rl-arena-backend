# RL Arena Backend Architecture

## Overview

RL Arena is a reinforcement learning agent competition platform that provides:
- **Agent Submission & Building**: Dockerized agent builds using Kaniko on Kubernetes
- **Real-time Build Monitoring**: Kubernetes Watch API + WebSocket notifications
- **Auto-Matchmaking System**: ELO-based automatic match creation
- **Match Execution**: gRPC communication with executor service
- **Security Scanning**: Trivy vulnerability scanning for Docker images

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                         │
└─────────────────┬───────────────────────────────────────────────┘
                  │ REST API + WebSocket
┌─────────────────▼───────────────────────────────────────────────┐
│                    Backend (Go + Gin)                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ API Layer (Handlers + Middleware)                          │ │
│  └────────────┬───────────────────────────────────────────────┘ │
│               │                                                  │
│  ┌────────────▼───────────────────────────────────────────────┐ │
│  │ Service Layer                                              │ │
│  │  - AgentService                                            │ │
│  │  - MatchService (ELO calculation)                          │ │
│  │  - MatchmakingService (auto-matching)                      │ │
│  │  - BuilderService (Kubernetes Jobs)                        │ │
│  │  - BuildMonitor (Watch API + WebSocket)                    │ │
│  │  - SecurityScanner (Trivy)                                 │ │
│  └────────────┬───────────────────────────────────────────────┘ │
│               │                                                  │
│  ┌────────────▼───────────────────────────────────────────────┐ │
│  │ Repository Layer (Database Operations)                     │ │
│  └────────────┬───────────────────────────────────────────────┘ │
└───────────────┼──────────────────────────────────────────────────┘
                │
┌───────────────▼──────────────────────────────────────────────────┐
│                      PostgreSQL Database                          │
│  Tables: users, agents, submissions, matches,                     │
│          matchmaking_queue, matchmaking_history                   │
└───────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                   Kubernetes Cluster                              │
│  ┌──────────────────────────────────────────────────────────────┐│
│  │ Kaniko Build Jobs (Docker image builds)                      ││
│  └──────────────────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│              Executor Service (Python gRPC)                       │
│  - Receives match execution requests                              │
│  - Runs RL environment simulations                                │
│  - Reports results back to backend                                │
└──────────────────────────────────────────────────────────────────┘
```

## Key Components

### 1. Build System
- **Kaniko**: In-cluster Docker builds without Docker daemon
- **Build Cache**: 24-hour TTL for faster rebuilds
- **Real-time Monitoring**: Kubernetes Watch API instead of polling
- **Retry Mechanism**: Up to 3 build retries on failure
- **Security Scanning**: Trivy scans all built images for CVEs

### 2. Auto-Matchmaking System (Phase 4)
- **Queue**: `matchmaking_queue` table with agent_id, environment_id, ELO rating
- **Matching Algorithm**: 
  - ELO range starts at ±100, expands to ±500 in steps
  - Priority-based (1-10, lower = higher priority)
  - FIFO for agents with same ELO/priority
- **Interval**: Runs every 30 seconds
- **Auto-enrollment**: Agents automatically join queue after successful build
- **Expiration**: Queue entries expire after 24 hours

### 3. WebSocket Notifications
- **Build Status**: Real-time build progress updates
- **Match Results**: Live match completion notifications
- **Hub Pattern**: Central hub manages all client connections
- **Authentication**: JWT-based WebSocket auth

### 4. Match Execution Flow
1. Match created (manual or auto-matchmaking)
2. Backend sends gRPC request to Executor
3. Executor runs simulation in RL environment
4. Results sent back to Backend
5. ELO ratings updated
6. WebSocket notification to clients

## Database Schema

### Core Tables
- **users**: User accounts
- **agents**: RL agents owned by users
- **submissions**: Code submissions with build status
- **matches**: Match records with results
- **environments**: Available RL environments (pong, etc.)

### Matchmaking Tables
- **matchmaking_queue**: Active matchmaking queue
- **matchmaking_history**: Historical match pairings

## Technology Stack

- **Backend**: Go 1.25, Gin web framework
- **Database**: PostgreSQL 15+
- **Kubernetes**: client-go v0.34.1
- **Container Registry**: Docker Hub / Private registry
- **Build Tool**: Kaniko
- **Security**: Trivy scanner
- **Communication**: gRPC (executor), WebSocket (frontend)

## Configuration

Key environment variables:
- `USE_K8S`: Enable Kubernetes integration
- `K8S_NAMESPACE`: Kubernetes namespace for builds
- `CONTAINER_REGISTRY_URL`: Docker registry URL
- `EXECUTOR_URL`: gRPC executor service address
- `JWT_SECRET`: Authentication secret
- `DATABASE_URL`: PostgreSQL connection string

## Development

### Prerequisites
- Go 1.25+
- PostgreSQL 15+
- Kubernetes cluster (for full functionality)
- Docker registry access

### Setup
```bash
# Install dependencies
go mod download

# Run migrations
psql -U postgres -d rl_arena -f migrations/*.sql

# Run server
go run cmd/server/main.go
```

### Build
```bash
go build -o rl-arena-backend ./cmd/server
```

## API Endpoints

See [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for detailed API reference.

### Core Endpoints
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `GET /api/v1/agents` - List agents
- `POST /api/v1/submissions` - Submit agent code
- `POST /api/v1/matches` - Create match
- `GET /api/v1/ws` - WebSocket connection

## Future Enhancements

- [ ] Replay upload/download functionality
- [ ] Tournament bracket system
- [ ] Multi-environment support expansion
- [ ] Advanced matchmaking options (custom rules)
- [ ] Agent performance analytics
