# Setup Guide

## Prerequisites

- **Go**: 1.25 or higher
- **PostgreSQL**: 15 or higher
- **Kubernetes Cluster**: For build functionality (optional for development)
- **Docker Registry**: Docker Hub or private registry

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/rl-arena/rl-arena-backend.git
cd rl-arena-backend
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Database Setup

Create PostgreSQL database:

```bash
createdb rl_arena
```

Run migrations:

```bash
psql -U postgres -d rl_arena -f migrations/001_initial_schema.sql
psql -U postgres -d rl_arena -f migrations/002_add_pong_environment.sql
psql -U postgres -d rl_arena -f migrations/003_add_docker_image.sql
psql -U postgres -d rl_arena -f migrations/004_add_retry_fields.sql
psql -U postgres -d rl_arena -f migrations/005_add_security_scan.sql
psql -U postgres -d rl_arena -f migrations/006_add_priority_queue.sql
psql -U postgres -d rl_arena -f migrations/007_add_environment_to_submissions.sql
psql -U postgres -d rl_arena -f migrations/008_add_matchmaking.sql
```

Or use a single command:

```bash
cat migrations/*.sql | psql -U postgres -d rl_arena
```

### 4. Environment Configuration

Copy example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgresql://postgres:password@localhost:5432/rl_arena?sslmode=disable

# JWT
JWT_SECRET=your-secret-key-change-this
JWT_EXPIRY=24h

# Storage
STORAGE_PATH=./storage

# Kubernetes (optional for local dev)
USE_K8S=false
K8S_NAMESPACE=rl-arena
CONTAINER_REGISTRY_URL=docker.io/your-username
CONTAINER_REGISTRY_SECRET=regcred

# Executor
EXECUTOR_URL=localhost:50051

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
```

### 5. Run Server

```bash
go run cmd/server/main.go
```

The server will start on `http://localhost:8080`.

## Kubernetes Setup (Production)

### 1. Create Namespace

```bash
kubectl apply -f k8s/namespace.yaml
```

### 2. Configure Secrets

Create registry secret for pulling images:

```bash
kubectl create secret docker-registry regcred \
  --docker-server=docker.io \
  --docker-username=your-username \
  --docker-password=your-password \
  --docker-email=your-email \
  -n rl-arena
```

Create application secrets:

```bash
kubectl apply -f k8s/secret.yaml
```

### 3. Deploy PostgreSQL

```bash
kubectl apply -f k8s/postgres-statefulset.yaml
```

### 4. Deploy Backend

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### 5. Configure Ingress (optional)

```bash
kubectl apply -f k8s/ingress.yaml
```

## Docker Compose (Local Development)

Run entire stack with Docker Compose:

```bash
docker-compose up -d
```

This starts:
- PostgreSQL database
- Backend API server
- Mock executor service

## Testing

### Run Tests

```bash
go test ./...
```

### Test API

```bash
# Register user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"password123"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Health check
curl http://localhost:8080/health
```

## Development Tips

### Enable Debug Logging

Set in `.env`:

```env
ENV=development
LOG_LEVEL=debug
```

### Disable Kubernetes (Local Dev)

```env
USE_K8S=false
```

When Kubernetes is disabled:
- Build jobs won't be created
- Submissions will be marked as "pending"
- Auto-matchmaking will still work

### Mock Executor

Use the Python mock executor for testing:

```bash
cd scripts
python mock_executor.py
```

## Troubleshooting

### Database Connection Issues

Check PostgreSQL is running:

```bash
pg_isready -h localhost -p 5432
```

Verify connection string in `.env`.

### Kubernetes Build Jobs Stuck

Check job status:

```bash
kubectl get jobs -n rl-arena
kubectl describe job <job-name> -n rl-arena
```

View pod logs:

```bash
kubectl logs <pod-name> -n rl-arena
```

### Port Already in Use

Change port in `.env`:

```env
PORT=8081
```

## Next Steps

- Read [ARCHITECTURE.md](./ARCHITECTURE.md) for system overview
- Check [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) for API reference
- Review [AUTO_MATCHMAKING.md](./AUTO_MATCHMAKING.md) for matchmaking details
