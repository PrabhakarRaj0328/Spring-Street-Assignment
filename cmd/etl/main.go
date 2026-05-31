package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"spring-street-backend/internal/config"
	"spring-street-backend/internal/database"
	"spring-street-backend/internal/etl"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()

	pipeline := etl.NewPipeline(db)

	// Run once immediately on startup
	if err := pipeline.RunDaily(); err != nil {
		log.Printf("Error running initial ETL pipeline: %v", err)
	}

	// Schedule to run daily using a simple ticker
	// In production, consider using a cron library like github.com/robfig/cron
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Channel to listen for interrupt signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("ETL Scheduler started. Waiting for next run...")

	for {
		select {
		case <-ticker.C:
			if err := pipeline.RunDaily(); err != nil {
				log.Printf("Error running scheduled ETL pipeline: %v", err)
			}
		case <-stop:
			log.Println("Shutting down ETL scheduler...")
			return
		}
	}
}
