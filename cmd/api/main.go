package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/config"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/database"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/device"
	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/httpserver"
)

func run(ctx context.Context) error {
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

	handler := httpserver.NewServer(deviceHandler)

	// 5. Setup HTTP Server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.Server.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Start Server in Goroutine
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case <-quit:
		log.Println("Shutting down server...")

		// Give outstanding requests 5 seconds to complete
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("shutdown error: %w", err)
		}
	}

	log.Println("Server stopped")
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatalf("%s\n", err)
	}
}
