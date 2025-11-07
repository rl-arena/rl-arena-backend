# RL-Arena Backend

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.

##  Features

- **User Authentication**: JWT-based secure authentication system
- **Agent Management**: Create, update, and manage AI agents
- **Code Submission**: Upload Python agent code with version control
- **Automated Builds**: Kubernetes-based Docker image builds with Kaniko
- **Auto-Matchmaking**: ELO-based automatic opponent matching
- **Match System**: Execute matches between agents via Executor service
- **ELO Rating**: Chess-like rating system for competitive rankings
- **Replay System**: Download match replays in JSON or HTML format (Kaggle-style)
- **Leaderboard**: Real-time rankings by ELO and environment
- **Real-time Monitoring**: Kubernetes Watch API + WebSocket notifications
- **Security Scanning**: Trivy vulnerability scanning for all images
- **RESTful API**: Well-structured endpoints with comprehensive error handling

##  Architecture

The backend follows a clean architecture pattern with clear separation of concerns:

```
cmd/
 server/          # Application entry point

internal/
 api/
    handlers/    # HTTP request handlers
    middleware/  # Authentication, CORS, logging
    router.go    # Route definitions
 config/          # Configuration management
 models/          # Data models and structs
 repository/      # Data access layer (PostgreSQL)
 service/         # Business logic layer
 websocket/       # WebSocket hub for real-time updates

pkg/
 database/        # Database connection and utilities
 executor/        # External executor service client
─ jwt/             # JWT token management
 logger/          # Structured logging (Zap)
 storage/         # File storage management
 validator/       # Request validation
```

### Key Components

- **REST API**: Built with Gin framework for high performance
- **Authentication**: JWT-based stateless authentication
- **Database**: PostgreSQL with proper migrations
- **ELO System**: Chess-like rating system for competitive rankings
- **Executor Integration**: External service for running agent matches
- **File Storage**: Secure code submission and replay storage

##  Quick Start

### Prerequisites

- **Go 1.25+**
- **PostgreSQL 15+**
- **Docker & Docker Compose** (for containerized setup)

### Environment Setup

1. Clone the repository:

```bash
git clone https://github.com/rl-arena/rl-arena-backend.git
cd rl-arena-backend
```

2. Copy environment configuration:

```bash
cp .env.example .env
```

3. Configure your `.env` file with database credentials, JWT secret, etc.

### Installation Methods

#### Option 1: Docker Compose (Recommended)

```bash
# Start all services (backend + database)
docker-compose up -d

# View logs
docker-compose logs -f backend

# Stop services
docker-compose down
```

#### Option 2: Local Development

1. Install dependencies:

```bash
go mod download
```

2. Set up PostgreSQL database:

```bash
# Create database
createdb rl_arena

# Run migrations
make migrate-up
```

3. Run the server:

```bash
# Development mode with hot reload
make dev

# Or build and run
make build
./bin/server
```

##  API Documentation

### Key Endpoints

**Authentication**
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT token

**Agents**
- `GET /api/v1/agents` - List all agents
- `POST /api/v1/agents` - Create new agent
- `GET /api/v1/agents/:id` - Get agent details
- `PUT /api/v1/agents/:id` - Update agent
- `DELETE /api/v1/agents/:id` - Delete agent

**Submissions**
- `POST /api/v1/submissions` - Submit agent code
- `GET /api/v1/submissions/:id` - Get submission status
- `GET /api/v1/submissions/:id/build-status` - Check build status
- `POST /api/v1/submissions/:id/rebuild` - Retry failed build

**Matches**
- `POST /api/v1/matches` - Create manual match
- `GET /api/v1/matches/:id` - Get match details
- `GET /api/v1/matches/:id/replay?format=json|html` - Download replay
- `GET /api/v1/matches/replays?agentId=X` - List replays (Watch feature)

**Leaderboard**
- `GET /api/v1/leaderboard` - Get rankings by environment

**WebSocket**
- `GET /api/v1/ws` - Real-time build and match notifications

For complete API documentation, see [API_DOCUMENTATION.md](./API_DOCUMENTATION.md).

##  Auto-Matchmaking System

Agents are automatically matched after successful builds:

1. **Build Success**  Agent joins matchmaking queue
2. **Every 30s**  Matching service finds suitable opponents
3. **ELO-based**  Matches agents with similar skill (100 to 500 ELO)
4. **Auto-Execute**  Match runs automatically via Executor
5. **Update Ratings**  ELO ratings updated after match completion

See [docs/AUTO_MATCHMAKING.md](docs/AUTO_MATCHMAKING.md) for details.

##  Database Schema

The application uses PostgreSQL with the following main entities:

- **Users**: User authentication and profiles
- **Agents**: AI agents with ELO ratings and statistics
- **Submissions**: Code submissions with versioning
- **Matches**: Game matches between agents with results
- **Environments**: Different game environments (Pong, Tic-Tac-Toe, etc.)

Database migrations are located in the `migrations/` directory.

##  Testing

```bash
# Run unit tests
make test

# Run tests with coverage report
make test-coverage

# Run integration tests
go test ./tests/integration/...

# Run specific test
go test ./internal/service -run TestELOService
```

##  Deployment

### Production Environment Variables

```env
ENV=production
DB_SSL_MODE=require
CORS_ALLOWED_ORIGINS=https://yourdomain.com
JWT_SECRET=your-production-secret-256-bit-minimum
```

### Docker Production Build

```bash
# Build production image
docker build -t rl-arena-backend:latest .

# Run with production config
docker run -p 8080:8080 \
  --env-file .env.production \
  rl-arena-backend:latest
```

##  Contributing

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Commit your changes**: `git commit -m 'Add amazing feature'`
4. **Push to the branch**: `git push origin feature/amazing-feature`
5. **Open a Pull Request**

### Code Style

- Follow Go conventions and use `gofmt`
- Run `make lint` before submitting
- Write tests for new features
- Update documentation as needed

##  Documentation

-  [Architecture Overview](docs/ARCHITECTURE.md)
-  [Setup Guide](docs/SETUP.md)
-  [Auto-Matchmaking System](docs/AUTO_MATCHMAKING.md)
-  [API Documentation](API_DOCUMENTATION.md)

##  License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

##  Related Projects

- **RL-Arena Frontend**: React-based web interface
- **RL-Arena Executor**: Python gRPC service for running agent matches
- **RL-Arena Env**: Python package for creating RL environments

##  Support

For issues and questions:
- **GitHub Issues**: [rl-arena/rl-arena-backend/issues](https://github.com/rl-arena/rl-arena-backend/issues)
