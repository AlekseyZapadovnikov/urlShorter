package main

import (
	"context"
	"os"
	"fmt"
	"io"
	"log"
	"log/slog"
	"url-shorter/internal/config"
	"url-shorter/internal/store"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)


func main() {
    // Загружаем конфиг (возьмёт настройки из env или default)
    cfg := config.MustLoad()

    // Подключаемся к БД
    db, err := store.NewDBConnection(&cfg.Storage)
    if err != nil {
        log.Fatalf("Connection error: %v", err)
    }
    defer db.Close()

    ctx := context.Background()







    // 7) GiveShortUrl non-existing
    fmt.Print("Test GiveShortUrl non-existing: ")
    _, err = db.GiveShortUrl(ctx, "https://unknown.com")
    if err == store.ErrShortURLNotFound {
        fmt.Println("OK (got ErrShortURLNotFound)")
    } else if err != nil {
        fmt.Println("FAIL (unexpected error):", err)
    } else {
        fmt.Println("FAIL (expected ErrShortURLNotFound)")
    }
}


// doubleNewlineWriter добавляет дополнительную пустую строку после каждой записи лога.
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
