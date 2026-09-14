package kafka

import "time"

type Config struct {
	Brokers                []string
	ConnectTimeout         time.Duration
	PublishTimeout         time.Duration
	AllowAutoTopicCreation bool // 开发环境开启生产环境关闭
	ResetOffset            string
}

func DefaultConfig() Config {
	return Config{
		Brokers:                []string{"localhost:9092"},
		ConnectTimeout:         5 * time.Second,
		PublishTimeout:         2 * time.Second,
		AllowAutoTopicCreation: true,
		ResetOffset:            "latest",
	}
}
