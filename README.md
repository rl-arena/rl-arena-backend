# RL-Arena Backend# RL-Arena Backend



[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-336791?style=flat&logo=postgresql)](https://www.postgresql.org)

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)



**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.



## 🎯 Features## 🎯 Features



- **User Authentication**: JWT-based secure authentication system- **User Authentication**: JWT-based secure authentication system

- **Agent Management**: Create, update, and manage AI agents- **Agent Management**: Create, update, and manage AI agents

- **Code Submission**: Upload Python agent code with version control- **Code Submission**: Upload Python agent code with version control

- **Automated Builds**: Kubernetes-based Docker image builds with Kaniko- **Automated Builds**: Kubernetes-based Docker image builds with Kaniko

- **Auto-Matchmaking**: ELO-based automatic opponent matching- **Auto-Matchmaking**: ELO-based automatic opponent matching

- **Match System**: Execute matches between agents via Executor service- **Match System**: Execute matches between agents via Executor service

- **ELO Rating**: Chess-like rating system for competitive rankings- **ELO Rating**: Chess-like rating system for competitive rankings

- **Replay System**: Download match replays in JSON or HTML format (Kaggle-style)- **Replay System**: Download match replays in JSON or HTML format (Kaggle-style)

- **Leaderboard**: Real-time rankings by ELO and environment- **Leaderboard**: Real-time rankings by ELO and environment

- **Real-time Monitoring**: Kubernetes Watch API + WebSocket notifications- **Real-time Monitoring**: Kubernetes Watch API + WebSocket notifications

- **Security Scanning**: Trivy vulnerability scanning for all images- **Security Scanning**: Trivy vulnerability scanning for all images

- **RESTful API**: Well-structured endpoints with comprehensive error handling- **RESTful API**: Well-structured endpoints with comprehensive error handling



## 🏗️ Architecture## 🏗️ Architecture



The backend follows a clean architecture pattern with clear separation of concerns:The backend follows a clean architecture pattern with clear separation of concerns:



```

cmd/

├── server/          # Application entry point## Quick Start```

internal/

├── api/cmd/

│   ├── handlers/    # HTTP request handlers

│   ├── middleware/  # Authentication, CORS, logging### Prerequisites├── server/          # Application entry point

│   └── router.go    # Route definitions

├── config/          # Configuration managementinternal/

├── models/          # Data models and structs

├── repository/      # Data access layer (PostgreSQL)- Go 1.25+├── api/

├── service/         # Business logic layer

└── websocket/       # WebSocket hub for real-time updates- PostgreSQL 15+│   ├── handlers/    # HTTP request handlers

pkg/

├── database/        # Database connection and utilities- Kubernetes cluster (optional for local dev)│   ├── middleware/  # Authentication, CORS, logging

├── executor/        # External executor service client

├── jwt/             # JWT token management│   └── router.go    # Route definitions

├── logger/          # Structured logging (Zap)

├── storage/         # File storage management### Installation├── config/          # Configuration management

└── validator/       # Request validation

```├── models/          # Data models and structs



### Key Components```bash├── repository/      # Data access layer (PostgreSQL)



- **REST API**: Built with Gin framework for high performance# Clone repository├── service/         # Business logic layer

- **Authentication**: JWT-based stateless authentication

- **Database**: PostgreSQL with proper migrationsgit clone https://github.com/rl-arena/rl-arena-backend.git└── queue/           # Background job processing

- **ELO System**: Chess-like rating system for competitive rankings

- **Executor Integration**: External service for running agent matchescd rl-arena-backendpkg/

- **File Storage**: Secure code submission and replay storage

├── database/        # Database connection and utilities

## 🚀 Quick Start

# Install dependencies├── executor/        # External executor service client

### Prerequisites

go mod download├── jwt/             # JWT token management

- **Go 1.25+**

- **PostgreSQL 15+**├── logger/          # Structured logging (Zap)

- **Docker & Docker Compose** (for containerized setup)

# Setup database├── storage/         # File storage management

### Environment Setup

createdb rl_arena├── utils/           # Common utilities

1. Clone the repository:

```bashcat migrations/*.sql | psql -U postgres -d rl_arena└── validator/       # Request validation

git clone https://github.com/rl-arena/rl-arena-backend.git

cd rl-arena-backend```

