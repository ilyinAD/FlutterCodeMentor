package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ilyin-ad/flutter-code-mentor/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPgxPoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	cfgDB := cfg.Database
	dbURL := cfgDB.GetDatabaseURL()
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30
	poolConfig.HealthCheckPeriod = time.Minute

	return poolConfig, nil
}

func NewPostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolConfig, err := NewPgxPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	fmt.Printf("Connected to database %s\n", cfg.Database.GetDatabaseURL())

	return pool, nil
}

func Close(pool *pgxpool.Pool) {
	if pool != nil {
		pool.Close()
	}
}
