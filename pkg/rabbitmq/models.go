package rabbitmq

import (
	"encoding/json"

	"github.com/google/uuid"
)

type EventPayload struct {
	ID      uuid.UUID       `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}