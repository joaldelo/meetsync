package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"meetsync/internal/config"
	"meetsync/internal/router"
	"meetsync/pkg/logs"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := logs.New(cfg.Log.Level, os.Stdout)
	logs.SetDefaultLogger(logger)

	logs.Info("Starting MeetSync API server")
	logs.Info("Log level: %s", cfg.Log.Level)

	// Create router
	r := router.New()
	r.Setup()

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logs.Info("Server listening on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logs.Fatal("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logs.Info("Shutting down server...")

	// Create a deadline to wait for current operations to complete
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logs.Fatal("Server forced to shutdown: %v", err)
	}

	logs.Info("Server exited gracefully")
}
