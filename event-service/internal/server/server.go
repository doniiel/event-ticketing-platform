package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"
)

func UnaryLoggerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	log.Printf("gRPC method %s called with request: %+v", info.FullMethod, req)

	resp, err := handler(ctx, req)

	log.Printf("gRPC method %s completed in %v with response: %+v, error: %v",
		info.FullMethod, time.Since(start), resp, err)

	return resp, err
}

func HttpLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		log.Printf("HTTP %s %s called", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

		log.Printf("HTTP %s %s completed in %v",
			r.Method, r.URL.Path, time.Since(start))
	})
}
