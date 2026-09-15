package main

import (
	"atlas-trading-infrastructure/internal/domain"
	"atlas-trading-infrastructure/internal/infrastructure/db"
	"atlas-trading-infrastructure/internal/infrastructure/kafka"
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"atlas-trading-infrastructure/internal/infrastructure/redis"
	"atlas-trading-infrastructure/internal/matching"
	"atlas-trading-infrastructure/internal/matching/engine"
	"atlas-trading-infrastructure/internal/repository"
	"context"
	"os"
	"strings"

	"go.uber.org/zap"
)

func main() {
	defer logger.Sync()

	// 数据库
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		logger.Log.Fatal("engine DATABASE_URL 没设置环境变量")
	}
	dbCfg := db.DefaultDBConfig(dbURL)
	dbCfg.MaxOpenConns = 20 // 撮合引擎数据库操作较少

	pool, err := db.NewPostgresPool(context.Background(), dbCfg)
	if err != nil {
		logger.Log.Fatal("engine 数据库连接失败", zap.Error(err))
	}
	defer pool.Close()

	repo := repository.NewPostgresRepository(pool)

	// redis
	redisCfg := redis.DefaultConfig()
	if redisAddr := os.Getenv("REDIS_URL"); redisAddr != "" {
		redisCfg.Addr = redisAddr
	}
	redisClient, err := redis.NewClient(redisCfg)
	if err != nil {
		logger.Log.Warn("engine redis创建失败, 订单薄缓存不可用", zap.Error(err))
	}

	var cacheRepo domain.CacheRepository

	if redisClient != nil {
		cacheRepo = repository.NewRedisCacheRepository(redisClient)
		logger.Log.Info("redis已连接")
	}

	// kafka producer
	kafkaCfg := kafka.DefaultConfig()
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		kafkaCfg.Brokers = strings.Split(brokers, ",")
	}
	if resetOffset := os.Getenv("KAFKA_RESET_OFFSET"); resetOffset != "" {
		kafkaCfg.ResetOffset = strings.ToLower(resetOffset)
	}
	if os.Getenv("KAFKA_ALLOW_AUTO_CREATE") == "false" {
		kafkaCfg.AllowAutoTopicCreation = false
	} else if os.Getenv("KAFKA_ALLOW_AUTO_CREATE") == "true" {
		kafkaCfg.AllowAutoTopicCreation = true
	}

	producer, err := kafka.NewProducer(kafkaCfg)
	if err != nil {
		logger.Log.Fatal("kafka连接失败", zap.Error(err))
	}
	defer producer.Close()
	logger.Log.Info("kafka producer已连接")

	// 撮合引擎
	engineManager := engine.NewEngineManager()
	svc := matching.NewSubscriber(engineManager, producer, cacheRepo)

	// leader election设置 （election设置完后才启动consumer）

}
