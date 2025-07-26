package store

import (
	"context"
	"errors"
	"fmt"
	"url-shorter/internal/config"

	pgerr "github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbManager struct {
	conn *pgx.Conn
}

func NewDBConnection(cfg *config.Storage) (*dbManager, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	conn, err := pgx.Connect(context.Background(), connStr)

	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err) // обернули ошибку
	}

	return &dbManager{conn: conn}, nil
}

func (db *dbManager) Close() error {
	if db.conn != nil {
		return db.conn.Close(context.Background())
	}
	return nil
}

func (db *dbManager) AddUser(ctx context.Context, mail, password string) error {
	query := `
        INSERT INTO users (mail, password, created_at)
        VALUES ($1, $2, NOW())
    `
	if _, err := db.conn.Exec(ctx, query, mail, password); err != nil {
		var curErr *pgconn.PgError
		if errors.As(err, &curErr) && curErr.Code == pgerr.UniqueViolation {
			return ErrUserExists
		}
		return fmt.Errorf("error while adding user: %w", err)
	}
	return nil
}

func (db *dbManager) AddUrl(ctx context.Context, shortCode, longUrl, mail string) (int64, error) {
	query := `
        INSERT INTO urls (short_code, original_url, user_mail, created_at)
        VALUES ($1, $2, $3, NOW())
        RETURNING id
    `
	var id int64
	err := db.conn.QueryRow(ctx, query, shortCode, longUrl, mail).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerr.UniqueViolation {
			return -1, ErrShortURLExists
		}

		if errors.As(err, &pgErr) && pgErr.Code == pgerr.ForeignKeyViolation {
			return -1, fmt.Errorf("link was created, but %w", ErrUserNotFound)
		} // на самом деле это так потренироваться больше
		return -1, fmt.Errorf("error while adding URL: %w", err)
	}
	return id, nil
}

func (db *dbManager) GiveShortUrl(ctx context.Context, longUrl string) (string, error) {
	query := `
        SELECT short_code FROM urls
        WHERE original_url = $1
    `

	var shortUrl string
	err := db.conn.QueryRow(ctx, query, longUrl).Scan(&shortUrl)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("original (long) url doesn`t exist in Data Basse %w", ErrShortURLNotFound)
		}
		return "", fmt.Errorf("error while getting short URL: %w", err)
	}
	return shortUrl, nil
}
