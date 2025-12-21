package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	ssogrpc "url-shortener/internal/client/grpc"
	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	mwAuth "url-shortener/internal/http-server/middleware/auth"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/jwt"
	"url-shortener/internal/lib/logger/slogcute"
	"url-shortener/internal/storage"
	"url-shortener/internal/storage/sqlite"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := SetupLogger(cfg.Env)

	log.Info("Starting URL Shortener Service", slog.String("env", cfg.Env))

	// Initialize SSO gRPC client
	ssoClient, err := ssogrpc.New(
		context.Background(),
		log,
		cfg.Clients.SSO.Address,
		cfg.Clients.SSO.Timeout,
		cfg.Clients.SSO.Retries,
	)
	if err != nil {
		log.Error("Failed to create SSO gRPC client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// TODO: Use the SSO gRPC client where needed
	_ = ssoClient // Mark as intentionally unused until implementation

	// Initialize storage
	var storageInstance storage.Storage
	storageInstance, err = sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("Failed to initialize storage", slog.String("error", err.Error()))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	jwtValidator, err := jwt.New(cfg.AppSecret)
	if err != nil {
		log.Error("failed to init jwt validator", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Protected routes (require JWT)
	router.Group(func(r chi.Router) {
		r.Use(mwAuth.New(log, jwtValidator))

		r.Post("/url", save.New(log, storageInstance))
		// TODO: r.Delete("/url", save.New(log, storageInstance))
	})

	// Public routes
	router.Get("/{alias}", redirect.New(log, storageInstance))

	log.Info("starting HTTP server", slog.String("addr", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Info("HTTP server is listening", slog.String("addr", cfg.HTTPServer.Address))
		serverErrors <- srv.ListenAndServe()
	}()

	// Channel to listen for interrupt or terminate signal from the OS
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking select
	select {
	case err := <-serverErrors:
		log.Error("HTTP server start failed", slog.String("error", err.Error()))
		os.Exit(1)

	case sig := <-shutdown:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))

		// Create context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPServer.ShutdownTimeout)
		defer cancel()

		// Gracefully shutdown the server
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", slog.String("error", err.Error()))
			// Force close if graceful shutdown fails
			if err := srv.Close(); err != nil {
				log.Error("force close failed", slog.String("error", err.Error()))
			}
			os.Exit(1)
		}

		log.Info("HTTP server stopped gracefully")
	}

	// Close storage connection
	if err := storageInstance.Close(); err != nil {
		log.Error("failed to close storage", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("storage closed successfully")
	log.Info("application shutdown complete")
}

func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = SetupCuteSlog()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func SetupCuteSlog() *slog.Logger {
	opts := slogcute.CuteHandlerOptions{
		SlogOptions: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewCuteHandler(os.Stdout)

	return slog.New(handler)
}
