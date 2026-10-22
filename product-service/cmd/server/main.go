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
	"strconv"
	"syscall"
	"time"

	"tokoherbal/product-service/internal/auth"
	"tokoherbal/product-service/internal/cache"
	"tokoherbal/product-service/internal/handler"
	"tokoherbal/product-service/internal/service"
	"tokoherbal/product-service/internal/store"
	"tokoherbal/product-service/internal/upload"
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
	authURL := os.Getenv("AUTH_URL")
	if authURL == "" {
		return errors.New("AUTH_URL belum diisi (contoh: http://auth-service:8080)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := store.Connect(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// Redis bersifat opsional: tanpa REDIS_ADDR aplikasi berjalan tanpa cache dan rate limit.
	var (
		appCache  service.Cache = cache.Noop{}
		limiter   handler.Counter
		pingCache func(context.Context) error
	)
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		rdb := cache.NewRedis(addr)
		defer rdb.Close()
		appCache, limiter, pingCache = rdb, rdb, rdb.Ping
		log.Info("Redis dipakai", "addr", addr)
	} else {
		log.Warn("REDIS_ADDR kosong: berjalan tanpa cache dan rate limit")
	}

	uploads, err := upload.New(getenv("UPLOAD_DIR", "./uploads"))
	if err != nil {
		return err
	}

	productSvc := service.NewProductService(store.NewProductStore(pool), appCache, log)
	orderSvc := service.NewOrderService(store.NewOrderStore(pool), store.NewCartStore(pool), appCache, log)

	h := handler.New(handler.Deps{
		Products:        productSvc,
		Orders:          orderSvc,
		Verifier:        auth.NewClient(authURL, 3*time.Second),
		Uploads:         uploads,
		Log:             log,
		PingDB:          pool.Ping,
		PingCache:       pingCache,
		RateLimiter:     limiter,
		RateLimitPerMin: getenvInt("RATE_LIMIT_PER_MIN", 60),
		TrustProxy:      getenv("TRUST_PROXY", "false") == "true",
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h.Router(getenv("CORS_ORIGIN", "http://localhost:3000")),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second, // lebih longgar dari auth-service: ada unggah gambar
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("product-service berjalan", "addr", srv.Addr)
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

func getenvInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return n
	}
	return def
}
