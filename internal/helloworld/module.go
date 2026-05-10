package helloworld

import (
	"encoding/json"
	"log"
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", handleHelloWorld)
	mux.HandleFunc("GET /hello-world", handleHelloWorld)
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /hello-world/health", handleHealth)
}

func handleHelloWorld(w http.ResponseWriter, r *http.Request) {
	log.Printf("hello-world")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(`{"message":"Hello-World"}`)); err != nil {
		log.Printf("response err: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"module": "hello-world",
		"status": "ok",
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("response err: %v", err)
	}
}
