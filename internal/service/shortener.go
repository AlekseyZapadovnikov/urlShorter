// internal/service/shortener.go
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"url-shorter/internal/store"
)


type StoreUrl interface {
    SaveUrl(ctx context.Context, shortCode, longUrl, mail string) (int64, error)
    GetUrl(ctx context.Context, alias string) (string, error)
}

type StoreUser interface {
	SaveUser(ctx context.Context, mail string) error 
}


type ShortenerService struct {
    storage StoreUrl
	dbManger *store.DbManager
}


func NewShortenerService(s StoreUrl, db *store.DbManager) *ShortenerService {
    return &ShortenerService{storage: s,  dbManger: db}
}

// CreateShortURL генерирует короткую ссылку, сохраняет ее и возвращает.
func (s *ShortenerService) CreateShortURL(originalURL, mail string) (string, error) {

    alias, err := generateUniqueAlias(context.Background(), s.dbManger, 7, originalURL, mail)

	if err != nil {
		return "", err
	}

    if _, err := s.storage.SaveUrl(context.Background(), alias, originalURL, mail); err != nil {
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
func generateUniqueAlias(ctx context.Context, storage StoreUrl, length int, originalURL, mail string) (string, error) {
    const maxAttempts = 5

    for i := 0; i < maxAttempts; i++ {
        alias := randomString(length)

        // пробуем сохранить
        _, err := storage.SaveUrl(ctx, alias, originalURL, mail)
        if err == nil {
            return alias, nil
        }
        // если конфликт по уникальности — пробуем другой alias
        if errors.Is(err, store.ErrShortURLExists) {
            continue
        }
        // любая другая ошибка — дальше не пытаемся
        return "", err
    }
    return "", fmt.Errorf("could not generate unique alias after %d attempts", maxAttempts)
}


// randomString — генерация случайной строки из charset
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, length)
    for i := range b {
        n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        b[i] = charset[n.Int64()]
    }
    return string(b)
}

