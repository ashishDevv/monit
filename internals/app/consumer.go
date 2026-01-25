package app

import (
	"context"
	"project-k/pkg/rabbitmq"
)

func StartConsumer(ctx context.Context, c *Container) {

	userService := c.userSvc
	eventHandler := rabbitmq.NewEventHandler(userService)

	// this run as seperate goroutine as consume method is ranging on the message delivery channel
	go func() {
		if err := c.Consumer.Consume(ctx, eventHandler); err != nil {
			c.Logger.Error().
				Err(err).
				Msg("rabbitmq consumer stopped")
		}
	}()
}
