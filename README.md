# RL-Arena Backend

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.

##  Features

- **User Authentication**: JWT-based secure authentication system
- **Agent Management**: Create, update, and manage AI agents
- **Code Submission**: Upload Python agent code with automated Docker builds
- **Match Rate Limiting**: 5-minute cooldown between matches, 100 matches/day per agent
- **Automated Builds**: Kubernetes-based Docker image builds with Kaniko
- **Auto-Matchmaking**: Intelligent ELO-based automatic opponent matching every 30 seconds
- **Match System**: Execute matches between agents via gRPC Executor service
- **ELO Rating**: Chess-like rating system with provisional rating support
- **Replay System**: Download match replays in JSON or HTML format
- **Leaderboard**: Real-time ELO-based rankings with agent statistics
- **RESTful API**: Well-structured endpoints with Swagger documentation
- **WebSocket**: Real-time build and match notifications
- **Security**: Code validation and Docker image vulnerability scanning

##  Architecture

```
cmd/
  server/          # Application entry point

internal/
  api/
    handlers/      # HTTP request handlers
    middleware/    # Authentication, CORS, rate limiting
    router.go      # Route definitions
  config/          # Configuration management
  models/          # Data models and structs
  repository/      # Data access layer (PostgreSQL)
  service/         # Business logic layer
  websocket/       # WebSocket hub for real-time updates

pkg/
  database/        # Database connection utilities
  executor/        # gRPC client for executor service
  jwt/             # JWT token management
  logger/          # Structured logging (Zap)
  storage/         # File storage management
```

##  Quick Start

### Prerequisites

- **Go 1.25+**
- **PostgreSQL 15+**
- **Docker & Docker Compose**

### Installation

1. Clone the repository:
```bash
git clone https://github.com/rl-arena/rl-arena-backend.git
cd rl-arena-backend
```

2. Copy environment configuration:
```bash
cp .env.example .env
```

3. Start with Docker Compose (recommended):
```bash
docker-compose up -d
```

4. Or run locally:
```bash
# Install dependencies
go mod download

# Run migrations
make migrate-up

# Start server
go run cmd/server/main.go
```

Visit `http://localhost:8080/swagger/index.html` for API documentation.

##  API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT token

### Agents
- `GET /api/v1/agents` - List all agents
- `POST /api/v1/agents` - Create new agent
- `GET /api/v1/agents/:id` - Get agent details
- `GET /api/v1/agents/:id/stats` - Get agent statistics
- `PUT /api/v1/agents/:id` - Update agent
- `DELETE /api/v1/agents/:id` - Delete agent

### Submissions
- `POST /api/v1/submissions` - Submit agent code
- `GET /api/v1/submissions/:id` - Get submission status
- `GET /api/v1/submissions/:id/build-status` - Check build status
- `GET /api/v1/submissions/:id/build-logs` - Get build logs
- `POST /api/v1/submissions/:id/rebuild` - Retry failed build

### Matches
- `POST /api/v1/matches` - Create manual match
- `GET /api/v1/matches` - List matches
- `GET /api/v1/matches/:id` - Get match details
- `GET /api/v1/matches/:id/replay?format=json|html` - Download replay
- `GET /api/v1/matches/:id/replay-url` - Get replay URL
- `GET /api/v1/matches/agent/:agentId` - List agent matches

### Leaderboard
- `GET /api/v1/leaderboard` - Get all rankings
- `GET /api/v1/leaderboard/environment/:envId` - Get environment leaderboard

### WebSocket
- `GET /api/v1/ws` - Real-time build and match notifications

For complete documentation, visit `/swagger/index.html` when server is running.

##  Auto-Matchmaking System

The matchmaking system automatically creates matches for eligible agents:

1. **Build Success**  Agent joins matchmaking queue
2. **Every 30 seconds**  Matching service finds suitable opponents
3. **ELO-based matching**  Selects agents within 100-500 ELO difference
4. **Rate limiting**  5-minute cooldown, max 100 matches/day
5. **Auto-execution**  Match runs automatically via Executor
6. **Rating update**  ELO ratings updated after match completion

**Match Rate Limits:**
- **Cooldown**: 5 minutes between matches per agent
- **Daily limit**: 100 matches per day per agent
- **Automatic reset**: Daily counter resets at midnight UTC

##  ELO Rating System

RL-Arena implements a provisional rating system similar to chess and Kaggle:

| Experience | Matches | K-Factor | Purpose |
|-----------|---------|----------|---------|
| Provisional | < 10 | 40 | Fast convergence for new agents |
| Intermediate | 10-20 | 32 | Moderate adjustments |
| Established | > 20 | 24 | Stable ratings |

**Benefits:**
- New agents reach true skill level quickly (within 10 matches)
- Better matchmaking accuracy from the start
- Veteran agents experience less rating volatility
- Fair and competitive ranking system

##  Database

PostgreSQL database with migrations in `migrations/` directory:

**Main Tables:**
- `users` - User accounts and authentication
- `agents` - AI agents with ELO ratings
- `submissions` - Code submissions with build status
- `matches` - Match history and results
- `agent_match_stats` - Rate limiting and match statistics
- `matchmaking_queue` - Agents waiting for matches

**Reset Database:**
```bash
# Using provided script
docker exec -i rl-arena-backend-db-1 psql -U postgres -d rl_arena < scripts/reset_database.sql
```

##  Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific tests
go test ./internal/service -run TestELOService
```

##  Deployment

### Docker Production Build

```bash
# Build image
docker build -t rl-arena-backend:latest .

# Run container
docker run -p 8080:8080 --env-file .env rl-arena-backend:latest
```

### Environment Variables

```env
ENV=production
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=rl_arena
DB_SSL_MODE=disable

JWT_SECRET=your-secret-key-min-256-bits

EXECUTOR_GRPC_ADDRESS=localhost:50051

CORS_ALLOWED_ORIGINS=http://localhost:5173
```

##  Documentation

- [API Documentation](API_DOCUMENTATION.md) - Complete API reference
- [Architecture](docs/ARCHITECTURE.md) - System architecture overview
- [Rate Limiting](docs/RATE_LIMITING.md) - Match rate limiting details
- [Setup Guide](docs/SETUP.md) - Detailed setup instructions
- [Quick Start](docs/QUICK_START.md) - Get started quickly

##  Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

##  License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

##  Related Projects

- [rl-arena-web](https://github.com/rl-arena/rl-arena-web) - React web frontend
- [rl-arena-executor](https://github.com/rl-arena/rl-arena-executor) - Python gRPC match executor
- [rl-arena-env](https://github.com/rl-arena/rl-arena-env) - RL environment library
