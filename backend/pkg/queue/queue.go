// Package queue wraps RabbitMQ for durable background jobs: queues and
// messages survive broker restarts, publishers and consumers reconnect
// automatically.
package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// dialTimeout bounds opening a broker connection, which runs under the
// client lock.
const dialTimeout = 5 * time.Second

var errClosed = errors.New("queue client is closed")

// Client manages a connection to one RabbitMQ broker.
type Client struct {
	url    string
	mu     sync.Mutex
	conn   *amqp.Connection
	closed bool
}

// Dial opens the initial connection to the broker.
func Dial(url string) (*Client, error) {
	conn, err := dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	return &Client{url: url, conn: conn}, nil
}

func dial(url string) (*amqp.Connection, error) {
	return amqp.DialConfig(url, amqp.Config{Dial: amqp.DefaultDial(dialTimeout)})
}

// Publish sends one persistent message to a durable queue, creating the queue
// if needed, and waits until the broker confirms it stored the message. When
// the connection dropped (broker restart) it dials again once (T26). Only
// getting the connection is serialized: publishers never wait for each other's
// broker confirms, so a broker that does not confirm only stalls its callers
// up to their context deadline.
func (c *Client) Publish(ctx context.Context, queueName string, body []byte) error {
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		conn, cerr := c.connection()
		if errors.Is(cerr, errClosed) {
			return cerr
		}
		if cerr != nil {
			err = cerr
			continue
		}
		if err = publishOn(ctx, conn, queueName, body); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return err
		}
		c.drop(conn)
	}
	return err
}

// connection returns the live connection, dialing a new one when it dropped.
func (c *Client) connection() (*amqp.Connection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errClosed
	}
	if c.conn == nil || c.conn.IsClosed() {
		conn, err := dial(c.url)
		if err != nil {
			return nil, fmt.Errorf("dial rabbitmq: %w", err)
		}
		c.conn = conn
	}
	return c.conn, nil
}

// drop closes a connection that failed, unless another publisher replaced it.
func (c *Client) drop(conn *amqp.Connection) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == conn {
		_ = conn.Close()
		c.conn = nil
	}
}

func publishOn(ctx context.Context, conn *amqp.Connection, queueName string, body []byte) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %s: %w", queueName, err)
	}
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirm, err := ch.PublishWithDeferredConfirmWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish to %s: %w", queueName, err)
	}
	acked, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait for broker confirm on %s: %w", queueName, err)
	}
	if !acked {
		return fmt.Errorf("broker rejected message on %s", queueName)
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
		conn, err = dial(c.url)
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

// Close releases the connection; Publish fails after this.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}
