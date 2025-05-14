FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY ./ticket-service ./ticket-service
COPY ./proto ./proto

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/ticket-service ./ticket-service/cmd/server


FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /app/bin/ticket-service .

EXPOSE 50052 8082

CMD ["./ticket-service"]
