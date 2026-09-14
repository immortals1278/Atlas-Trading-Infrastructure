package kafka

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type Producer struct {
	client         *kgo.Client
	publishTimeout time.Duration
}

// NewProducer 创建 Kafka 生产者
func NewProducer(cfg Config) (*Producer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
	}

	// 仅在配置允许时启用自动创建 Topic（通常仅用于开发环境）
	if cfg.AllowAutoTopicCreation {
		opts = append(opts, kgo.AllowAutoTopicCreation())
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 Kafka Producer 失败: %w", err)
	}

	// 确认 Broker 可达
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("kafka Broker 连接失败: %w", err)
	}

	logger.Info("✅ Kafka Producer 连接成功", zap.Strings("brokers", cfg.Brokers))
	return &Producer{client: client, publishTimeout: cfg.PublishTimeout}, nil
}

func (p *Producer) Publish(ctx context.Context, topic, key string, payload interface{}) error {
	PubCtx, cancel := context.WithTimeout(context.Background(), p.publishTimeout)
	defer cancel()

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化事件失败(topic: %s): %w", topic, err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	}

	if err := p.client.ProduceSync(PubCtx, record).FirstErr(); err != nil {
		return fmt.Errorf("发送事件失败(: %s): %w", topic, err)
	}

	return nil

}

func (p *Producer) Close() {
	p.client.Close()
}
