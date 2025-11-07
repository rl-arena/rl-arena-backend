# RL-Arena Backend

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-yellow.svg)](https://opensource.org/licenses/Apache-2.0)

**RL-Arena Backend** is the core REST API server for the RL-Arena platform - a competitive reinforcement learning environment where AI agents battle against each other with ELO-based rankings.

## 🎯 Features

- **User Authentication**: JWT-based secure authentication system
- **Agent Management**: Create, update, and manage AI agents
- **Code Submission**: Upload Python agent code with version control
- **Match System**: Execute matches between agents via Executor service
- **ELO Rating**: Chess-like rating system for competitive rankings
- **Leaderboard**: Real-time rankings by ELO and environment
- **RESTful API**: Well-structured endpoints with comprehensive error handling

## 🏗️ Architecture

The backend follows a clean architecture pattern with clear separation of concerns:

```
cmd/
├── server/          # Application entry point
internal/
├── api/
│   ├── handlers/    # HTTP request handlers
│   ├── middleware/  # Authentication, CORS, logging
│   └── router.go    # Route definitions
├── config/          # Configuration management
├── models/          # Data models and structs
├── repository/      # Data access layer (PostgreSQL)
├── service/         # Business logic layer
└── queue/           # Background job processing
pkg/
├── database/        # Database connection and utilities
├── executor/        # External executor service client
├── jwt/             # JWT token management
├── logger/          # Structured logging (Zap)
├── storage/         # File storage management
├── utils/           # Common utilities
└── validator/       # Request validation
```

### Key Components

- **REST API**: Built with Gin framework for high performance
- **Authentication**: JWT-based stateless authentication
- **Database**: PostgreSQL with proper migrations
- **ELO System**: Chess-like rating system for competitive rankings
- **Executor Integration**: External service for running agent matches
- **File Storage**: Secure code submission and replay storage

## 🚀 Quick Start

### Prerequisites

- **Go 1.25+**
- **PostgreSQL 12+**
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

3. Configure your `.env` file:
```env
# Server Configuration
PORT=8080
ENV=development

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=rl_arena
DB_SSL_MODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRY=24h

# CORS Configuration
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Storage Configuration
STORAGE_PATH=./storage

# Executor Service
EXECUTOR_URL=http://localhost:9000
```

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

### Available Make Commands

```bash
make help          # Show available commands
make build         # Build the application
make run           # Run the application
make dev           # Run with hot reload
make test          # Run tests
make test-coverage # Run tests with coverage
make lint          # Run linters
make migrate-up    # Apply database migrations
make migrate-down  # Rollback database migrations
make docker-build  # Build Docker image
make clean         # Clean build artifacts
```

## 📖 API Usage

### Authentication

First, register a new user or login to get a JWT token:

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "securepassword",
    "fullName": "Alice Smith"
  }'

# Login to get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "password": "securepassword"
  }'
```

### Agent Management

```bash
# Create a new agent
curl -X POST http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My RL Agent",
    "description": "A reinforcement learning agent for Tic-Tac-Toe",
    "environmentId": "tictactoe"
  }'

# Get leaderboard
curl http://localhost:8080/api/v1/leaderboard
```

### Code Submission

```bash
# Submit agent code
curl -X POST http://localhost:8080/api/v1/submissions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "agentId": "agent-uuid",
    "codeUrl": "https://github.com/user/agent/archive/main.zip"
  }'
```

For complete API documentation, see [API_DOCUMENTATION.md](./API_DOCUMENTATION.md).

## 🗂️ Database Schema

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
