package service

import (
	"context"
)

// UserStorage определяет контракт для хранилища пользователей.
type UserStorage interface {
	SaveUser(ctx context.Context, mail, hash string) error
	GetUserByEmail(ctx context.Context, mail string) (int64, string, error)
	CreateSession(ctx context.Context, userID int64) (string, error)
	DeleteSession(ctx context.Context, token string) error
	GetUserIDBySessionToken(ctx context.Context, token string) (int64, error)
}

// UserService реализует бизнес-логику для пользователей.
type UserService struct {
	storage UserStorage
}

func NewUserService(s UserStorage) *UserService {
	return &UserService{storage: s}
}

// RegisterUser регистрирует нового пользователя.
func (us *UserService) RegisterUser(ctx context.Context, mail, hash string) error {
	return us.storage.SaveUser(ctx, mail, hash)
}

// GetUserByEmail находит пользователя по email.
func (us *UserService) GetUserByEmail(ctx context.Context, mail string) (int64, string, error) {
	return us.storage.GetUserByEmail(ctx, mail)
}

// CreateSession создает сессию для пользователя.
func (us *UserService) CreateSession(ctx context.Context, userID int64) (string, error) {
	return us.storage.CreateSession(ctx, userID)
}

// DeleteSession удаляет сессию.
func (us *UserService) DeleteSession(ctx context.Context, token string) error {
	return us.storage.DeleteSession(ctx, token)
}

// GetUserIDBySessionToken получает ID пользователя по токену сессии.
func (us *UserService) GetUserIDBySessionToken(ctx context.Context, token string) (int64, error) {
	return us.storage.GetUserIDBySessionToken(ctx, token)
}