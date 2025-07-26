package store

import "errors"

var (
    ErrUserExists        = errors.New("user already exists")
    ErrShortURLExists    = errors.New("short URL already exists")
    ErrUserNotFound      = errors.New("user not found")
    ErrShortURLNotFound  = errors.New("short URL not found")
    ErrInvalidData       = errors.New("invalid data")
    ErrDatabase          = errors.New("database error")
    ErrNotFound          = errors.New("not found")
)
