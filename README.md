# RL Arena Backend# RL-Arena Backend

A reinforcement learning agent competition platform with automatic matchmaking, real-time monitoring, and Kubernetes-based Docker builds.
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev/)

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-yellow.svg)](https://opensource.org/licenses/Apache-2.0)

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)

[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org)

**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.

## 🎯 Features

## Features

- **User Authentication**: JWT-based secure authentication system

- ✅ **Agent Management**: Create and manage RL agents- **Agent Management**: Create, update, and manage AI agents

- ✅ **Automated Builds**: Kubernetes-based Docker image builds with Kaniko- **Code Submission**: Upload Python agent code with version control

- ✅ **Real-time Monitoring**: Kubernetes Watch API + WebSocket notifications- **Match System**: Execute matches between agents via Executor service

- ✅ **Auto-Matchmaking**: ELO-based automatic match creation- **ELO Rating**: Chess-like rating system for competitive rankings

- ✅ **Match Execution**: gRPC communication with executor service- **Leaderboard**: Real-time rankings by ELO and environment

- ✅ **ELO Rating System**: Competitive ranking with automatic updates- **RESTful API**: Well-structured endpoints with comprehensive error handling

- ✅ **Security Scanning**: Trivy vulnerability scanning for all images

- ✅ **Build Caching**: 24-hour cache for faster rebuilds## 🏗️ Architecture

- ✅ **Priority Queue**: Configurable build priorities

- ✅ **Retry Mechanism**: Automatic build retry on failureThe backend follows a clean architecture pattern with clear separation of concerns:



## Quick Start```

cmd/

### Prerequisites├── server/          # Application entry point

internal/

- Go 1.25+├── api/

- PostgreSQL 15+│   ├── handlers/    # HTTP request handlers

- Kubernetes cluster (optional for local dev)│   ├── middleware/  # Authentication, CORS, logging

│   └── router.go    # Route definitions

### Installation├── config/          # Configuration management

├── models/          # Data models and structs

```bash├── repository/      # Data access layer (PostgreSQL)

# Clone repository├── service/         # Business logic layer

git clone https://github.com/rl-arena/rl-arena-backend.git└── queue/           # Background job processing

cd rl-arena-backendpkg/

├── database/        # Database connection and utilities

# Install dependencies├── executor/        # External executor service client

go mod download├── jwt/             # JWT token management

├── logger/          # Structured logging (Zap)

# Setup database├── storage/         # File storage management

createdb rl_arena├── utils/           # Common utilities

cat migrations/*.sql | psql -U postgres -d rl_arena└── validator/       # Request validation

```

# Configure environment

cp .env.example .env### Key Components

# Edit .env with your settings

- **REST API**: Built with Gin framework for high performance

# Run server- **Authentication**: JWT-based stateless authentication

go run cmd/server/main.go- **Database**: PostgreSQL with proper migrations

```- **ELO System**: Chess-like rating system for competitive rankings

- **Executor Integration**: External service for running agent matches

Server starts on `http://localhost:8080`- **File Storage**: Secure code submission and replay storage



## Documentation## 🚀 Quick Start



- 📖 [Architecture Overview](docs/ARCHITECTURE.md)### Prerequisites

- 🚀 [Setup Guide](docs/SETUP.md)

- 🤖 [Auto-Matchmaking System](docs/AUTO_MATCHMAKING.md)- **Go 1.25+**

- 📡 [API Documentation](API_DOCUMENTATION.md)- **PostgreSQL 12+**

- **Docker & Docker Compose** (for containerized setup)

## Architecture

### Environment Setup

```

┌─────────────┐1. Clone the repository:

│   Frontend  │```bash

└──────┬──────┘git clone https://github.com/rl-arena/rl-arena-backend.git

       │ REST + WebSocketcd rl-arena-backend

