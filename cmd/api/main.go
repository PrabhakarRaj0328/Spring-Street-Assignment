package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"spring-street-backend/internal/api"
	"spring-street-backend/internal/config"
	"spring-street-backend/internal/database"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()

	server := api.NewServer(db)
	r := mux.NewRouter()
	
	// Add a simple healthcheck
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	server.RegisterRoutes(r)

	// CORS middleware — allows frontend (React) applications to consume this API.
	// In production, restrict AllowedOrigins to specific domains.
	corsHandler := corsMiddleware(r)

	srv := &http.Server{
		Handler:      corsHandler,
		Addr:         ":" + cfg.APIPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server starting on port %s", cfg.APIPort)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// corsMiddleware adds CORS headers to all responses.
// This allows a frontend (e.g., React) running on a different origin to call the API.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
