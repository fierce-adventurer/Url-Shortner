package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type URLStore interface {
	Save(shortCode string, originalUrl string) error
	Get(shortCode string) (string, error)
	IncrementClick(shortCode string) error
	Close() error
}

type CloudStore struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewCloudStore(dbURL string, redisURL string) (*CloudStore, error) {

	// initializing postgres connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}

	// initializing redis connection
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)

	return &CloudStore{db: db, rdb: rdb}, nil
}

func (s *CloudStore) Close() error {
	if err := s.db.Close(); err != nil {
		return err
	}
	return s.rdb.Close()
}

func (s *CloudStore) Save(shortCode string, originalUrl string) error {
	ctx := context.Background()

	// save to postgres
	query := `INSERT INTO urls (short_code, original_url) VALUES ($1, $2)`
	_, err := s.db.ExecContext(ctx, query, shortCode, originalUrl)
	if err != nil {
		return err
	}

	err = s.rdb.Set(ctx, shortCode, originalUrl, 24*time.Hour).Err()
	return err
}

func (s *CloudStore) Get(shortCode string) (string, error) {
	ctx := context.Background()

	// TRY FETCHING FROM REDIS
	url, err := s.rdb.Get(ctx, shortCode).Result()
	if err == redis.Nil {

		// cache miss
		var originalUrl string
		query := `SELECT original_url FROM urls WHERE short_code = $1`
		err := s.db.QueryRowContext(ctx, query, shortCode).Scan(&originalUrl)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", errors.New("short code not found")
			}
			return "", err
		}

		// found in postgres caching in redis
		s.rdb.Set(ctx, shortCode, originalUrl, 24*time.Hour)
		return originalUrl, nil

	} else if err != nil {
		// some other redis error
		return "", err
	}

	return url, nil
}

func (s *CloudStore) IncrementClick(shortCode string) error {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_code = $1`
	_, err := s.db.Exec(query, shortCode)
	return err
}
