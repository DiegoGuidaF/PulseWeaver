package main

import (
	"log"
	"net/http"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	conf, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	log.Printf("starting server. Port: %v, debug: %v, db_file: %v",
		conf.Server.Port,
		conf.Server.Debug,
		conf.DB.File,
	)

	db, err := database.NewSQLite(&conf.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database initialized and connected successfully")

	deviceRepo := device.NewRepository(db)
	deviceService := device.NewService(deviceRepo)
	deviceHandler := device.NewHandler(deviceService)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	r.Get("/health", handler.Health)
	r.Get("/api/v1/devices", deviceHandler.GetDevices)

	// Start server
	err = http.ListenAndServe(":8080", r)
	if err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
