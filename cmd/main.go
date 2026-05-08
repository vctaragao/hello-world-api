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
	"syscall"
	"time"
)

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
		Handler: mux,
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
