package kafka

import (
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client         kgo.Client
	publishTimeout time.Duration
}
