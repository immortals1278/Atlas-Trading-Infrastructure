package logger

import (
	"go.uber.org/zap"
)

var Log *zap.Logger

// 把内存缓冲区里还没落盘的日志强制写入磁盘
func Sync() {
	_ = Log.Sync()
}

func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}
