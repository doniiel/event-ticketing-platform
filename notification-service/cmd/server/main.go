package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doniiel/event-ticketing-platform/notification-service/internal/config"
	"github.com/doniiel/event-ticketing-platform/notification-service/internal/consumer"
	"github.com/doniiel/event-ticketing-platform/notification-service/internal/database"
	"github.com/doniiel/event-ticketing-platform/notification-service/internal/handler"
	"github.com/doniiel/event-ticketing-platform/notification-service/internal/repository"
	notificationpb "github.com/doniiel/event-ticketing-platform/proto/notification"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// @title Notification Service API
// @version 1.0
// @description This is the API for the Notification Service of the Event Ticketing Platform.
// @host localhost:8080
// @BasePath /api/v1
func main() {
	cfg := config.LoadConfig()

	db, err := database.NewMySQLConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Printf("Failed to close DB connection: %v", err)
		}
	}(db)

	notificationRepo := repository.NewNotificationRepository(db)
	notificationHandler := handler.NewNotificationHandler(notificationRepo)

	ticketConsumer := consumer.NewTicketConsumer(cfg.TicketServiceAddr, notificationRepo)
	if err := ticketConsumer.Start(); err != nil {
		log.Printf("Warning: Failed to start ticket consumer: %v", err)
	}
	defer ticketConsumer.Stop()

	grpcServer := grpc.NewServer()
	notificationpb.RegisterNotificationServiceServer(grpcServer, notificationHandler)
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	go func() {
		log.Printf("gRPC server listening on :%d", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := notificationpb.RegisterNotificationServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("localhost:%d", cfg.GRPCPort), opts,
	); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	registerHealthCheckEndpoint(mux)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		log.Printf("HTTP server listening on :%d", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")

	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server shutdown error: %v", err)
	}

	log.Println("Servers gracefully stopped")
}

func registerHealthCheckEndpoint(mux *runtime.ServeMux) {

	err := mux.HandlePath("GET", "/health", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			return
		}
	})
	if err != nil {
		return
	}

}
