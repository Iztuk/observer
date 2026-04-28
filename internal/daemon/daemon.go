package daemon

import (
	"cf-observer/internal/audit"
	"cf-observer/internal/config"
	"cf-observer/internal/proxy"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func RunDaemon(hosts map[string]config.Host) error {
	f, err := os.OpenFile(config.AppProcessConfig.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	logger := log.New(f, "cf-observer: ", log.LstdFlags)

	pm, err := proxy.NewProxyManager(hosts, logger)
	if err != nil {
		return fmt.Errorf("create proxy manager: %w", err)
	}

	queue := audit.NewQueue(config.AppRunTimeConfig.AuditConfig.QueueSize)
	wg := queue.StartWorkers(config.AppRunTimeConfig.AuditConfig.Workers)
	defer queue.Close()
	wg.Wait()

	server := &http.Server{
		Addr:              config.AppRunTimeConfig.Listen,
		Handler:           pm,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}
