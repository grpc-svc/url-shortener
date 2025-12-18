package main

import (
	"log/slog"
	"os"
	"url-shortener/internal/config"
	"url-shortener/internal/lib/logger/slogcute"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	logger := SetupLogger(cfg.Env)

	logger.Info("Starting URL Shortener Service", slog.String("env", cfg.Env))

	// TODO: init storage

	// TODO: init router : chi

	// TODO: start server
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
