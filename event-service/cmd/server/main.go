package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doniiel/event-ticketing-platform/event-service/internal/config"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/database"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/handler"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/repository"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/server"
	eventpb "github.com/doniiel/event-ticketing-platform/proto/event"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.NewMySQLConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}(db)

	eventRepo := repository.NewEventRepository(db)
	eventHandler := handler.NewEventHandler(eventRepo)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.UnaryLoggerInterceptor),
	)
	eventpb.RegisterEventServiceServer(grpcServer, eventHandler)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux()

	mux.HandlePath("GET", "/swagger.json", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeFile(w, r, "docs/ticket.swagger.json")
	})

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := eventpb.RegisterEventServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("localhost:%d", cfg.GRPCPort), opts,
	); err != nil {
		log.Fatalf("Failed to register gateway: %v", err)
	}

	err = mux.HandlePath("GET", "/health", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err2 := w.Write([]byte(`{"status":"ok"}`))
		if err2 != nil {
			return
		}
	})
	if err != nil {
		return
	}

	fs := http.FileServer(http.Dir("docs"))
	err = mux.HandlePath("GET", "/swagger.json", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.ServeFile(w, r, "docs/event.swagger.json")
	})
	if err != nil {
		return
	}
	err = mux.HandlePath("GET", "/swagger/*", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		http.StripPrefix("/swagger", fs).ServeHTTP(w, r)
	})
	if err != nil {
		return
	}

	err = mux.HandlePath("GET", "/metrics", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		promhttp.Handler().ServeHTTP(w, r)
	})
	if err != nil {
		log.Fatalf("Failed to register /metrics handler: %v", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: server.HttpLoggerMiddleware(mux),
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

func initTracer() func() {
	exporter, _ := otlptracehttp.New(context.Background())
	tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
	otel.SetTracerProvider(tp)
	return func() { _ = tp.Shutdown(context.Background()) }
}