```

# Configure environment

2. Copy environment configuration:

```bashcp .env.example .env### Key Components

cp .env.example .env

```# Edit .env with your settings



3. Configure your `.env` file:- **REST API**: Built with Gin framework for high performance

```env

# Server Configuration# Run server- **Authentication**: JWT-based stateless authentication

PORT=8080

ENV=developmentgo run cmd/server/main.go- **Database**: PostgreSQL with proper migrations



# Database Configuration```- **ELO System**: Chess-like rating system for competitive rankings

DB_HOST=localhost

DB_PORT=5432- **Executor Integration**: External service for running agent matches

DB_USER=postgres

DB_PASSWORD=your_passwordServer starts on `http://localhost:8080`- **File Storage**: Secure code submission and replay storage

DB_NAME=rl_arena

DB_SSL_MODE=disable



# JWT Configuration## Documentation## 🚀 Quick Start

JWT_SECRET=your-super-secret-jwt-key

JWT_EXPIRY=24h



# CORS Configuration- 📖 [Architecture Overview](docs/ARCHITECTURE.md)### Prerequisites

CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

- 🚀 [Setup Guide](docs/SETUP.md)

# Storage Configuration

STORAGE_PATH=./storage- 🤖 [Auto-Matchmaking System](docs/AUTO_MATCHMAKING.md)- **Go 1.25+**



# Executor Service- 📡 [API Documentation](API_DOCUMENTATION.md)- **PostgreSQL 12+**

EXECUTOR_URL=http://localhost:9000

```- **Docker & Docker Compose** (for containerized setup)



### Installation Methods## Architecture



#### Option 1: Docker Compose (Recommended)### Environment Setup



```bash```

# Start all services (backend + database)

docker-compose up -d┌─────────────┐1. Clone the repository:



# View logs│   Frontend  │```bash

docker-compose logs -f backend

└──────┬──────┘git clone https://github.com/rl-arena/rl-arena-backend.git

# Stop services

docker-compose down       │ REST + WebSocketcd rl-arena-backend

```

┌──────▼──────────────────────┐```

#### Option 2: Local Development

│   Backend (Go + Gin)        │

1. Install dependencies:

```bash│   - API Handlers            │2. Copy environment configuration:

go mod download

```│   - Auto-Matchmaking        │```bash



2. Set up PostgreSQL database:│   - Build Monitor           │cp .env.example .env

```bash

# Create database│   - WebSocket Hub           │```

createdb rl_arena

└──────┬──────────────────────┘

# Run migrations

make migrate-up       │3. Configure your `.env` file:

```

┌──────▼──────────────────────┐```env

3. Run the server:

```bash│   PostgreSQL Database       │# Server Configuration

# Development mode with hot reload

make dev└─────────────────────────────┘PORT=8080



# Or build and runENV=development

make build

./bin/server┌─────────────────────────────┐

```

│   Kubernetes Cluster        │# Database Configuration

### Available Make Commands

│   - Kaniko Build Jobs       │DB_HOST=localhost

```bash

make help          # Show available commands│   - Security Scanning       │DB_PORT=5432

make build         # Build the application

make run           # Run the application└─────────────────────────────┘DB_USER=postgres

make dev           # Run with hot reload

make test          # Run testsDB_PASSWORD=your_password

make test-coverage # Run tests with coverage

make lint          # Run linters┌─────────────────────────────┐DB_NAME=rl_arena

make migrate-up    # Apply database migrations

make migrate-down  # Rollback database migrations│   Executor (Python gRPC)    │DB_SSL_MODE=disable

make docker-build  # Build Docker image

make clean         # Clean build artifacts│   - Match Simulation        │

```

│   - Result Reporting        │# JWT Configuration

## 📖 API Documentation

└─────────────────────────────┘JWT_SECRET=your-super-secret-jwt-key

### Key Endpoints

