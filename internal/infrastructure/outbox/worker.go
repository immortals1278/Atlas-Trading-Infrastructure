package outbox

import (
	"context"
	"encoding/json"
)

// 解耦worker与kafka
type Publisher interface {
	PublishRaw(ctx context.Context, topic, partitionKey string, value []byte) error
}

// 将任意结构体转化成json []byte
func MarshalPayload(v any) ([]byte, error) {
	return json.Marshal(v)
}
