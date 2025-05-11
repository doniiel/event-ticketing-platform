# Event Ticketing Platform

A microservices-based platform for managing event tickets, allowing users to browse events, purchase tickets, and receive notifications. The system emphasizes scalability and real-time updates.

## Architecture

The platform consists of three microservices:

1. **Event Service**: Manages event details (name, date, location, ticket stock)
2. **Ticket Service**: Handles ticket purchases and availability checks
3. **Notification Service**: Sends notifications (e.g., purchase confirmation, event reminders)

## Technologies Used

- **Go**: Main programming language
- **gRPC**: For service-to-service communication
- **Protocol Buffers**: For message definitions
- **MySQL**: For Event and Notification services data
- **MongoDB**: For Ticket service data
- **Docker**: For containerization and orchestration

## Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Protocol Buffers Compiler (protoc)
- MySQL (for local development)
- MongoDB (for local development)

## Getting Started

### Running with Docker Compose

The easiest way to run the entire platform is with Docker Compose:

```bash
# Build the images
make docker-build

# Run the services
make docker-run
```

### Running Locally

1. Start MySQL and MongoDB:

```bash
# Initialize databases
make init-db
```

2. Build and run the services:

```bash
# Generate protocol buffer code
make proto

# Build the services
make build

# Run the services
make run
```

## API Documentation

### Event Service

- GET `/events`: List all events
- GET `/events/{id}`: Get event details
- POST `/events`: Create a new event
- PUT `/events/{id}`: Update event details
- DELETE `/events/{id}`: Delete an event

### Ticket Service

- POST `/tickets`: Purchase tickets
- GET `/tickets/{id}`: Get ticket details
- GET `/tickets/user/{user_id}`: List user tickets

### Notification Service

- POST `/notifications`: Send a notification
- GET `/notifications/user/{user_id}`: List user notifications

## Development

### Directory Structure

```
event-ticketing-platform/
├── event-service/
│   ├── cmd/
│   ├── internal/
│   └── Dockerfile
├── ticket-service/
│   ├── cmd/
│   ├── internal/
│   └── Dockerfile
├── notification-service/
│   ├── cmd/
│   ├── internal/
│   └── Dockerfile
├── proto/
│   ├── event.proto
│   ├── ticket.proto
│   └── notification.proto
├── docker-compose.yml
├── go.mod
├── go.sum
└── Makefile
```

### Adding a New Feature

1. Update the protocol buffer definitions if needed
2. Generate code: `make proto`
3. Implement the feature in the appropriate service
4. Test locally: `make run`
5. Build and run with Docker: `make docker-run`

## Testing

Run tests with:

```bash
make test
```

## Cleanup

Stop all services and remove containers:

```bash
make stop
```

Clean build artifacts:

```bash
make clean
```