```JWT_EXPIRY=24h

**Authentication**

- `POST /api/v1/auth/register` - Register new user

- `POST /api/v1/auth/login` - Login and get JWT token

## Tech Stack# CORS Configuration

**Agents**

- `GET /api/v1/agents` - List all agentsCORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

- `POST /api/v1/agents` - Create new agent

- `GET /api/v1/agents/:id` - Get agent details- **Backend**: Go 1.25, Gin web framework

- `PUT /api/v1/agents/:id` - Update agent

- `DELETE /api/v1/agents/:id` - Delete agent- **Database**: PostgreSQL 15+# Storage Configuration



**Submissions**- **Container Orchestration**: Kubernetes (client-go v0.34.1)STORAGE_PATH=./storage

- `POST /api/v1/submissions` - Submit agent code

- `GET /api/v1/submissions/:id` - Get submission status- **Build Tool**: Kaniko (in-cluster Docker builds)

- `GET /api/v1/submissions/:id/build-status` - Check build status

- `POST /api/v1/submissions/:id/rebuild` - Retry failed build- **Security**: Trivy scanner# Executor Service



**Matches**- **Communication**: gRPC (executor), WebSocket (frontend)EXECUTOR_URL=http://localhost:9000

- `POST /api/v1/matches` - Create manual match

- `GET /api/v1/matches/:id` - Get match details- **Authentication**: JWT-based auth```

- `GET /api/v1/matches/:id/replay?format=json|html` - Download replay

- `GET /api/v1/matches/replays?agentId=X` - List replays (Watch feature)



**Leaderboard**## API Endpoints### Installation Methods

- `GET /api/v1/leaderboard` - Get rankings by environment



**WebSocket**

- `GET /api/v1/ws` - Real-time build and match notifications### Authentication#### Option 1: Docker Compose (Recommended)



For complete API documentation, see [API_DOCUMENTATION.md](./API_DOCUMENTATION.md).- `POST /api/v1/auth/register` - Register new user



## 🤖 Auto-Matchmaking System- `POST /api/v1/auth/login` - Login```bash



Agents are automatically matched after successful builds:# Start all services (backend + database)



1. **Build Success** → Agent joins matchmaking queue### Agentsdocker-compose up -d

2. **Every 30s** → Matching service finds suitable opponents

3. **ELO-based** → Matches agents with similar skill (±100 to ±500 ELO)- `GET /api/v1/agents` - List all agents

4. **Auto-Execute** → Match runs automatically via Executor

5. **Update Ratings** → ELO ratings updated after match completion- `GET /api/v1/agents/my` - Get user's agents# View logs



See [docs/AUTO_MATCHMAKING.md](docs/AUTO_MATCHMAKING.md) for details.- `POST /api/v1/agents` - Create agentdocker-compose logs -f backend



## 🗂️ Database Schema- `PUT /api/v1/agents/:id` - Update agent



The application uses PostgreSQL with the following main entities:- `DELETE /api/v1/agents/:id` - Delete agent# Stop services



- **Users**: User authentication and profilesdocker-compose down

- **Agents**: AI agents with ELO ratings and statistics

- **Submissions**: Code submissions with versioning### Submissions```

- **Matches**: Game matches between agents with results

- **Environments**: Different game environments (Pong, Tic-Tac-Toe, etc.)- `POST /api/v1/submissions` - Submit agent code



Database migrations are located in the `migrations/` directory.- `GET /api/v1/submissions/:id` - Get submission details#### Option 2: Local Development



## 🧪 Testing- `GET /api/v1/submissions/:id/build-status` - Check build status



```bash- `POST /api/v1/submissions/:id/rebuild` - Retry failed build1. Install dependencies:

# Run unit tests

make test```bash



# Run tests with coverage report### Matchesgo mod download

make test-coverage

- `POST /api/v1/matches` - Create match (manual)```

# Run integration tests

go test ./tests/integration/...- `GET /api/v1/matches/:id` - Get match details



# Run specific test- `GET /api/v1/matches/agent/:id` - List agent matches2. Set up PostgreSQL database:

go test ./internal/service -run TestELOService

``````bash



## 🚀 Deployment### WebSocket# Create database



### Production Environment Variables- `GET /api/v1/ws` - WebSocket connection for real-time updatescreatedb rl_arena



```env

ENV=production

DB_SSL_MODE=requireSee [API_DOCUMENTATION.md](API_DOCUMENTATION.md) for complete reference.# Run migrations

CORS_ALLOWED_ORIGINS=https://yourdomain.com

JWT_SECRET=your-production-secret-256-bit-minimummake migrate-up

```

## Auto-Matchmaking```

