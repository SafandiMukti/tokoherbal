package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tokoherbal/auth-service/internal/handler"
	"tokoherbal/auth-service/internal/service"
	"tokoherbal/auth-service/internal/store"
	"tokoherbal/auth-service/internal/token"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "periksa /health lalu keluar (dipakai Docker HEALTHCHECK)")
	flag.Parse()

	port := getenv("PORT", "8080")
	if *healthcheck {
		os.Exit(runHealthcheck(port))
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log, port); err != nil {
		log.Error("server berhenti dengan error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, port string) error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL belum diisi")
	}
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		return errors.New("JWT_SECRET harus diisi, minimal 32 karakter")
	}

	// ctx dibatalkan saat menerima Ctrl+C atau SIGTERM (docker stop)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	tokens := token.NewManager(secret, 15*time.Minute)
	svc, err := service.New(store.NewPostgres(pool), tokens, service.Config{})
	if err != nil {
		return err
	}
	h := handler.New(svc, log)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h.Router(getenv("CORS_ORIGIN", "http://localhost:3000")),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("auth-service berjalan", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("menerima sinyal berhenti, menunggu request yang berjalan selesai")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func runHealthcheck(port string) int {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
