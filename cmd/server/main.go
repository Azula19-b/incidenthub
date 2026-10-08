package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Azula19-b/incidenthub/internal/handlers"
	"github.com/Azula19-b/incidenthub/internal/middleware"
)

func main() {
	addr := envOrDefault("ADDR", ":8080")
	databasePath := envOrDefault("DATABASE_PATH", "data/incidenthub.db")
	app, err := handlers.New(databasePath)
	if err != nil {
		log.Fatalf("initialize application: %v", err)
	}
	defer app.Close()

	server := &http.Server{
		Addr:              addr,
		Handler:           middleware.RequestLogger(app.Routes()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("IncidentHub listening on http://localhost%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
