package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"url-shorter/internal/config"

	pgerr "github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DbManager struct {
	conn *pgx.Conn
}

func NewDBConnection(cfg *config.Storage) (*DbManager, error) {
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

	return &DbManager{conn: conn}, nil
}

func (db *DbManager) Close() error {
	if db.conn != nil {
		return db.conn.Close(context.Background())
	}
	return nil
}

func (db *DbManager) SaveUser(ctx context.Context, mail, password string) error {
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

func (db *DbManager) SaveUrl(ctx context.Context, shortCode, longUrl string) (int64, error) {
    query := `
      INSERT INTO urls (short_code, original_url, created_at)
      VALUES ($1, $2, NOW())
      RETURNING id
    `
    var id int64
    err := db.conn.QueryRow(ctx, query, shortCode, longUrl).Scan(&id)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == pgerr.UniqueViolation {
            return -1, ErrShortURLExists
        }
        return -1, fmt.Errorf("error while adding URL: %w", err)
    }
	slog.Info("url was saved", "url", longUrl, "alias", shortCode)
    return id, nil
}


// GetURL возвращает original_url из таблицы urls по переданному short_code.
// Если записи с таким alias нет — возвращает ErrShortURLNotFound.
func (db *DbManager) GetUrl(ctx context.Context, alias string) (string, error) {
	const query = `
        SELECT original_url
          FROM urls
         WHERE short_code = $1
    `
	var longURL string
	err := db.conn.QueryRow(ctx, query, alias).Scan(&longURL)
	if err != nil {
		// Если в БД нет строки с таким short_code
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrShortURLNotFound
		}
		// Все прочие ошибки отдаем дальше
		return "", fmt.Errorf("error while getting original URL: %w", err)
	}
	return longURL, nil
}

// SaveAlias обновляет короткий код (short_code) для уже существующего original_url.
// Если такого original_url нет — возвращает ErrShortURLNotFound.
// Если новый alias уже занят — возвращает ErrShortURLExists.
func (db *DbManager) SaveAlias(ctx context.Context, alias, longURL string) error {
	const query = `
        UPDATE urls
           SET short_code = $1
         WHERE original_url = $2
    `

	// Выполняем UPDATE
	cmd, err := db.conn.Exec(ctx, query, alias, longURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgerr.UniqueViolation:
				return ErrShortURLExists
			}
		}
		return fmt.Errorf("error while updating alias: %w", err)
	}

	// Если не затронулось ни одной строки — такого longURL нет
	if cmd.RowsAffected() == 0 {
		return ErrShortURLNotFound
	}

	return nil
}

func (db *DbManager) GetAlias(ctx context.Context, longUrl string) (string, error) {
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


// после написания пакета service я понимаю, что вот это - 
// "func (db *DbManager) SaveUrl(ctx context.Context, shortCode, longUrl, mail string) (int64, error) {"
// это просто глупость, во-первых, singlResp, если SaveUrl, оно должно saveUrl, а не Url и mail,
// это сильно ухудшило читаемость кода в пакете service, где пришлось везде за собой тоскать этот mail, и я подозреваю, что ещё
// придётся, если у меня будет время, то я перепишу всё по-нормальному