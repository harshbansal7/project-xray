// Package clickhouse implements the Store interface using ClickHouse.
// Production-grade schema with columnar storage, optimized for analytics queries.
package clickhouse

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ClickHouseStore implements store.Store using ClickHouse
type ClickHouseStore struct {
	db *gorm.DB
}

// Config holds ClickHouse store configuration
type Config struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

// New creates a new ClickHouse store
func New(ctx context.Context, cfg Config) (*ClickHouseStore, error) {
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 9000
	}
	if cfg.Database == "" {
		cfg.Database = "xray"
	}
	if cfg.Username == "" {
		cfg.Username = "default"
	}

	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%d/%s?dial_timeout=10s&read_timeout=20s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := gorm.Open(clickhouse.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	store := &ClickHouseStore{
		db: db,
	}

	// Initialize schema
	if err := store.InitSchema(ctx); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return store, nil
}

// Ping checks if ClickHouse is accessible
func (s *ClickHouseStore) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the ClickHouse connection
func (s *ClickHouseStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
