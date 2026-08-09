package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger

// sync确保缓存区写入
func Sync() {
	_ = Log.Sync()
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}
