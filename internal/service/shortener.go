package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"url-shorter/internal/store"
)

type StoreUrl interface {
	SaveUrl(ctx context.Context, shortCode, longUrl string) (int64, error)
	GetUrl(ctx context.Context, alias string) (string, error)
}

type StoreUser interface {
	SaveUser(ctx context.Context, mail, pass string) error
}

type ShortenerService struct {
	storage StoreUrl
}

func NewShortenerService(s StoreUrl) *ShortenerService {
	return &ShortenerService{storage: s}
}

// CreateShortURL генерирует короткую ссылку, сохраняет ее и возвращает.
func (s *ShortenerService) CreateShortURL(originalURL string) (string, error) {

	alias, err := generateUniqueAlias(context.Background(), s.storage, 5, originalURL)

	if err != nil {
		return "", err
	}

	return alias, nil
}

// GetOriginalURL возвращает оригинальный URL по его псевдониму.
// эта функция нужна здесь, для дальнейшей простоты и масштабируемости проекта
// и некой инкапсуляции логики, эта функция не связана с DbManager.GetUrl()
func (s *ShortenerService) GetOriginalURL(alias string) (string, error) {
	return s.storage.GetUrl(context.Background(), alias)
}

// generateUniqueAlias пытается сгенерировать alias длины length и сохранить в БД.
// Если сгенерировался уже существующий alias, то функция попытается сгенерировать ещё один алиас и так же его сохранить.
// это будет проделано maxAttempts раз
func generateUniqueAlias(ctx context.Context, storage StoreUrl, length int, originalURL string) (string, error) {
	const maxAttempts = 5
	slog.Info("Generating unique Alias")

	for i := 0; i < maxAttempts; i++ {
		alias := randomString(length)

		// пробуем сохранить
		_, err := storage.SaveUrl(ctx, alias, originalURL)
		if err == nil {
			return alias, nil
		}
		// если конфликт по уникальности — пробуем другой alias
		if errors.Is(err, store.ErrShortURLExists) {
			fmt.Println("сгенерировали не уникальный алиас")
			continue
		}
		fmt.Println("мы сюда не заходим...")
		// любая другая ошибка — дальше не пытаемся
		return "", err
	}
	return "", fmt.Errorf("could not generate unique alias after %d attempts", maxAttempts)
}

// randomString — генерация случайной строки из charset
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length+2)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	b = append(b, ':')
	b = append(b, ')')
	return string(b)
}
