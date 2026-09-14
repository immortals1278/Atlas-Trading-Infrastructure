package db

import (
	"time"
)

type DBConfig struct {
	URL          string
	MaxOpenConns int           // pgx Config.MaxConns
	MinOpenConns int           // pgx Config.MinConns
	MaxIdleTime  time.Duration // pgx Config.MaxConnIdleTime
	MaxLifeTime  time.Duration // pgx Config.MaxConnLifetime
}

func DefaultDBConfig(url string) DBConfig {
	return DBConfig{
		URL:          url,
		MaxOpenConns: 50,
		MinOpenConns: 5, // 保持 5 条连接预热
		MaxIdleTime:  5 * time.Minute,
		MaxLifeTime:  1 * time.Hour,
	}
}