┌──────▼──────────────────────┐```

│   Backend (Go + Gin)        │

│   - API Handlers            │2. Copy environment configuration:

│   - Auto-Matchmaking        │```bash

│   - Build Monitor           │cp .env.example .env

│   - WebSocket Hub           │```

└──────┬──────────────────────┘

       │3. Configure your `.env` file:

┌──────▼──────────────────────┐```env

│   PostgreSQL Database       │# Server Configuration

└─────────────────────────────┘PORT=8080

ENV=development

┌─────────────────────────────┐

│   Kubernetes Cluster        │# Database Configuration

│   - Kaniko Build Jobs       │DB_HOST=localhost

│   - Security Scanning       │DB_PORT=5432

└─────────────────────────────┘DB_USER=postgres

DB_PASSWORD=your_password

┌─────────────────────────────┐DB_NAME=rl_arena

│   Executor (Python gRPC)    │DB_SSL_MODE=disable

│   - Match Simulation        │

│   - Result Reporting        │# JWT Configuration

└─────────────────────────────┘JWT_SECRET=your-super-secret-jwt-key

```JWT_EXPIRY=24h



## Tech Stack# CORS Configuration

CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

- **Backend**: Go 1.25, Gin web framework

- **Database**: PostgreSQL 15+# Storage Configuration

- **Container Orchestration**: Kubernetes (client-go v0.34.1)STORAGE_PATH=./storage

- **Build Tool**: Kaniko (in-cluster Docker builds)

- **Security**: Trivy scanner# Executor Service

- **Communication**: gRPC (executor), WebSocket (frontend)EXECUTOR_URL=http://localhost:9000

- **Authentication**: JWT-based auth```



## API Endpoints### Installation Methods



### Authentication#### Option 1: Docker Compose (Recommended)

- `POST /api/v1/auth/register` - Register new user

- `POST /api/v1/auth/login` - Login```bash

# Start all services (backend + database)

### Agentsdocker-compose up -d

- `GET /api/v1/agents` - List all agents

- `GET /api/v1/agents/my` - Get user's agents# View logs

- `POST /api/v1/agents` - Create agentdocker-compose logs -f backend

- `PUT /api/v1/agents/:id` - Update agent

- `DELETE /api/v1/agents/:id` - Delete agent# Stop services

docker-compose down

### Submissions```

- `POST /api/v1/submissions` - Submit agent code

- `GET /api/v1/submissions/:id` - Get submission details#### Option 2: Local Development

- `GET /api/v1/submissions/:id/build-status` - Check build status

- `POST /api/v1/submissions/:id/rebuild` - Retry failed build1. Install dependencies:

```bash

### Matchesgo mod download

- `POST /api/v1/matches` - Create match (manual)```

- `GET /api/v1/matches/:id` - Get match details

- `GET /api/v1/matches/agent/:id` - List agent matches2. Set up PostgreSQL database:

```bash

### WebSocket# Create database

- `GET /api/v1/ws` - WebSocket connection for real-time updatescreatedb rl_arena



See [API_DOCUMENTATION.md](API_DOCUMENTATION.md) for complete reference.# Run migrations

make migrate-up

## Auto-Matchmaking```



Agents are automatically matched after successful build:3. Run the server:

```bash

1. **Build Success** → Agent joins matchmaking queue# Development mode with hot reload

2. **Every 30s** → Matching service finds suitable opponentsmake dev

3. **ELO-based** → Matches agents with similar skill (±100 to ±500 ELO)

4. **Auto-Execute** → Match runs automatically# Or build and run

5. **Update Ratings** → ELO ratings updated after matchmake build

./bin/server

See [AUTO_MATCHMAKING.md](docs/AUTO_MATCHMAKING.md) for details.```



## Environment Variables### Available Make Commands



```env```bash

# Servermake help          # Show available commands

PORT=8080make build         # Build the application

ENV=developmentmake run           # Run the application

make dev           # Run with hot reload

