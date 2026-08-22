package outbox

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

type WorkerRepository interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error //把回调函数包进一个数据库事务里执行。
}

var _ WorkerRepository = (*Repository)(nil) //没实现

// 解耦worker与kafka
type Publisher interface {
	PublishRaw(ctx context.Context, topic, partitionKey string, value []byte) error
}

// Worker：定期扫描outbox_message表，将pending的消息发送到kafka，并标记为published
type Worker struct {
	repo      WorkerRepository
	publisher Publisher
	interval  time.Duration //定时扫描间隔
	batchSize int           //每批处理量
}

func NewWorker(repo *Repository, publisher Publisher, interval time.Duration, batchSize int) *Worker {
	return &Worker{
		repo:      repo,
		publisher: publisher,
		interval:  interval,
		batchSize: batchSize,
	}
}

// 在orderService main.go 中开一个goroutine启动
func (w *Worker) Start(ctx context.Context) {
	logger.Log.Info("Outbox worker 已启动")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("outbox worker收到关闭信号,正在停止")
			return
		case <-ticker.C: // 计时器每响一次
			w.process(ctx)
		}
	}
}

// 进行一次批量扫描并发送
func (w *Worker) process(ctx context.Context) {

	err := w.repo.WithTx(ctx, func(txCtx context.Context) error {})

	if err != nil {
		logger.Log.Error("outbox worker 批量处理失败", zap.Error(err))
	}
}

// 将任意结构体转化成json []byte
func MarshalPayload(v any) ([]byte, error) {
	return json.Marshal(v)
}
