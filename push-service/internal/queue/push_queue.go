package queue

import (
	"context"
	"push-service/internal/config"
	"push-service/internal/models"
	"push-service/pkg/rabbitmq"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	PushExchangeName     = "push_exchange"
	PushQueueName        = "push_notifications"
	RetryQueueName       = "push_retries"
	DeadLetterQueue      = "push_dead_letters"
	DeadLetterExchange   = "push_dlx"
	GatewayPushQueueName = "push.queue"
	GatewayExchangeName  = "notifications.direct"
)

type PushQueue struct {
	rabbitmqClient *rabbitmq.RabbitMQClient
	cfg            *config.QueueConfig
}

func NewPushQueue(rabbitmqClient *rabbitmq.RabbitMQClient, cfg *config.QueueConfig) (*PushQueue, error) {
	ctx := context.Background()

	// Set up dead letter exchange
	if err := rabbitmqClient.EnsureExchange(ctx, DeadLetterExchange, "direct"); err != nil {
		return nil, err
	}

	// Set up main exchange
	if err := rabbitmqClient.EnsureExchange(ctx, PushExchangeName, "direct"); err != nil {
		return nil, err
	}

	// Set up dead letter queue with arguments
	dlqArgs := amqp.Table{
		"x-message-ttl": int64(7 * 24 * time.Hour / time.Millisecond), // 7 days
	}
	if err := rabbitmqClient.EnsureQueue(ctx, DeadLetterQueue, dlqArgs); err != nil {
		return nil, err
	}
	if err := rabbitmqClient.BindQueue(ctx, DeadLetterQueue, DeadLetterExchange, "dead_letter"); err != nil {
		return nil, err
	}

	// Set up retry queue with DLX
	retryArgs := amqp.Table{
		"x-dead-letter-exchange":    PushExchangeName,
		"x-dead-letter-routing-key": PushQueueName,
	}
	if err := rabbitmqClient.EnsureQueue(ctx, RetryQueueName, retryArgs); err != nil {
		return nil, err
	}
	if err := rabbitmqClient.BindQueue(ctx, RetryQueueName, PushExchangeName, RetryQueueName); err != nil {
		return nil, err
	}

	// Set up main push queue with DLX
	pushArgs := amqp.Table{
		"x-dead-letter-exchange":    DeadLetterExchange,
		"x-dead-letter-routing-key": "dead_letter",
	}
	if err := rabbitmqClient.EnsureQueue(ctx, PushQueueName, pushArgs); err != nil {
		return nil, err
	}
	if err := rabbitmqClient.BindQueue(ctx, PushQueueName, PushExchangeName, PushQueueName); err != nil {
		return nil, err
	}

	zap.L().Info("Push queue initialized with RabbitMQ",
		zap.String("exchange", PushExchangeName),
		zap.String("queue", PushQueueName),
	)

	return &PushQueue{
		rabbitmqClient: rabbitmqClient,
		cfg:            cfg,
	}, nil
}

type PushMessage struct {
	Notification models.PushNotification `json:"notification"`
	DeviceTokens []string                `json:"device_tokens"`
	RetryCount   int                     `json:"retry_count"`
}

func (q *PushQueue) EnqueuePush(ctx context.Context, notification models.PushNotification, deviceTokens []string) error {
	message := PushMessage{
		Notification: notification,
		DeviceTokens: deviceTokens,
		RetryCount:   0,
	}

	if err := q.rabbitmqClient.Enqueue(ctx, PushExchangeName, PushQueueName, message); err != nil {
		zap.L().Error("Failed to enqueue push message", zap.Error(err))
		return err
	}

	zap.L().Info("Push message enqueued",
		zap.Int("device_count", len(deviceTokens)),
		zap.String("title", notification.Title),
	)
	return nil
}

func (q *PushQueue) ConsumePush(ctx context.Context) (<-chan amqp.Delivery, error) {
	prefetchCount := q.cfg.Worker.PrefetchCount
	if prefetchCount == 0 {
		prefetchCount = 10 // default
	}
	return q.rabbitmqClient.Consume(ctx, PushQueueName, prefetchCount)
}

func (q *PushQueue) EnqueueRetry(ctx context.Context, message PushMessage) error {
	message.RetryCount++

	maxRetries := q.cfg.Retry.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5 // default
	}

	if message.RetryCount > maxRetries {
		// Move to dead letter queue after max retries
		zap.L().Warn("Message exceeded max retries, moving to dead letter queue",
			zap.Int("retry_count", message.RetryCount),
			zap.Int("max_retries", maxRetries),
		)
		return q.rabbitmqClient.Enqueue(ctx, DeadLetterExchange, "dead_letter", message)
	}

	// Calculate backoff delay
	backoff := q.cfg.Retry.Backoff
	if backoff == 0 {
		backoff = 5 * time.Second // default
	}
	delay := time.Duration(message.RetryCount) * backoff

	zap.L().Info("Enqueuing retry",
		zap.Int("retry_count", message.RetryCount),
		zap.Duration("delay", delay),
	)

	// Publish to retry queue with delay
	return q.rabbitmqClient.EnqueueWithDelay(ctx, PushExchangeName, RetryQueueName, message, delay)
}

func (q *PushQueue) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	queues := []string{PushQueueName, RetryQueueName, DeadLetterQueue}
	for _, queueName := range queues {
		length, err := q.rabbitmqClient.QueueLength(ctx, queueName)
		if err != nil {
			zap.L().Warn("Failed to get queue length",
				zap.String("queue", queueName),
				zap.Error(err),
			)
			// Continue with other queues
			stats[queueName] = 0
			continue
		}
		stats[queueName] = length
	}

	return stats, nil
}

// GetRabbitMQClient returns the underlying RabbitMQ client for ack/nack operations
func (q *PushQueue) GetRabbitMQClient() *rabbitmq.RabbitMQClient {
	return q.rabbitmqClient
}

// ConsumeFromGateway consumes messages from the API Gateway's push.queue
func (q *PushQueue) ConsumeFromGateway(ctx context.Context) (<-chan amqp.Delivery, error) {
	// Ensure the gateway exchange exists
	if err := q.rabbitmqClient.EnsureExchange(ctx, GatewayExchangeName, "direct"); err != nil {
		return nil, err
	}

	// Ensure the gateway queue exists
	if err := q.rabbitmqClient.EnsureQueue(ctx, GatewayPushQueueName, nil); err != nil {
		return nil, err
	}

	// Bind queue to exchange with routing key "push"
	if err := q.rabbitmqClient.BindQueue(ctx, GatewayPushQueueName, GatewayExchangeName, "push"); err != nil {
		return nil, err
	}

	prefetchCount := q.cfg.Worker.PrefetchCount
	if prefetchCount == 0 {
		prefetchCount = 10 // default
	}

	zap.L().Info("Gateway queue consumer initialized",
		zap.String("exchange", GatewayExchangeName),
		zap.String("queue", GatewayPushQueueName),
	)

	return q.rabbitmqClient.Consume(ctx, GatewayPushQueueName, prefetchCount)
}
