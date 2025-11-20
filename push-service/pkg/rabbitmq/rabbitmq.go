package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"push-service/internal/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     *config.RabbitMQConfig
}

func NewRabbitMQClient(cfg *config.RabbitMQConfig) (*RabbitMQClient, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.VHost,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	client := &RabbitMQClient{
		conn:    conn,
		channel: channel,
		cfg:     cfg,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping RabbitMQ: %w", err)
	}

	zap.L().Info("Connected to RabbitMQ",
		zap.String("host", cfg.Host),
		zap.String("port", cfg.Port),
		zap.String("vhost", cfg.VHost),
	)

	return client, nil
}

func (r *RabbitMQClient) Close() error {
	var errs []error
	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing RabbitMQ: %v", errs)
	}
	return nil
}

func (r *RabbitMQClient) Ping(ctx context.Context) error {
	// Check if connection is still alive
	if r.conn.IsClosed() {
		return fmt.Errorf("connection is closed")
	}
	return nil
}

// EnsureExchange declares an exchange if it doesn't exist
func (r *RabbitMQClient) EnsureExchange(ctx context.Context, name, kind string) error {
	return r.channel.ExchangeDeclare(
		name,  // name
		kind,  // kind (direct, topic, fanout, headers)
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
}

// EnsureQueue declares a queue if it doesn't exist
func (r *RabbitMQClient) EnsureQueue(ctx context.Context, name string, args amqp.Table) error {
	_, err := r.channel.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,  // arguments (for DLX, TTL, etc.)
	)
	return err
}

// BindQueue binds a queue to an exchange
func (r *RabbitMQClient) BindQueue(ctx context.Context, queueName, exchangeName, routingKey string) error {
	return r.channel.QueueBind(
		queueName,    // queue name
		routingKey,   // routing key
		exchangeName, // exchange
		false,        // no-wait
		nil,          // arguments
	)
}

// Enqueue publishes a message to an exchange
func (r *RabbitMQClient) Enqueue(ctx context.Context, exchange, routingKey string, message interface{}) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.channel.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         jsonMessage,
			DeliveryMode: amqp.Persistent, // Make message persistent
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// EnqueueWithDelay publishes a message with a delay (using TTL)
func (r *RabbitMQClient) EnqueueWithDelay(ctx context.Context, exchange, routingKey string, message interface{}, delay time.Duration) error {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	delayMs := int64(delay.Milliseconds())

	err = r.channel.PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         jsonMessage,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Headers: amqp.Table{
				"x-delay": delayMs,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish delayed message: %w", err)
	}

	return nil
}

// Consume starts consuming messages from a queue
func (r *RabbitMQClient) Consume(ctx context.Context, queueName string, prefetchCount int) (<-chan amqp.Delivery, error) {
	// Set QoS to control how many messages are delivered at once
	if err := r.channel.Qos(
		prefetchCount, // prefetch count
		0,             // prefetch size
		false,         // global
	); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := r.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack (we'll manually ack)
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)

	if err != nil {
		return nil, fmt.Errorf("failed to register consumer: %w", err)
	}

	return msgs, nil
}

// QueueLength returns the number of messages in a queue
func (r *RabbitMQClient) QueueLength(ctx context.Context, queueName string) (int64, error) {
	// Use QueueDeclare with Passive: true as QueueInspect is deprecated.
	queue, err := r.channel.QueueDeclare(
		queueName, // queue name
		false,     // durable (unknown, as we're just inspecting)
		false,     // autoDelete
		false,     // exclusive
		true,      // passive: true, only check if the queue exists and inspect it
		nil,       // args
	)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect queue (does it exist?): %w", err)
	}
	return int64(queue.Messages), nil
}

// Ack acknowledges a message
func (r *RabbitMQClient) Ack(tag uint64, multiple bool) error {
	return r.channel.Ack(tag, multiple)
}

// Nack negatively acknowledges a message (reject and requeue)
func (r *RabbitMQClient) Nack(tag uint64, multiple bool, requeue bool) error {
	return r.channel.Nack(tag, multiple, requeue)
}
