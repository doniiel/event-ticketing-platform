# --- Builder Stage ---
# Use a specific version of Golang Alpine for reproducibility
FROM golang:1.23-alpine AS builder

# Set the working directory in the builder stage
WORKDIR /app

# Copy module files first to leverage Docker cache for dependencies
COPY go.mod go.sum ./

# Download dependencies
# This step benefits from caching if go.mod and go.sum haven't changed
RUN go mod download

# Copy all local source code required for the build.
# This includes 'ticket-service' itself, the 'proto' directory,
# and any other local packages that are part of your main module.
# Assuming 'proto' and other potential shared local directories are at the root of the build context.
COPY ./ticket-service ./ticket-service
COPY ./proto ./proto
# If you have other local directories at the root of your project
# that are part of the 'github.com/doniiel/event-ticketing-platform' module
# and are imported by ticket-service, copy them here as well.
# Example: COPY ./common ./common
# The 'COPY . .' you had before is also fine if you have a good .dockerignore
# and all source code is relevant. Being specific can sometimes help with cache.

# Build the application
# The output path /app/bin/ticket-service is good.
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/ticket-service ./ticket-service/cmd/server

# --- Final Stage ---
# Use a specific version of Alpine for reproducibility and consistency
FROM alpine:3.20

# Install ca-certificates for HTTPS calls and curl for potential health checks
RUN apk --no-cache add ca-certificates curl

# Set the working directory for the final image
WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/bin/ticket-service .

# Expose the gRPC and HTTP ports the service listens on
EXPOSE 50052 8082

# Add a basic healthcheck if your service supports an HTTP health endpoint.
# Adjust the port and path as needed.
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD curl -f http://localhost:8082/health || exit 1

# Command to run the application
CMD ["./ticket-service"]
