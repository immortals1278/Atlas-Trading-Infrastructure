package kafka

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type HandleFunc func(ctx context.Context, key, value []byte) error

type Consumer struct {
	client  *kgo.Client
	groupID string
	topics  []string
	wg      sync.WaitGroup
}

func NewConsumer(cfg Config, groupID string, topics []string) (*Consumer, error) {
	resetOffset := kgo.NewOffset().AtEnd() // 默认从最新开始读
	switch cfg.ResetOffset {
	case "", "latest":
		resetOffset = kgo.NewOffset().AtEnd()
	case "earliest":
		resetOffset = kgo.NewOffset().AtStart()
	default:
		logger.Log.Warn("未设置offset 默认用latest")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(),             // 取消自动提交offset 处理成功再提交避免丢消息
		kgo.ConsumeResetOffset(resetOffset), //重置offset
	)

	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:  client,
		groupID: groupID,
		topics:  topics,
	}, nil
}

// ctx被取消时 自动取消并停止consumer
func (c *Consumer) Start(ctx context.Context, Handler HandleFunc) {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()
		defer c.client.Close()
		for {
			fetches := c.client.PollFetches(ctx) // 从kafka一次拉一批消息

			// ctx被取消 优雅退出
			if ctx.Err() != nil {
				logger.Log.Info("kafka consumer 已停止", zap.String("groupID", c.groupID))
			}

			if errs := fetches.Errors(); len(errs) != 0 {
				for _, e := range errs {
					logger.Log.Error("kafka拉取消息失败", zap.String("groupID", c.groupID), zap.Error(e.Err))
				}
				continue
			}

			fetches.EachRecord(func(record *kgo.Record) {
				backoff := 100 * time.Millisecond
				for {
					// 处理每条消息前再确认一次
					if ctx.Err() != nil {
						return
					}

					if err := Handler(ctx, record.Key, record.Value); err != nil {
						logger.Error("Kafka 信息处理失败 等待重试",
							zap.String("topic", record.Topic),
							zap.String("group", c.groupID),
							zap.Duration("backoff", backoff),
							zap.Error(err),
						)

						select {
						case <-time.After(backoff): // 不用time.sleep （等待时无法响应取消）
							backoff *= 2
							if backoff > 30*time.Second {
								backoff = 30 * time.Second
							}
						case <-ctx.Done():
							return
						}
						continue
					}

					if err := c.client.CommitRecords(ctx, record); err != nil { // 提交offset
						if ctx.Err() != nil {
							return
						}
						logger.Error("Kafka offset commit 失败",
							zap.String("topic", record.Topic),
							zap.String("group", c.groupID),
							zap.Duration("backoff", backoff),
							zap.Error(err),
						)

						select {
						case <-time.After(backoff):
							backoff *= 2
							if backoff > 30*time.Second {
								backoff = 30 * time.Second
							}
						case <-ctx.Done():
							return
						}
						continue
					}
				}
			})
		}
	}()
}

func (c *Consumer) Wait() {
	c.wg.Wait()
}
