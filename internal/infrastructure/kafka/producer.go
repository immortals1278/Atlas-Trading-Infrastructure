package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client         kgo.Client
	publishTimeout time.Duration
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
