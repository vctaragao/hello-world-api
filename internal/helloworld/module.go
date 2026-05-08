package helloworld

import (
	"log"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", handleHelloWorld)
}

func handleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Printf("hello-world")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(`{"message":"Hello-World"}`)); err != nil {
		log.Printf("response err: %v", err)
	}
}
