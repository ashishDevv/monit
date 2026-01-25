package rabbitmq

import (
	"context"
	"errors"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp091.Channel                   // AMQP channel for publishing messages
	confirms <-chan amqp091.Confirmation  // Channel to receive publish confirmations
	exchange string                       // Exchange to publish messages to
	routingKey string                     // Routing key for the messages
}

func NewPublisher(conn *amqp091.Connection, exchange, routingKey string) (*Publisher, error) {

	if conn == nil {
		return nil, errors.New("AMQP connection is nil")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		ch.Close()
		return nil, err
	}

	confirms := ch.NotifyPublish(make(chan amqp091.Confirmation, 100))

	return &Publisher{
		ch: ch,
		confirms: confirms,
		exchange: exchange,
		routingKey: routingKey,
	}, nil
}

func (p *Publisher) PublishBatch(ctx context.Context, bodies [][]byte) error {
	for _, body := range bodies {
		if err := p.publish(ctx, body); err != nil {
			return err
		}
	}

	for range bodies {
		select {
		case confirm := <-p.confirms:
			if !confirm.Ack {
				return errors.New("Confirmation not received for message")
			}
		case <-time.After(5 * time.Second):
			return errors.New("Publish confirms timeout")		
	    }
	}

	return nil
}

func (p *Publisher) publish(ctx context.Context, body []byte) error {

	if p.ch == nil {
		return errors.New("AMQP channel is nil")
	}

	return p.ch.PublishWithContext(
		ctx,
		p.exchange,
		p.routingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body: body,
		},
	)
}

func (p *Publisher) Close() error {
	if p.ch != nil {
		return p.ch.Close()
	}
	return nil
}