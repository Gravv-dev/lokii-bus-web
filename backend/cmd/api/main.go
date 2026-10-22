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

	"github.com/gravv-dev/lokii-bus-web/backend/internal/config"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/database"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/httpapi"
	"github.com/gravv-dev/lokii-bus-web/backend/internal/store"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := checkHealth(); err != nil {
			log.Printf("health check failed: %v", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("apply database migrations: %v", err)
	}

	repository := store.New(pool)
	if cfg.BootstrapAdminEmail != "" {
		if err := repository.BootstrapAdmin(ctx, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword); err != nil {
			log.Fatalf("bootstrap admin account: %v", err)
		}
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpapi.New(cfg, repository).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("API listening on %s", cfg.HTTPAddress)
		serverErrors <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve HTTP: %v", err)
		}
	case <-signals:
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown: %v", err)
		}
	}
}

func checkHealth() error {
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:8080/health")
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("API is not ready")
	}
	return nil
}
