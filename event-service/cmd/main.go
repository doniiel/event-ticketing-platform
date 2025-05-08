package main

import (
	"context"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/repository"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/event"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"sync"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	repo, err := repository.NewEventRepository("root:password@tcp(127.0.0.1:3306)/event_ticketing?parseTime=true")
	if err != nil {
		logger.Fatalf("Failed to connect to MySQL: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Start gRPC server
	go func() {
		defer wg.Done()
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			logger.Fatalf("Failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		event.RegisterEventServiceServer(grpcServer, &eventService{repo: repo, logger: logger})
		logger.Info("gRPC server running on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		defer wg.Done()
		mux := runtime.NewServeMux()
		err := event.RegisterEventServiceHandlerServer(context.Background(), mux, &eventService{repo: repo, logger: logger})
		if err != nil {
			logger.Fatalf("Failed to register gateway: %v", err)
		}
		logger.Info("HTTP server running on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			logger.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	wg.Wait()
}
