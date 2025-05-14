package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials/insecure"
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
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	client, err := database.NewMongoClient(cfg.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func(client *mongo.Client, ctx context.Context) {
		err := client.Disconnect(ctx)
		if err != nil {

		}
	}(client, context.Background())

	db := client.Database(cfg.DatabaseName)

	ticketRepo := repository.NewTicketRepository(db)

	eventConn, err := grpc.Dial(
		cfg.EventServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to event service: %v", err)
	}
	defer func(eventConn *grpc.ClientConn) {
		err := eventConn.Close()
		if err != nil {

		}
	}(eventConn)

	notifConn, err := grpc.Dial(
		cfg.NotificationServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to notification service: %v", err)
	}
	defer func(notifConn *grpc.ClientConn) {
		err := notifConn.Close()
		if err != nil {

		}
	}(notifConn)

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

	mux.HandlePath("GET", "/swagger.json", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeFile(w, r, "docs/ticket.swagger.json")
	})

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := ticketpb.RegisterTicketServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("localhost:%d", cfg.GRPCPort), opts,
	); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}
	registerHealthCheckEndpoint(mux)

	err = mux.HandlePath("GET", "/metrics", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		promhttp.Handler().ServeHTTP(w, r)
	})
	if err != nil {
		log.Fatalf("Failed to register /metrics handler: %v", err)
	}

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

func initTracer() func() {
	exporter, _ := otlptracehttp.New(context.Background())
	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)
	return func() { _ = tp.Shutdown(context.Background()) }
}