# Databasemake test          # Run tests

DATABASE_URL=postgresql://postgres:password@localhost:5432/rl_arenamake test-coverage # Run tests with coverage

make lint          # Run linters

# JWTmake migrate-up    # Apply database migrations

JWT_SECRET=your-secret-keymake migrate-down  # Rollback database migrations

JWT_EXPIRY=24hmake docker-build  # Build Docker image

make clean         # Clean build artifacts

# Kubernetes```

USE_K8S=true

K8S_NAMESPACE=rl-arena## 📖 API Usage

CONTAINER_REGISTRY_URL=docker.io/username

CONTAINER_REGISTRY_SECRET=regcred### Authentication



# ExecutorFirst, register a new user or login to get a JWT token:

EXECUTOR_URL=localhost:50051

```bash

# CORS# Register a new user

CORS_ALLOWED_ORIGINS=http://localhost:3000curl -X POST http://localhost:8080/api/v1/auth/register \

```  -H "Content-Type: application/json" \

  -d '{

## Development    "username": "alice",

    "email": "alice@example.com",

### Run Tests    "password": "securepassword",

    "fullName": "Alice Smith"

```bash  }'

go test ./...

```# Login to get JWT token

curl -X POST http://localhost:8080/api/v1/auth/login \

### Build  -H "Content-Type: application/json" \

  -d '{

```bash    "username": "alice",

go build -o rl-arena-backend ./cmd/server    "password": "securepassword"

```  }'

```

### Docker

### Agent Management

```bash

docker build -t rl-arena-backend .```bash

docker run -p 8080:8080 rl-arena-backend# Create a new agent

```curl -X POST http://localhost:8080/api/v1/agents \

  -H "Authorization: Bearer YOUR_JWT_TOKEN" \

### Kubernetes Deploy  -H "Content-Type: application/json" \

  -d '{

```bash    "name": "My RL Agent",

kubectl apply -f k8s/    "description": "A reinforcement learning agent for Tic-Tac-Toe",

```    "environmentId": "tictactoe"

  }'

## Contributing

# Get leaderboard

Contributions welcome! Please:curl http://localhost:8080/api/v1/leaderboard

```

1. Fork the repository

2. Create a feature branch### Code Submission

3. Make your changes

4. Submit a pull request```bash

# Submit agent code

## Licensecurl -X POST http://localhost:8080/api/v1/submissions \

  -H "Authorization: Bearer YOUR_JWT_TOKEN" \

MIT License - see [LICENSE](LICENSE) file for details.  -H "Content-Type: application/json" \

  -d '{

## Authors    "agentId": "agent-uuid",

    "codeUrl": "https://github.com/user/agent/archive/main.zip"

- RL Arena Team  }'

```

## Support

For complete API documentation, see [API_DOCUMENTATION.md](./API_DOCUMENTATION.md).

For issues and questions:

- GitHub Issues: [github.com/rl-arena/rl-arena-backend/issues](https://github.com/rl-arena/rl-arena-backend/issues)## 🗂️ Database Schema


The application uses PostgreSQL with the following main entities:

- **Users**: User authentication and profiles
- **Agents**: AI agents with ELO ratings and statistics
- **Submissions**: Code submissions with versioning
- **Matches**: Game matches between agents with results
- **Environments**: Different game environments (Tic-Tac-Toe, etc.)

Database migrations are located in the `migrations/` directory.

## 🧪 Testing

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

## 🚀 Deployment

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

## 🤝 Contributing

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

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Projects

- **RL-Arena Frontend**: [Coming Soon]
- **RL-Arena Executor**: External service for running agent matches
- **Agent Templates**: Sample agents for different environments

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/rl-arena/rl-arena-backend/issues)
- **Documentation**: [API Documentation](./API_DOCUMENTATION.md)
- **Wiki**: [Project Wiki](https://github.com/rl-arena/rl-arena-backend/wiki)
