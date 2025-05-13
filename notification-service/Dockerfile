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
# This includes 'notification-service' itself, the 'proto' directory,
# and any other local packages that are part of your main module.
# Assuming 'proto' and other potential shared local directories are at the root of the build context.
COPY ./notification-service ./notification-service
COPY ./proto ./proto
# If you have other local directories at the root of your project
# that are part of the 'github.com/doniiel/event-ticketing-platform' module
# and are imported by notification-service, copy them here as well.
# Example: COPY ./common ./common
# Alternatively, if all source code is needed and you have a good .dockerignore:
# COPY . .
# However, being specific with COPY commands can be better for cache optimization
# if only certain parts of the codebase change. For this example, we'll assume
# notification-service primarily needs its own code and the proto definitions.

# Build the application
# Output the binary to a 'bin' subdirectory for clarity
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/notification-service ./notification-service/cmd/server

# --- Final Stage ---
# Use a specific version of Alpine for reproducibility and consistency
FROM alpine:3.20

# Install ca-certificates for HTTPS calls and curl for potential health checks
RUN apk --no-cache add ca-certificates curl

# Set the working directory for the final image
# Using /app is a common convention
WORKDIR /app

# Copy the compiled binary from the builder stage
# Ensure the source path matches the output path from the build command in the builder stage
COPY --from=builder /app/bin/notification-service .

# Expose the gRPC and HTTP ports the service listens on
EXPOSE 50053 8083

# Add a basic healthcheck if your service supports an HTTP health endpoint
# This is similar to your event-service. Adjust the port and path as needed.
# HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
#   CMD curl -f http://localhost:8083/health || exit 1

# Command to run the application
# The binary is now in the WORKDIR (/app)
CMD ["./notification-service"]
