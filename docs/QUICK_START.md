# Quick Start Guide - Database Setup

## Option 1: Docker Compose (Recommended)

### Prerequisites
- Docker Desktop installed and running

### Steps

1. **Start Docker Desktop**
   - Windows: Search for "Docker Desktop" and run it
   - Wait for Docker to fully start (whale icon in system tray)

2. **Start PostgreSQL and Redis**
   ```bash
   docker-compose up -d db redis
   ```

3. **Verify containers are running**
   ```bash
   docker-compose ps
   ```

   You should see:
   - `rl-arena-backend-db-1` - PostgreSQL
   - `rl-arena-backend-redis-1` - Redis

4. **Run database migrations**
   ```bash
   # Install golang-migrate (one-time)
   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   
   # Run migrations
   migrate -path migrations -database "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" up
   ```

5. **Start the backend server**
   ```bash
   go run cmd/server/main.go
   ```

6. **Access Swagger UI**
   ```
   http://localhost:8080/swagger/index.html
   ```

---

## Option 2: Local PostgreSQL Installation

### Prerequisites
- PostgreSQL 15+ installed locally
- PostgreSQL service running

### Steps

1. **Create database and user**
   ```bash
   # Connect to PostgreSQL
   psql -U postgres
   
   # Create database
   CREATE DATABASE rl_arena;
   
   # Create user (if needed)
   CREATE USER postgres WITH PASSWORD 'password';
   
   # Grant privileges
   GRANT ALL PRIVILEGES ON DATABASE rl_arena TO postgres;
   
   # Exit
   \q
   ```

2. **Update .env file**
   ```bash
   DATABASE_URL=postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable
   REDIS_URL=redis://localhost:6379
   ```

3. **Start Redis (if needed)**
   ```bash
   # Option A: Docker
   docker run -d -p 6379:6379 redis:7-alpine
   
   # Option B: Windows Redis
   # Download from: https://github.com/microsoftarchive/redis/releases
   # Run: redis-server.exe
   ```

4. **Run database migrations**
   ```bash
   migrate -path migrations -database "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" up
   ```

5. **Start the backend server**
   ```bash
   go run cmd/server/main.go
   ```

---

## Option 3: Quick Test Without Database

For quick testing without setting up a database:

1. **Use SQLite (requires code changes)**
   - Not recommended for production
   - Requires modifying database driver

2. **Use Docker Compose for everything**
   ```bash
   # Start all services including backend
   docker-compose up
   ```
   
   This will:
   - Start PostgreSQL
   - Start Redis
   - Build and run the backend
   - Automatically run migrations

---

## Troubleshooting

### Error: "password authentication failed for user postgres"

**Solution 1**: Check password in `.env`
```bash
# Make sure .env has:
DATABASE_URL=postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable
```

**Solution 2**: Reset PostgreSQL password
```bash
# Windows (run as Administrator)
# Edit: C:\Program Files\PostgreSQL\15\data\pg_hba.conf
# Change: md5 -> trust
# Restart PostgreSQL service
# Connect and reset password:
psql -U postgres
ALTER USER postgres WITH PASSWORD 'password';
# Change pg_hba.conf back to md5
# Restart PostgreSQL
```

### Error: "database rl_arena does not exist"

**Solution**: Create the database
```bash
psql -U postgres -c "CREATE DATABASE rl_arena;"
```

### Error: "dial tcp 127.0.0.1:5432: connect: connection refused"

**Solution**: PostgreSQL is not running
```bash
# Windows: Start PostgreSQL service
# Services -> postgresql-x64-15 -> Start

# Or use Docker:
docker-compose up -d db
```

### Error: "dial tcp 127.0.0.1:6379: connect: connection refused"

**Solution**: Redis is not running
```bash
# Use Docker:
docker run -d -p 6379:6379 redis:7-alpine

# Or:
docker-compose up -d redis
```

---

## Verify Setup

### 1. Check PostgreSQL
```bash
psql -U postgres -d rl_arena -c "SELECT version();"
```

### 2. Check Redis
```bash
# If you have redis-cli installed:
redis-cli ping
# Should return: PONG

# Or use Docker:
docker exec -it rl-arena-backend-redis-1 redis-cli ping
```

### 3. Check Backend
```bash
curl http://localhost:8080/health
# Should return: {"service":"rl-arena-backend","status":"ok"}
```

### 4. Check Swagger
Open browser: `http://localhost:8080/swagger/index.html`

---

## Database Migrations

### Apply all migrations
```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" up
```

### Rollback one migration
```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" down 1
```

### Check migration status
```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/rl_arena?sslmode=disable" version
```

---

## Production Deployment

For production, use environment variables:

```bash
# .env.production
PORT=8080
ENV=production
DATABASE_URL=postgres://user:pass@prod-db:5432/rl_arena?sslmode=require
REDIS_URL=redis://prod-redis:6379
JWT_SECRET=<generate-strong-secret-256-bit>
CORS_ALLOWED_ORIGINS=https://yourdomain.com
```

Generate strong JWT secret:
```bash
# PowerShell
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))

# Or use online tool (for development only):
# https://randomkeygen.com/
```

---

## Next Steps

1. ✅ Set up database (PostgreSQL + Redis)
2. ✅ Run migrations
3. ✅ Start backend server
4. ✅ Test with Swagger UI
5. 🚀 Start developing!

For API documentation, see:
- Swagger UI: http://localhost:8080/swagger/index.html
- Markdown: [API_DOCUMENTATION.md](../API_DOCUMENTATION.md)
- Swagger Guide: [SWAGGER.md](SWAGGER.md)
