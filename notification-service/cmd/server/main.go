package main

import (
	"context"
	"fmt"
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
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.NewMySQLConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

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
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := notificationpb.RegisterNotificationServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("localhost:%d", cfg.GRPCPort), opts,
	); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
	}

	go func() {
		log.Printf("HTTP server listening on :%d", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
