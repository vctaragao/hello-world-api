package main

import (
	"context"
	"errors"
	"github.com/vctaragao/hello-world-api/internal/auth"
	"github.com/vctaragao/hello-world-api/internal/helloworld"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const defaultAllowedOrigin = "https://moraes.lat"

func main() {
	mux := http.NewServeMux()
	helloworld.RegisterRoutes(mux)
	authModule, err := auth.NewModuleFromEnv()
	if err != nil {
		log.Fatalf("unable to initialize auth module: %v", err)
	}

	authModule.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8097",
		Handler: corsMiddleware(mux),
	}

	log.Printf("Server listening on port: 8097")

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("unable to start server: %v", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	<-shutdown
	log.Println("Shutingdown server...")

	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*30)
	defer cancelFunc()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("unable to shutdown server: %v", err)
	}

	log.Println("Server terminated cleanly")
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if _, ok := allowedOrigins[origin]; !ok {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
			return
		}

		headers := w.Header()
		headers.Set("Access-Control-Allow-Origin", origin)
		headers.Add("Vary", "Origin")
		headers.Add("Vary", "Access-Control-Request-Method")
		headers.Add("Vary", "Access-Control-Request-Headers")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		headers.Set("Access-Control-Max-Age", "600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func parseAllowedOrigins(rawOrigins string) map[string]struct{} {
	if rawOrigins == "" {
		rawOrigins = defaultAllowedOrigin
	}

	allowedOrigins := make(map[string]struct{})
	for _, origin := range strings.Split(rawOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}

		allowedOrigins[origin] = struct{}{}
	}

	return allowedOrigins
}
