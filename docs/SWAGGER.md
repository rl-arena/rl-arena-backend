# OpenAPI/Swagger Documentation Guide

## Overview

RL-Arena Backend uses **Swagger/OpenAPI 2.0** for automatic API documentation generation. This provides an interactive, web-based interface for exploring and testing the API.

## Quick Start

### 1. Access Swagger UI

Start the server and navigate to:

```
http://localhost:8080/swagger/index.html
```

### 2. Explore Endpoints

The Swagger UI displays all available endpoints organized by tags:

- **auth**: Authentication (login, register)
- **agents**: AI agent management
- **leaderboard**: ELO rankings
- **system**: Health checks

### 3. Test Authenticated Endpoints

Many endpoints require authentication:

1. **Get a token** by calling `/auth/login` or `/auth/register`
2. **Click "Authorize"** button (🔒 icon) at the top right
3. **Enter**: `Bearer <your-jwt-token>` (replace `<your-jwt-token>` with actual token)
4. **Click "Authorize"**
5. Now you can test protected endpoints

## API Annotations

### Endpoint Annotation Example

```go
// CreateAgent godoc
// @Summary Create a new agent
// @Description Create a new AI agent for the authenticated user
// @Tags agents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateAgentRequest true "Agent creation details"
// @Success 201 {object} map[string]interface{} "Created agent"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /agents [post]
func (h *AgentHandler) CreateAgent(c *gin.Context) {
    // Implementation
}
```

### Main API Configuration

In `cmd/server/main.go`:

```go
// @title RL-Arena API
// @version 1.0
// @description REST API server for RL-Arena
// @termsOfService http://swagger.io/terms/

// @contact.name RL-Arena Support
// @contact.url https://github.com/rl-arena/rl-arena-backend/issues
// @contact.email support@rl-arena.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
```

## Regenerating Documentation

### When to Regenerate

Regenerate Swagger docs whenever you:
- Add new endpoints
- Modify request/response structures
- Change endpoint descriptions
- Update API metadata

### How to Regenerate

```bash
# Install swag CLI (one-time)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Rebuild the server
go build -o bin/server.exe ./cmd/server
```

### Verify Generation

Check for these files in the `docs/` directory:
- `docs.go` - Generated Go code
- `swagger.json` - OpenAPI JSON spec
- `swagger.yaml` - OpenAPI YAML spec

## Common Annotation Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `@Summary` | Short endpoint description | `@Summary Get agent by ID` |
| `@Description` | Detailed description | `@Description Get detailed information about a specific agent` |
| `@Tags` | Group endpoints | `@Tags agents` |
| `@Accept` | Request content type | `@Accept json` |
| `@Produce` | Response content type | `@Produce json` |
| `@Param` | Request parameter | `@Param id path string true "Agent ID"` |
| `@Success` | Success response | `@Success 200 {object} Agent` |
| `@Failure` | Error response | `@Failure 404 {object} map[string]string` |
| `@Router` | Route path and method | `@Router /agents/{id} [get]` |
| `@Security` | Required auth | `@Security BearerAuth` |

## Parameter Types

### Path Parameter
```go
// @Param id path string true "Agent ID"
```

### Query Parameter
```go
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
```

### Body Parameter
```go
// @Param request body models.CreateAgentRequest true "Agent creation details"
```

### Header Parameter
```go
// @Param Authorization header string true "Bearer token"
```

## Response Types

### Object Response
```go
// @Success 200 {object} models.Agent "Agent details"
```

### Array Response
```go
// @Success 200 {array} models.Agent "List of agents"
```

### Map Response
```go
// @Success 200 {object} map[string]interface{} "Dynamic response"
```

### String Map
```go
// @Failure 400 {object} map[string]string "Error message"
```

## Best Practices

### 1. Use Descriptive Summaries
```go
// Good
// @Summary Get agent statistics including win/loss records
// Bad
// @Summary Get stats
```

### 2. Document All Parameters
```go
// Good
// @Param id path string true "Agent ID (UUID format)"
// @Param limit query int false "Number of results (max 100)" default(20)

// Bad
// @Param id path string true "ID"
```

### 3. Include All Response Codes
```go
// Good
// @Success 200 {object} Agent
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Agent not found"
// @Failure 500 {object} map[string]string "Internal server error"

// Bad
// @Success 200 {object} Agent
```

### 4. Use Proper Data Models
```go
// Good - Define typed struct
type AuthResponse struct {
    Token string   `json:"token"`
    User  UserInfo `json:"user"`
}

// Bad - Use generic map
// @Success 200 {object} gin.H
```

### 5. Add Security Where Needed
```go
// Protected endpoint
// @Security BearerAuth
func (h *Handler) CreateAgent(c *gin.Context) { }

// Public endpoint (no @Security)
func (h *Handler) GetLeaderboard(c *gin.Context) { }
```

## Troubleshooting

### Issue: "cannot find type definition"

**Problem**: Swag can't parse `gin.H` or other generic types

**Solution**: Define explicit structs
```go
// Before
type Response struct {
    Data gin.H `json:"data"`
}

// After
type UserInfo struct {
    ID       string `json:"id"`
    Username string `json:"username"`
}

type Response struct {
    Data UserInfo `json:"data"`
}
```

### Issue: Documentation Not Updating

**Problem**: Changes to annotations don't appear in Swagger UI

**Solution**:
1. Regenerate docs: `swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal`
2. Rebuild server: `go build -o bin/server.exe ./cmd/server`
3. Restart server
4. Hard refresh browser (Ctrl+F5)

### Issue: Missing Dependencies

**Problem**: Build fails with "missing go.sum entry"

**Solution**:
```bash
go mod tidy
go build -o bin/server.exe ./cmd/server
```

## Production Considerations

### 1. Disable in Production (Optional)

To disable Swagger UI in production:

```go
// internal/api/router.go
if cfg.Env != "production" {
    router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

### 2. Update Host for Production

```go
// cmd/server/main.go
// @host api.rl-arena.com
// @schemes https
```

### 3. Add Rate Limiting

Consider rate limiting the `/swagger/*` endpoint to prevent abuse.

## Resources

- **Swag GitHub**: https://github.com/swaggo/swag
- **OpenAPI 2.0 Spec**: https://swagger.io/specification/v2/
- **Gin-Swagger**: https://github.com/swaggo/gin-swagger
- **Swagger Editor**: https://editor.swagger.io/ (validate your spec)

## Examples

### Complete Endpoint Documentation

```go
// GetLeaderboard godoc
// @Summary Get global leaderboard
// @Description Get top agents ranked by ELO rating across all environments
// @Tags leaderboard
// @Accept json
// @Produce json
// @Param limit query int false "Number of top agents to return (max 100)" default(20)
// @Param environment query string false "Filter by environment ID"
// @Success 200 {object} map[string]interface{} "Leaderboard with agent rankings"
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /leaderboard [get]
func (h *LeaderboardHandler) GetLeaderboard(c *gin.Context) {
    // Implementation
}
```

### Model with Validation Tags

```go
// User represents a registered user
type User struct {
    ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
    Username  string    `json:"username" example:"johndoe" binding:"required,min=3,max=50"`
    Email     string    `json:"email" example:"john@example.com" binding:"required,email"`
    FullName  string    `json:"fullName" example:"John Doe"`
    CreatedAt time.Time `json:"createdAt" example:"2024-01-01T00:00:00Z"`
}
```

---

**Generated with swag v1.16.6**
