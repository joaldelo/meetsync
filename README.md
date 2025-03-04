# MeetSync

MeetSync is a REST API for scheduling meetings based on participant availability.

## Features

- User management (create, list, get users)
- Create, update, and delete meetings with multiple proposed time slots
- Add, update, and delete participant availability
- Get recommendations for optimal meeting times based on participant availability
- OpenAPI documentation with interactive Swagger UI
- Comprehensive test suite including integration tests
- Graceful shutdown handling
- Structured logging with multiple levels
- Docker support
- Middleware for request logging and error handling

## Requirements

- Go 1.22 or newer
- PostgreSQL (optional, not implemented yet)
- Docker (optional, for containerized deployment)

## Environment Variables

The application can be configured using the following environment variables:

- `SERVER_PORT`: Port for the HTTP server (default: 8080)
- `SERVER_READ_TIMEOUT`: Read timeout for the HTTP server (default: 5s)
- `SERVER_WRITE_TIMEOUT`: Write timeout for the HTTP server (default: 10s)
- `DB_DSN`: Database connection string (default: postgres://postgres:postgres@localhost:5432/meetsync?sslmode=disable)
- `LOG_LEVEL`: Logging level (default: info, options: debug, info, warn, error, fatal)

## Running the Application

### Local Development

```bash
# Clone the repository
git clone https://github.com/joaldelo/meetsync.git
cd meetsync

# Build the application
go build -o meetsync ./cmd/meetsync

# Run the application
./meetsync
```

### Using Docker

```bash
# Build the Docker image
docker build -t meetsync .

# Run the container
docker run -p 8080:8080 meetsync
```

## API Documentation

The API is documented using the OpenAPI 3.1 specification. The documentation is available in the `docs/openapi.yaml` file.

### Viewing the Documentation

You can view the documentation in several ways:

1. **Built-in Swagger UI**: When the server is running locally, you can access the Swagger UI at:
   ```
   http://localhost:8080/docs
   ```
   This provides an interactive UI to explore the API endpoints.

2. **Raw OpenAPI Specification**: You can also access the raw OpenAPI YAML file at:
   ```
   http://localhost:8080/docs/openapi.yaml
   ```

3. **External Swagger UI**: You can use external tools to view the API documentation:
   - [Swagger Editor](https://editor.swagger.io/) - Paste the contents of `docs/openapi.yaml`
   - [Redocly](https://redocly.github.io/redoc/) - Upload the `docs/openapi.yaml` file

4. **Local Swagger UI with Docker**: You can run a local Swagger UI instance using Docker:
   ```bash
   docker run -p 8081:8080 -e SWAGGER_JSON=/docs/openapi.yaml -v $(pwd)/docs:/docs swaggerapi/swagger-ui
   ```
   Then open your browser to http://localhost:8081

## Running Tests

The project includes a comprehensive test suite for all components.

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/handlers

# Run integration tests
go test ./tests/integration
```

## API Endpoints

### User Management

#### Create a User

```
POST /api/users
```

Request body:
```json
{
  "name": "John Doe",
  "email": "john.doe@example.com"
}
```

#### List Users

```
GET /api/users
```

#### Get a User

```
GET /api/users/{id}
```

### Meeting Management

#### Create a Meeting

```
POST /api/meetings
```

Request body:
```json
{
  "title": "Brainstorming meeting",
  "organizerId": "user123",
  "estimatedDuration": 60,
  "proposedSlots": [
    {
      "startTime": "2025-01-12T14:00:00Z",
      "endTime": "2025-01-12T16:00:00Z"
    },
    {
      "startTime": "2025-01-14T18:00:00Z",
      "endTime": "2025-01-14T21:00:00Z"
    }
  ],
  "participantIds": ["user456", "user789"]
}
```

#### Update a Meeting

```
PUT /api/meetings/{id}
```

Request body: Same as create meeting

#### Delete a Meeting

```
DELETE /api/meetings/{id}
```

### Availability Management

#### Add Participant Availability

```
POST /api/availabilities
```

Request body:
```json
{
  "userId": "user456",
  "meetingId": "meeting123",
  "availableSlots": [
    {
      "startTime": "2025-01-12T14:00:00Z",
      "endTime": "2025-01-12T16:00:00Z"
    }
  ]
}
```

#### Update Availability

```
PUT /api/availabilities/{id}
```

Request body:
```json
{
  "availableSlots": [
    {
      "startTime": "2025-01-12T14:00:00Z",
      "endTime": "2025-01-12T16:00:00Z"
    }
  ]
}
```

#### Delete Availability

```
DELETE /api/availabilities/{id}
```

#### Get Meeting Recommendations

```
GET /api/recommendations?meetingId=meeting123
```

Response:
```json
{
  "recommendedSlots": [
    {
      "timeSlot": {
        "id": "slot123",
        "startTime": "2025-01-14T18:00:00Z",
        "endTime": "2025-01-14T21:00:00Z"
      },
      "availableCount": 3,
      "totalParticipants": 3,
      "unavailableParticipants": []
    },
    {
      "timeSlot": {
        "id": "slot456",
        "startTime": "2025-01-12T14:00:00Z",
        "endTime": "2025-01-12T16:00:00Z"
      },
      "availableCount": 2,
      "totalParticipants": 3,
      "unavailableParticipants": [
        {
          "id": "user789",
          "name": "Bob Smith",
          "email": "bob.smith@example.com"
        }
      ]
    }
  ]
}
```

## Project Structure

```
meetsync/
├── api/                    # API-related files
├── cmd/
│   └── meetsync/          # Application entry point
│       └── main.go
├── docs/                   # Documentation files
│   ├── README.md
│   └── openapi.yaml
├── internal/
│   ├── config/            # Application configuration
│   │   └── config.go
│   ├── handlers/          # HTTP request handlers
│   │   ├── meeting.go
│   │   ├── meeting_test.go
│   │   ├── user.go
│   │   └── user_test.go
│   ├── interfaces/        # Service interfaces
│   │   └── services.go
│   ├── middleware/        # HTTP middleware
│   │   └── error.go
│   ├── models/            # Data models
│   │   ├── meeting.go
│   │   └── user.go
│   ├── repositories/      # Data access layer
│   │   ├── user_repository.go
│   │   └── user_repository_test.go
│   └── router/            # HTTP routing
│       ├── router.go
│       └── router_test.go
├── pkg/
│   ├── errors/            # Error handling utilities
│   │   └── errors.go
│   └── logs/             # Logging utilities
│       └── logger.go
├── tests/
│   └── integration/       # Integration tests
├── Dockerfile            # Docker build configuration
├── go.mod
├── go.sum
└── README.md
```
