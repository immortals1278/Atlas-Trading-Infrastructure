package kafka

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
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

// PublishRaw 将已序列化的 []byte 直接发布至指定topic
// outbox使用
func (p *Producer) PublishRaw(ctx context.Context, topic, key string, value []byte) error {
	pubCtx, cancel := context.WithTimeout(ctx, p.publishTimeout)
	defer cancel()

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	if err := p.client.ProduceSync(pubCtx, record).FirstErr(); err != nil {
		logger.Error("Outbox: 发布原始事件到kaka失败",
			zap.String("topic", topic),
			zap.String("key", key),
			zap.Error(err),
		)
		return fmt.Errorf("Outbox PublishRaw 失败: %w", err)
	}

	return nil
}

func (p *Producer) CreateTopics(ctx context.Context, topics []string) error {
	req := &kmsg.CreateTopicsRequest{
		TimeoutMillis: 10000,
		Topics:        make([]kmsg.CreateTopicsRequestTopic, 0, len(topics)),
	}
	for _, topic := range topics {
		req.Topics = append(req.Topics, kmsg.CreateTopicsRequestTopic{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}

	kresp, err := p.client.Request(ctx, req)
	if err != nil {
		return fmt.Errorf("CreateTopics 请求失败: %w", err)
	}

	resp := kresp.(*kmsg.CreateTopicsResponse)
	for _, t := range resp.Topics {
		// ErrorCode 36 = TOPIC_ALREADY_EXISTS，已存在
		if t.ErrorCode != 0 && t.ErrorCode != 36 {
			logger.Log.Warn("建立 Kafka topic 失败",
				zap.String("topic", t.Topic),
				zap.Int16("errorCode", t.ErrorCode),
			)
		}
	}
	return nil

}

func (p *Producer) Close() {
	p.client.Close()
}
