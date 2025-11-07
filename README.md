# RL-Arena Backend

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.

##  Features

- **User Authentication**: JWT-based secure authentication system
- **Agent Management**: Create, update, and manage AI agents
- **Code Submission**: Upload Python agent code with version control
- **Submission Quota**: Daily submission limit (5 submissions/day per agent)
- **Automated Builds**: Kubernetes-based Docker image builds with Kaniko
- **Auto-Matchmaking**: ELO-based automatic opponent matching
- **Match System**: Execute matches between agents via Executor service
- **ELO Rating**: Chess-like rating system for competitive rankings
- **Replay System**: Download match replays in JSON or HTML format (Kaggle-style)
- **Public/Private Leaderboard**: Separate rankings for real-time and post-competition (Kaggle-style)
- **Agent Statistics**: Opponent-based win/loss records and performance analytics
- **API Rate Limiting**: Token Bucket algorithm-based rate limiting for all endpoints
  - **In-Memory**: Fast, single-instance rate limiting (development)
  - **Redis**: Distributed rate limiting for multi-instance deployments (production)
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
- **Redis 7+** (optional, for distributed rate limiting)
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
- `GET /api/v1/agents/:id/stats` - Get agent opponent statistics (win/loss records)
- `PUT /api/v1/agents/:id` - Update agent
- `DELETE /api/v1/agents/:id` - Delete agent

**Submissions** (Rate Limited: 5 submissions/day per agent)
- `POST /api/v1/submissions` - Submit agent code (daily quota: 5)
- `GET /api/v1/submissions/:id` - Get submission status
- `GET /api/v1/submissions/:id/build-status` - Check build status
- `POST /api/v1/submissions/:id/rebuild` - Retry failed build

**Matches**
- `POST /api/v1/matches` - Create manual match
- `GET /api/v1/matches/:id` - Get match details
- `GET /api/v1/matches/:id/replay?format=json|html` - Download replay
- `GET /api/v1/matches/replays?agentId=X` - List replays (Watch feature)

**Leaderboard**
- `GET /api/v1/leaderboard` - Get all rankings
- `GET /api/v1/leaderboard/:envId?type=public|private|all` - Get environment-specific leaderboard
  - `type=public` (default): Public leaderboard (real-time rankings)
  - `type=private`: Private leaderboard (post-competition rankings)
  - `type=all`: All matches regardless of publicity

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

##  Distributed Rate Limiting

RL-Arena supports **Redis-based distributed rate limiting** for production multi-instance deployments:

### Features

- **Horizontal Scaling**: Consistent rate limiting across multiple API instances
- **Token Bucket Algorithm**: Smooth traffic control with automatic refill
- **Atomic Operations**: Lua scripts prevent race conditions
- **Fail-Open Strategy**: Maintains service availability if Redis is temporarily down
- **HTTP Headers**: X-RateLimit-* headers inform clients about limits

### Quick Setup

1. Add Redis configuration to `.env`:

```env
REDIS_URL=redis://localhost:6379
```

2. Start Redis with Docker:

```bash
docker-compose up -d redis
```

3. The backend automatically uses Redis rate limiting when `REDIS_URL` is configured.

### Rate Limit Presets

| Endpoint | Limit | Window | Key |
|----------|-------|--------|-----|
| Authentication | 5 requests | 1 minute | IP address |
| Code Submission | 5 requests | 1 minute | User ID |
| Match Creation | 10 requests | 1 minute | User ID |
| Replay Download | 20 requests | 1 minute | IP/User |

For detailed documentation, see [docs/REDIS_RATE_LIMITER.md](docs/REDIS_RATE_LIMITER.md).

##  Provisional ELO Rating System

RL-Arena implements a **Kaggle-style provisional rating system** to provide more accurate ratings for players at different experience levels:

### Dynamic K-Factors

| Experience Level | Match Count | K-Factor | Purpose |
|-----------------|-------------|----------|---------|
| **Provisional** | < 10 matches | **40** | Faster convergence for new players |
| **Intermediate** | 10-20 matches | **32** | Moderate rating adjustments |
| **Established** | > 20 matches | **24** | Stable ratings for veterans |

### Benefits

1. **Faster Convergence**: New players reach their true skill level quicker (within 10 matches)
2. **Better Matchmaking**: More accurate ELO means fairer matches from the start
3. **Rating Stability**: Veteran players experience less rating volatility
4. **Fairness**: Similar to competitive chess and Kaggle competitions

### Example

```
Player A (5 matches, ELO 1200) beats Player B (50 matches, ELO 1200)
  - Player A: +20 ELO (K=40)
  - Player B: -12 ELO (K=24)

Player A gains more rating due to provisional status, while Player B's 
established rating changes less dramatically.
```

For implementation details, see `internal/service/elo_service.go`.

##  API Documentation

RL-Arena provides **interactive API documentation** powered by Swagger/OpenAPI:

### Access Swagger UI

Once the server is running, visit:

```
http://localhost:8080/swagger/index.html
```

### Features

- **Interactive Testing**: Try out API endpoints directly from the browser
- **Authentication Support**: Test authenticated endpoints with JWT tokens
- **Schema Validation**: View request/response models and validation rules
- **Auto-Generated**: Documentation is automatically generated from code annotations

### Example Usage

1. **Start the server**:
   ```bash
   go run cmd/server/main.go
   ```

2. **Open Swagger UI** in your browser:
   ```
   http://localhost:8080/swagger/index.html
   ```

3. **Authenticate** (for protected endpoints):
   - Click "Authorize" button
   - Enter: `Bearer <your-jwt-token>`
   - Click "Authorize"

4. **Test endpoints**:
   - Expand any endpoint
   - Click "Try it out"
   - Fill in parameters
   - Click "Execute"

### Update Documentation

When you add or modify API endpoints:

```bash
# Regenerate Swagger docs
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Rebuild the server
go build -o bin/server.exe ./cmd/server
```

For static documentation, see [API_DOCUMENTATION.md](API_DOCUMENTATION.md).

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
-  [Redis Rate Limiter](docs/REDIS_RATE_LIMITER.md)
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
