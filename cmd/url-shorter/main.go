package main

import (
	"io"
	"log/slog"
	"os"
	"url-shorter/internal/config"
	"url-shorter/internal/server"
	"url-shorter/internal/service"
	"url-shorter/internal/store"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {

	cfg := config.MustLoad()
	servConf := cfg.HTTPServer

	logger := setupLogger(cfg.Env)
	logger.Info("logger is settuped")

	// Подключаемся к БД
	db, err := store.NewDBConnection(&cfg.Storage)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		return
	}
	defer db.Close()
	logger.Info("Successfully connected to database", "storage", cfg.Storage)

	shortService := service.NewShortenerService(db)
	userService := service.NewUserService(db)
	logger.Info("shortener-Service was successfuly created")

	logger.Info("Trying to connect to server")

	server := server.New(servConf.Address, shortService, userService)
	if err := server.Start(); err != nil {
		logger.Info("An error occurred while starting the server", "error", err)
	}
	logger.Info("server started successfully on", "address", servConf.Address)
}

type doubleNewlineWriter struct {
	w io.Writer
}

func (w *doubleNewlineWriter) Write(p []byte) (n int, err error) {

	newP := make([]byte, len(p)+1)
	copy(newP, p)
	newP[len(p)] = '\n'
	return w.w.Write(newP)
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	writer := &doubleNewlineWriter{w: os.Stdout}

	switch env {
	case envLocal:
		txtH := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
		log = slog.New(txtH)
	case envDev:
		jsonH := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelDebug})
		log = slog.New(jsonH)
	case envProd:
		log = slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
