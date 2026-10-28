package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stockflow/backend/internal/config"
	"stockflow/backend/internal/database"
	"stockflow/backend/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run fails fast on migration errors and gives in-flight requests a bounded graceful shutdown.
func run(logger *slog.Logger) error {
	cfg := config.Load()
	startup, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := database.Open(startup, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err = database.Migrate(startup, pool); err != nil {
		return err
	}
	server := &http.Server{Addr: cfg.Address, Handler: httpapi.New(pool, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("StockFlow API ready", "address", cfg.Address)
		serveErr <- server.ListenAndServe()
	}()
	select {
	case err = <-serveErr:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		return server.Shutdown(shutdown)
	}
}
