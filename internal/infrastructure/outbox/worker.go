package outbox

import (
	"encoding/json"
)

// 将任意结构体转化成json []byte
func MarshalPayload(v any) ([]byte, error) {
	return json.Marshal(v)
}
