// Package queue wraps RabbitMQ for durable background jobs: queues and
// messages survive broker restarts, the consumer reconnects automatically.
package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Client manages a connection to one RabbitMQ broker.
type Client struct {
	url  string
	mu   sync.Mutex
	conn *amqp.Connection
}

// Dial opens the initial connection to the broker.
func Dial(url string) (*Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	return &Client{url: url, conn: conn}, nil
}

// Publish sends one message to a durable queue, creating the queue if needed.
// Used to push email or heavy jobs off the HTTP lane.
func (c *Client) Publish(ctx context.Context, queueName string, body []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %s: %w", queueName, err)
	}

	err = ch.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish to %s: %w", queueName, err)
	}
	return nil
}

// Consume processes messages from a queue, reconnecting after 3s when the
// broker drops. A handler error requeues the message: jobs must be idempotent.
func (c *Client) Consume(ctx context.Context, queueName string, handler func(context.Context, []byte) error) error {
	var (
		conn       *amqp.Connection
		ch         *amqp.Channel
		deliveries <-chan amqp.Delivery
	)

	disconnect := func() {
		if ch != nil {
			_ = ch.Close()
		}
		if conn != nil {
			_ = conn.Close()
		}
	}

	reconnect := func() error {
		disconnect()
		var err error
		conn, err = amqp.Dial(c.url)
		if err != nil {
			return err
		}
		ch, err = conn.Channel()
		if err != nil {
			_ = conn.Close()
			return err
		}
		if _, err = ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
			_ = conn.Close()
			return err
		}
		deliveries, err = ch.Consume(queueName, "", false, false, false, false, nil)
		if err != nil {
			_ = conn.Close()
			return err
		}
		return nil
	}

	logger.Warn("rabbitmq consumer connecting", logger.String("queue", queueName))
	for {
		if ctx.Err() != nil {
			disconnect()
			return ctx.Err()
		}

		if deliveries == nil {
			if err := reconnect(); err != nil {
				select {
				case <-ctx.Done():
					disconnect()
					return ctx.Err()
				case <-time.After(3 * time.Second):
				}
				continue
			}
			logger.Info("rabbitmq consumer connected", logger.String("queue", queueName))
		}

		select {
		case <-ctx.Done():
			disconnect()
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				deliveries = nil
				continue
			}
			if err := handler(ctx, d.Body); err != nil {
				logger.Error("rabbitmq message failed",
					logger.String("queue", queueName), logger.Err(err))
				// Requeue is safe because jobs are idempotent.
				// ponytail: no dead-letter queue; a persistently failing message
				// loops at the queue head and blocks others behind it.
				_ = d.Nack(false, true)
			} else {
				_ = d.Ack(false)
			}
		}
	}
}

// Close releases the connection; Publish/Consume fail after this.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}