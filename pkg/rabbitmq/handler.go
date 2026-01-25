package rabbitmq

import (
	"context"
	"encoding/json"

	"github.com/ashishDevv/cosmic-user-service/internals/modules/user"
	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"
)

type UserService interface {
	CreateUser(context.Context, uuid.UUID, user.CreateUser) error
}

type EventHandler struct {
	service UserService
}

func NewEventHandler(svc UserService) *EventHandler {
	return &EventHandler{
		service: svc,
	}
}

func (h *EventHandler) Handle(ctx context.Context, msg amqp091.Delivery) error {
	var event EventPayload
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return err
	}

	if event.Type != "user.created" {
		return nil // ignore unknown events
	}

	var payload user.CreateUser
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	return h.service.CreateUser(ctx, event.ID, payload)
}
