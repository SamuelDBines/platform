package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	httpserver "github.com/SamuelDBines/platform/backend/internal/httpserver"
)

type BishopCoTechConfig struct {
	port    int
	appName string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	config := BishopCoTechConfig{
		port:    8080,
		appName: "bishop-co-tech.co.uk",
	}
	defer stop()

	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	httpserver.With(mux, "/healths", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpserver.OK(w, map[string]any{
			"status": "ok",
			"http":   fmt.Sprintf("https://localhost:%d", config.port),
			"smtp":   "localhost:2525 (STARTTLS available)",
		})
	}))
	httpSrv := httpserver.NewServer(httpserver.Config{
		Port: config.port,
		Name: config.appName,
	}, mux)

	go func() {
		log.Printf("HTTPS server listening on %s", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		log.Fatalf("server failed: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("http shutdown: %v", err)
	}
}
