package outbox

import "github.com/google/uuid"

type Status int

const (
	// 消息尚未发送到 Kafka
	StatusPending Status = 0
	// 消息已发送到 Kafka
	StatusPublished Status = 1
)

type Message struct {
	ID            uuid.UUID
	AggregateID   string
	AggregateType string
	Topic         string
	PartitionKey  string
	Payload       []byte
	Status        Status
	RetryCount    int
	CreatedAt     int64
	PublishedAt   int64
}
