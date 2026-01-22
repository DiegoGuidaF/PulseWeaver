package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"gorm.io/driver/sqlite" // Sqlite driver based on CGO
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

func InitDB() (*gorm.DB, error) {
	// 1. Define the database file name
	// GORM will create database in the current directory if it doesn't exist
	dsn := "data.db"

	// 2. Open the connection
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// 3. AutoMigrate: Creates the "devices" table if missing
	// This ensures your schema is ready before you start handling requests
	err = db.AutoMigrate(&Device{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

type Device struct {
	ID        uint `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time
}

func getDevices(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var devices []Device
	if result := db.Find(&devices); result.Error != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(devices); err != nil {
		// Note: If encoding fails after writing header, you can't change status code
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func main() {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database initialized and connected successfully")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	})

	// Routes
	r.Get("/api/v1/devices", func(w http.ResponseWriter, r *http.Request) {
		// This works because 'db' is captured from main() scope
		getDevices(w, r, db)
	})
	// Start server
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
