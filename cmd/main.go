package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	server := &http.Server{Addr: ":8097"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("hello-world")

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-type", "application/json")

		if _, err := w.Write([]byte(`{"message": "Hello-World"}`)); err != nil {
			log.Printf("reponse err: %v", err)
		}
	})

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
