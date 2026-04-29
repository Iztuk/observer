package daemon

import (
	"cf-observer/internal/audit"
	"cf-observer/internal/config"
	"cf-observer/internal/proxy"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunDaemon(hosts map[string]config.Host) error {
	f, err := os.OpenFile(config.AppProcessConfig.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(f, "cf-observer: ", log.LstdFlags)

	defer func() {
		if err := f.Close(); err != nil {
			logger.Printf("failed to close log file: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	queue := audit.NewQueue(config.AppRunTimeConfig.AuditConfig.QueueSize)
	wg := queue.StartWorkers(config.AppRunTimeConfig.AuditConfig.Workers, logger)

	defer func() {
		queue.Close()
		wg.Wait()
	}()

	pm, err := proxy.NewProxyManager(hosts, logger)
	if err != nil {
		return fmt.Errorf("create proxy manager: %w", err)
	}

	server := &http.Server{
		Addr:              config.AppRunTimeConfig.Listen,
		Handler:           pm,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		logger.Printf("daemon listening on %s", config.AppRunTimeConfig.Listen)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}

		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server failed: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	logger.Println("daemon shutdown complete")
	return nil
}
