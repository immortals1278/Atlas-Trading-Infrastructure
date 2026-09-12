package redis

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	Client *redis.Client
}

type Config struct {
	Addr         string // redis服务器地址
	Password     string
	DB           int           // 使用的数据库索引，redis有16个数据库
	PoolSize     int           // 最多能同时建立多少个redis连接
	MinIdleConns int           // 最少保持多少个空闲连接
	ReadTimeout  time.Duration // 读取超时
	WriteTimeout time.Duration // 写入超时
}

func DefaultConfig() Config {
	return Config{
		Addr:         "localhost:6379",
		Password:     "",
		DB:           0,
		PoolSize:     100,
		MinIdleConns: 20,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

func NewClient(cfg Config) (*Client, error) {
	var opt *redis.Options
	var err error

	if strings.HasPrefix(cfg.Addr, "redis://") || strings.HasPrefix(cfg.Addr, "rediss://") {
		// 判断 Redis 地址是否以 redis:// 或 rediss:// 开头
		opt, err = redis.ParseURL(cfg.Addr) // 解析url
		if err != nil {
			return nil, err
		}

		opt.PoolSize = cfg.PoolSize
		opt.MinIdleConns = cfg.MinIdleConns
		opt.ReadTimeout = cfg.ReadTimeout
		opt.WriteTimeout = cfg.WriteTimeout

	} else {
		opt = &redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		}
	}

	rdb := redis.NewClient(opt)
	// 测试链接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("无法连接至redis")
		return nil, err
	}

	logger.Info("链接成功")
	return &Client{Client: rdb}, nil
}

// 关闭链接池
func (c *Client) Close() error {
	if c.Client != nil {
		return c.Client.Close()
	}
	return nil
}