### Docker Production Build



```bash

# Build production imageAgents are automatically matched after successful build:3. Run the server:

docker build -t rl-arena-backend:latest .

```bash

# Run with production config

docker run -p 8080:8080 \1. **Build Success** → Agent joins matchmaking queue# Development mode with hot reload

  --env-file .env.production \

  rl-arena-backend:latest2. **Every 30s** → Matching service finds suitable opponentsmake dev

```

3. **ELO-based** → Matches agents with similar skill (±100 to ±500 ELO)

## 🤝 Contributing

4. **Auto-Execute** → Match runs automatically# Or build and run

1. **Fork the repository**

2. **Create a feature branch**: `git checkout -b feature/amazing-feature`5. **Update Ratings** → ELO ratings updated after matchmake build

3. **Commit your changes**: `git commit -m 'Add amazing feature'`

4. **Push to the branch**: `git push origin feature/amazing-feature`./bin/server

5. **Open a Pull Request**

See [AUTO_MATCHMAKING.md](docs/AUTO_MATCHMAKING.md) for details.```

### Code Style



- Follow Go conventions and use `gofmt`

- Run `make lint` before submitting## Environment Variables### Available Make Commands

- Write tests for new features

- Update documentation as needed



## 📚 Documentation```env```bash



- 📖 [Architecture Overview](docs/ARCHITECTURE.md)# Servermake help          # Show available commands

- 🚀 [Setup Guide](docs/SETUP.md)

- 🤖 [Auto-Matchmaking System](docs/AUTO_MATCHMAKING.md)PORT=8080make build         # Build the application

- 📡 [API Documentation](API_DOCUMENTATION.md)

ENV=developmentmake run           # Run the application

## 📄 License

make dev           # Run with hot reload

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

# Databasemake test          # Run tests

## 🔗 Related Projects

DATABASE_URL=postgresql://postgres:password@localhost:5432/rl_arenamake test-coverage # Run tests with coverage

- **RL-Arena Frontend**: React-based web interface

- **RL-Arena Executor**: Python gRPC service for running agent matchesmake lint          # Run linters

- **RL-Arena Env**: Python package for creating RL environments

# JWTmake migrate-up    # Apply database migrations

## 📞 Support

JWT_SECRET=your-secret-keymake migrate-down  # Rollback database migrations

For issues and questions:

- **GitHub Issues**: [rl-arena/rl-arena-backend/issues](https://github.com/rl-arena/rl-arena-backend/issues)JWT_EXPIRY=24hmake docker-build  # Build Docker image


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

- GitHub Issues: [github.com/rl-arena/rl-arena-backend/issues](https://github.com/rl-arena/rl-arena-backend/issues)## 🤖 Auto-Matchmaking System

Agents are automatically matched after successful builds:

1. **Build Success** → Agent joins matchmaking queue
2. **Every 30s** → Matching service finds suitable opponents
3. **ELO-based** → Matches agents with similar skill (±100 to ±500 ELO)
4. **Auto-Execute** → Match runs automatically via Executor
5. **Update Ratings** → ELO ratings updated after match completion

See [docs/AUTO_MATCHMAKING.md](docs/AUTO_MATCHMAKING.md) for details.

## 🗂️ Database Schema

The application uses PostgreSQL with the following main entities:

- **Users**: User authentication and profiles
- **Agents**: AI agents with ELO ratings and statistics
- **Submissions**: Code submissions with versioning
- **Matches**: Game matches between agents with results
- **Environments**: Different game environments (Pong, Tic-Tac-Toe, etc.)

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

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## � Documentation

- 📖 [Architecture Overview](docs/ARCHITECTURE.md)
- 🚀 [Setup Guide](docs/SETUP.md)
- 🤖 [Auto-Matchmaking System](docs/AUTO_MATCHMAKING.md)
- 📡 [API Documentation](API_DOCUMENTATION.md)

## �🔗 Related Projects

- **RL-Arena Frontend**: React-based web interface
- **RL-Arena Executor**: Python gRPC service for running agent matches
- **RL-Arena Env**: Python package for creating RL environments

## 📞 Support

For issues and questions:
- **GitHub Issues**: [rl-arena/rl-arena-backend/issues](https://github.com/rl-arena/rl-arena-backend/issues)
