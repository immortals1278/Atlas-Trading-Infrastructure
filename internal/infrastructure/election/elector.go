package election

import (
	"atlas-trading-infrastructure/internal/infrastructure/logger"
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type electorRepository interface {
	AcquireLock(ctx context.Context, partition, instanceID string) (fencingToken int64, acquired bool, err error)
	ExtendLease(ctx context.Context, partition, instanceID string, fencingToken int64) error
	ReleaseLock(ctx context.Context, partition, instanceID string) error
}

var _, electorRepository = (*Repository)(nil)

type Elector struct {
	repo          electorRepository
	partition     string
	instanceID    string
	renewInterval time.Duration

	fencingToken atomic.Int64
	isLeader     atomic.Bool
}

func NewElector(repo *Repository, partition string, instanceID string) *Elector {
	return &Elector{
		repo:          repo,
		partition:     partition,
		instanceID:    instanceID,
		renewInterval: 10 * time.Second, // 每 5 秒尝试竞选或续租（租约 15 秒，有 3 个心跳周期的容错窗口）
	}
}

// 是否为leader
func (e *Elector) IsLeader() bool {
	return e.isLeader.Load()
}

func (e *Elector) GetFencingToken() int64 {
	return e.fencingToken.Load()
}

func (e *Elector) Run(
	ctx context.Context,
	onBecomeLeader func(), // 成为leader时的回调
	onLoseLeader func(), // 失去leader时的回调
) {
	ticker := time.NewTicker(e.renewInterval)
	defer ticker.Stop()

	logger.Log.Info("LeaderElector 已启动",
		zap.String("partition", e.partition),
		zap.String("instanceID", e.instanceID),
	)

	for {
		select {
		case <-ctx.Done():
			if e.isLeader.Load() {
				// 优雅关闭 主动释放租约
				releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := e.repo.ReleaseLock(releaseCtx, e.partition, e.instanceID); err != nil {
					logger.Log.Warn("释放leader锁失败", zap.Error(err))
				}
				cancel()
				e.fencingToken.Store(0)
				e.isLeader.Store(false)
				logger.Log.Info("已释放leader锁")
			}
			return

		case <-ticker.C:
			e.tick(ctx, onBecomeLeader, onLoseLeader)
		}
	}
}

// 执行一次续租或选主逻辑
func (e *Elector) tick(
	ctx context.Context,
	onBecomeLeader func(),
	onLoseLeader func(),
) {
	// leader尝试续租
	if e.IsLeader() {
		err := e.repo.ExtendLease()
		if err != nil {
			// 续租失败
			logger.Log.Warn("leader 续租失败", zap.Error(err))
			e.fencingToken.Store(0)
			e.isLeader.Store(false)
			if onLoseLeader != nil {
				onLoseLeader()
			}
		}
		return
	}
	// 不是leader 尝试竞选
	token, acquired, err := e.repo.AcquireLock(ctx, e.partition, e.instanceID)
	if err != nil {
		logger.Log.Error("竞选leader时发生db错误", zap.Error(err))
		return
	}

	if acquired {
		e.fencingToken.Store(token)
		e.isLeader.Store(true)
		logger.Log.Info("成功取得 leader 锁",
			zap.String("partition", e.partition),
			zap.String("instanceID", e.instanceID),
			zap.Int64("fencingToken", token),
		)
		if onBecomeLeader != nil {
			onBecomeLeader()
		}
	}

}
