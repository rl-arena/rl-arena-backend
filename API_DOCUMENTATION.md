# RL Arena Backend API Documentation

## Overview

RL Arena Backend is a RESTful API service for managing reinforcement learning agent competitions. It provides endpoints for user authentication, agent management, code submissions, match execution, and leaderboard tracking with ELO-based rankings.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

The API uses JWT (JSON Web Token) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## Response Format

All API responses follow a consistent JSON format:

```json
{
  "data": {},
  "message": "Success message",
  "error": "Error message (if any)"
}
```

## Endpoints

### Authentication

#### POST /auth/login
Login with username/email and password.

**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response:**
```json
{
  "token": "jwt-token",
  "user": {
    "id": "string",
    "username": "string",
    "email": "string",
    "fullName": "string",
    "avatarUrl": "string",
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### POST /auth/register
Register a new user account.

**Request Body:**
```json
{
  "username": "string",
  "email": "string",
  "password": "string",
  "fullName": "string"
}
```

**Response:**
```json
{
  "token": "jwt-token",
  "user": {
    "id": "string",
    "username": "string",
    "email": "string",
    "fullName": "string",
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

### Users

#### GET /users/me
Get current user information. Requires authentication.

**Response:**
```json
{
  "user": {
    "id": "string",
    "username": "string",
    "email": "string",
    "fullName": "string",
    "avatarUrl": "string",
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### PUT /users/me
Update current user information. Requires authentication.

**Request Body:**
```json
{
  "fullName": "string",
  "avatarUrl": "string"
}
```

### Agents

#### GET /agents
Get list of all agents with pagination.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `pageSize` (optional): Number of items per page (default: 20)

**Response:**
```json
{
  "agents": [
    {
      "id": "string",
      "userId": "string",
      "name": "string",
      "description": "string",
      "environmentId": "string",
      "elo": 1200,
      "wins": 10,
      "losses": 5,
      "draws": 2,
      "totalMatches": 17,
      "activeSubmissionId": "string",
      "createdAt": "2023-01-01T00:00:00Z",
      "updatedAt": "2023-01-01T00:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "pageSize": 20
}
```

#### GET /agents/my
Get current user's agents. Requires authentication.

**Response:**
```json
{
  "agents": [
    {
      "id": "string",
      "userId": "string",
      "name": "string",
      "description": "string",
      "environmentId": "string",
      "elo": 1200,
      "wins": 10,
      "losses": 5,
      "draws": 2,
      "totalMatches": 17,
      "activeSubmissionId": "string",
      "createdAt": "2023-01-01T00:00:00Z",
      "updatedAt": "2023-01-01T00:00:00Z"
    }
  ]
}
```

#### GET /agents/:id
Get specific agent by ID.

**Response:**
```json
{
  "agent": {
    "id": "string",
    "userId": "string",
    "name": "string",
    "description": "string",
    "environmentId": "string",
    "elo": 1200,
    "wins": 10,
    "losses": 5,
    "draws": 2,
    "totalMatches": 17,
    "activeSubmissionId": "string",
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### POST /agents
Create a new agent. Requires authentication.

**Request Body:**
```json
{
  "name": "string",
  "description": "string",
  "environmentId": "string"
}
```

**Response:**
```json
{
  "message": "Agent created successfully",
  "agent": {
    "id": "string",
    "userId": "string",
    "name": "string",
    "description": "string",
    "environmentId": "string",
    "elo": 1200,
    "wins": 0,
    "losses": 0,
    "draws": 0,
    "totalMatches": 0,
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### PUT /agents/:id
Update an agent. Requires authentication and ownership.

**Request Body:**
```json
{
  "name": "string",
  "description": "string"
}
```

**Response:**
```json
{
  "message": "Agent updated successfully",
  "agent": {
    "id": "string",
    "userId": "string",
    "name": "string",
    "description": "string",
    "environmentId": "string",
    "elo": 1200,
    "wins": 10,
    "losses": 5,
    "draws": 2,
    "totalMatches": 17,
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### DELETE /agents/:id
Delete an agent. Requires authentication and ownership.

**Response:**
```json
{
  "message": "Agent deleted successfully"
}
```

### Submissions

#### POST /submissions
Create a new code submission. Requires authentication.

**Request Body:**
```json
{
  "agentId": "string",
  "codeUrl": "string"
}
```

**Response:**
```json
{
  "message": "Submission created successfully",
  "submission": {
    "id": "string",
    "agentId": "string",
    "version": 1,
    "status": "pending",
    "codeUrl": "string",
    "isActive": false,
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### GET /submissions/:id
Get specific submission by ID.

**Response:**
```json
{
  "submission": {
    "id": "string",
    "agentId": "string",
    "version": 1,
    "status": "active",
    "codeUrl": "string",
    "buildLog": "string",
    "isActive": true,
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

#### GET /submissions/agent/:agentId
Get all submissions for a specific agent.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `pageSize` (optional): Number of items per page (default: 20)

**Response:**
```json
{
  "submissions": [
    {
      "id": "string",
      "agentId": "string",
      "version": 1,
      "status": "active",
      "codeUrl": "string",
      "isActive": true,
      "createdAt": "2023-01-01T00:00:00Z",
      "updatedAt": "2023-01-01T00:00:00Z"
    }
  ]
}
```

#### PUT /submissions/:id/activate
Set a submission as the active version for its agent. Requires authentication.

**Response:**
```json
{
  "message": "Submission activated successfully",
  "submission": {
    "id": "string",
    "agentId": "string",
    "version": 1,
    "status": "active",
    "codeUrl": "string",
    "isActive": true,
    "createdAt": "2023-01-01T00:00:00Z",
    "updatedAt": "2023-01-01T00:00:00Z"
  }
}
```

### Matches

#### POST /matches
Create and execute a match between two agents. Requires authentication.

**Request Body:**
```json
{
  "agent1Id": "string",
  "agent2Id": "string"
}
```

**Response:**
```json
{
  "message": "Match created and executed",
  "match": {
    "id": "string",
    "environmentId": "string",
    "agent1Id": "string",
    "agent2Id": "string",
    "status": "completed",
    "winnerId": "string",
    "agent1Score": 85.5,
    "agent2Score": 72.3,
    "agent1EloChange": 15,
    "agent2EloChange": -15,
    "replayUrl": "string",
    "startedAt": "2023-01-01T00:00:00Z",
    "completedAt": "2023-01-01T00:00:00Z",
    "createdAt": "2023-01-01T00:00:00Z"
  }
}
```

#### GET /matches
Get list of all matches (currently returns empty - TODO implementation).

**Response:**
```json
{
  "matches": [],
  "message": "List all matches - TODO"
}
```

#### GET /matches/:id
Get specific match by ID.

**Response:**
```json
{
  "match": {
    "id": "string",
    "environmentId": "string",
    "agent1Id": "string",
    "agent2Id": "string",
    "status": "completed",
    "winnerId": "string",
    "agent1Score": 85.5,
    "agent2Score": 72.3,
    "agent1EloChange": 15,
    "agent2EloChange": -15,
    "replayUrl": "string",
    "startedAt": "2023-01-01T00:00:00Z",
    "completedAt": "2023-01-01T00:00:00Z",
    "createdAt": "2023-01-01T00:00:00Z"
  }
}
```

#### GET /matches/agent/:agentId
Get all matches for a specific agent.

**Query Parameters:**
- `page` (optional): Page number (default: 1)
- `pageSize` (optional): Number of items per page (default: 20)

**Response:**
```json
{
  "matches": [
    {
      "id": "string",
      "environmentId": "string",
      "agent1Id": "string",
      "agent2Id": "string",
      "status": "completed",
      "winnerId": "string",
      "agent1Score": 85.5,
      "agent2Score": 72.3,
      "agent1EloChange": 15,
      "agent2EloChange": -15,
      "replayUrl": "string",
      "startedAt": "2023-01-01T00:00:00Z",
      "completedAt": "2023-01-01T00:00:00Z",
      "createdAt": "2023-01-01T00:00:00Z"
    }
  ],
  "total": 25,
  "page": 1,
  "pageSize": 20
}
```

### Leaderboard

#### GET /leaderboard
Get global leaderboard with top agents ranked by ELO rating.

**Query Parameters:**
- `limit` (optional): Number of agents to return (default: 100)

**Response:**
```json
{
  "leaderboard": [
    {
      "rank": 1,
      "agent": {
        "id": "string",
        "userId": "string",
        "name": "string",
        "description": "string",
        "environmentId": "string",
        "elo": 1850,
        "wins": 45,
        "losses": 12,
        "draws": 3,
        "totalMatches": 60,
        "createdAt": "2023-01-01T00:00:00Z",
        "updatedAt": "2023-01-01T00:00:00Z"
      }
    }
  ]
}
```

#### GET /leaderboard/environment/:envId
Get leaderboard for a specific environment.

**Query Parameters:**
- `limit` (optional): Number of agents to return (default: 100)

**Response:**
```json
{
  "leaderboard": [
    {
      "rank": 1,
      "agent": {
        "id": "string",
        "userId": "string",
        "name": "string",
        "description": "string",
        "environmentId": "string",
        "elo": 1850,
        "wins": 45,
        "losses": 12,
        "draws": 3,
        "totalMatches": 60,
        "createdAt": "2023-01-01T00:00:00Z",
        "updatedAt": "2023-01-01T00:00:00Z"
      }
    }
  ],
  "environment": "string"
}
```

### Health Check

#### GET /health
Check API health status.

**Response:**
```json
{
  "status": "OK",
  "timestamp": "2023-01-01T00:00:00Z"
}
```

## Error Codes

| Status Code | Description |
|-------------|-------------|
| 200 | OK - Request successful |
| 201 | Created - Resource created successfully |
| 400 | Bad Request - Invalid request data |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Access denied |
| 404 | Not Found - Resource not found |
| 409 | Conflict - Resource conflict |
| 500 | Internal Server Error - Server error |

## Data Models

### Match Status
- `pending` - Match is queued for execution
- `running` - Match is currently being executed
- `completed` - Match finished successfully
- `failed` - Match execution failed

### Submission Status
- `pending` - Submission is queued for processing
- `building` - Code is being built/validated
- `active` - Submission is ready and active
- `failed` - Build/validation failed
- `inactive` - Submission exists but not active

## Rate Limiting

The API implements rate limiting to ensure fair usage:
- 100 requests per minute per IP address
- 1000 requests per hour per authenticated user

## Static File Access

Submitted code files and replay data can be accessed via:
```
GET /storage/{file-path}
```

Files are served directly from the configured storage path.