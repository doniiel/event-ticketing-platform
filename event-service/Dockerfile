FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY event-service ./event-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/event-service ./event-service/cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /app/bin/event-service .

EXPOSE 50051 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

CMD ["./event-service"]