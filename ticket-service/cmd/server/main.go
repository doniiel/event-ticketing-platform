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

	ticketpb "github.com/doniiel/event-ticketing-platform/proto/ticket"
	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/config"
	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/database"
	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/handler"
	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/repository"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	client, err := database.NewMongoClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	db := client.Database(cfg.DatabaseName)

	ticketRepo := repository.NewTicketRepository(db)

	eventConn, err := grpc.Dial(
		cfg.EventServiceAddr,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to event service: %v", err)
	}
	defer eventConn.Close()

	notifConn, err := grpc.Dial(
		cfg.NotificationServiceAddr,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to notification service: %v", err)
	}
	defer notifConn.Close()

	ticketHandler := handler.NewTicketHandler(ticketRepo, eventConn, notifConn)

	grpcServer := grpc.NewServer()
	ticketpb.RegisterTicketServiceServer(grpcServer, ticketHandler)
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
	if err := ticketpb.RegisterTicketServiceHandlerFromEndpoint(
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

func registerHealthCheckEndpoint(mux *runtime.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
}
