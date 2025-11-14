package service

import (
	"context"
	"encoding/json"
	"fmt"
	"push-service/internal/config"
	"push-service/internal/models"
	"push-service/internal/platform/fcm"
	"push-service/internal/queue"
	"push-service/internal/repository"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type PushService interface {
	SendPush(ctx context.Context, req models.SendPushRequest) error
	SendBulkPush(ctx context.Context, req models.BulkPushRequest) error
	ProcessPushFromQueue(ctx context.Context, delivery amqp.Delivery) error
	ProcessGatewayMessage(ctx context.Context, delivery amqp.Delivery) error
	GetQueueStats(ctx context.Context) (map[string]int64, error)
}

type pushService struct {
	deviceRepo repository.DeviceRepository
	fcmClient  fcm.FCMClient
	pushQueue  *queue.PushQueue
	cfg        *config.Config
}

func NewPushService(deviceRepo repository.DeviceRepository, fcmClient fcm.FCMClient, pushQueue *queue.PushQueue, cfg *config.Config) PushService {
	return &pushService{
		deviceRepo: deviceRepo,
		fcmClient:  fcmClient,
		pushQueue:  pushQueue,
		cfg:        cfg,
	}
}

func (s *pushService) SendPush(ctx context.Context, req models.SendPushRequest) error {
	zap.L().Debug("=== SEND PUSH START ===",
		zap.String("user_id", req.UserID),
		zap.String("title", req.Title),
		zap.String("body", req.Body),
		zap.Any("data", req.Data),
		zap.Strings("platforms", req.Platforms),
	)

	// Get user's devices
	devices, err := s.deviceRepo.GetByUserID(ctx, req.UserID)
	if err != nil {
		zap.L().Error("❌ FAILED to get user devices from database",
			zap.String("user_id", req.UserID),
			zap.Error(err),
		)
		return fmt.Errorf("database error: %w", err)
	}

	zap.L().Debug("📱 Database query result",
		zap.String("user_id", req.UserID),
		zap.Int("device_count", len(devices)),
		zap.Any("devices", devices), // Log all devices found
	)

	if len(devices) == 0 {
		zap.L().Warn("⚠️ No devices found for user", zap.String("user_id", req.UserID))
		return fmt.Errorf("no devices found for user: %s", req.UserID)
	}

	// Filter by platform if specified
	var targetDevices []models.Device
	if len(req.Platforms) > 0 {
		zap.L().Debug("🔍 Filtering devices by platforms", zap.Strings("platforms", req.Platforms))
		for _, device := range devices {
			for _, platform := range req.Platforms {
				if device.Platform == platform {
					targetDevices = append(targetDevices, device)
					break
				}
			}
		}
	} else {
		targetDevices = devices
	}

	zap.L().Debug("🎯 Devices after filtering",
		zap.Int("original_count", len(devices)),
		zap.Int("filtered_count", len(targetDevices)),
	)

	if len(targetDevices) == 0 {
		zap.L().Error("❌ No devices match the specified platforms",
			zap.String("user_id", req.UserID),
			zap.Strings("requested_platforms", req.Platforms),
			zap.Any("available_platforms", getPlatforms(devices)),
		)
		return fmt.Errorf("no devices match platforms: %v", req.Platforms)
	}

	// Extract device tokens
	deviceTokens := make([]string, len(targetDevices))
	for i, device := range targetDevices {
		deviceTokens[i] = device.Token
		zap.L().Debug("📲 Device token",
			zap.String("platform", device.Platform),
			zap.String("token", device.Token), // Log full token for debugging
		)
	}

	// Create notification
	notification := models.PushNotification{
		UserID: req.UserID,
		Title:  req.Title,
		Body:   req.Body,
		Image:  req.Image,
		Link:   req.Link,
		Data:   req.Data,
		Status: "queued",
	}

	zap.L().Info("🚀 Enqueuing push notification to RabbitMQ",
		zap.String("user_id", req.UserID),
		zap.Int("device_count", len(deviceTokens)),
		zap.String("title", req.Title),
	)

	// Enqueue to RabbitMQ instead of sending directly
	if err := s.pushQueue.EnqueuePush(ctx, notification, deviceTokens); err != nil {
		zap.L().Error("💥 Failed to enqueue push notification",
			zap.String("user_id", req.UserID),
			zap.Int("device_count", len(deviceTokens)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to enqueue push notification: %w", err)
	}

	zap.L().Info("✅ Push notification enqueued successfully",
		zap.String("user_id", req.UserID),
		zap.Int("device_count", len(deviceTokens)),
	)

	return nil
}

// Helper function to get unique platforms from devices
func getPlatforms(devices []models.Device) []string {
	platforms := make(map[string]bool)
	for _, device := range devices {
		platforms[device.Platform] = true
	}

	result := make([]string, 0, len(platforms))
	for platform := range platforms {
		result = append(result, platform)
	}
	return result
}

func (s *pushService) SendBulkPush(ctx context.Context, req models.BulkPushRequest) error {
	// For bulk pushes, use the queue for better scalability
	baseNotification := models.PushNotification{
		Title:  req.Title,
		Body:   req.Body,
		Data:   req.Data,
		Status: "queued",
	}

	enqueuedCount := 0
	for _, userID := range req.UserIDs {
		devices, err := s.deviceRepo.GetByUserID(ctx, userID)
		if err != nil {
			zap.L().Error("Failed to get devices for user",
				zap.String("user_id", userID),
				zap.Error(err),
			)
			continue
		}

		if len(devices) == 0 {
			zap.L().Debug("No devices found for user", zap.String("user_id", userID))
			continue
		}

		deviceTokens := make([]string, len(devices))
		for i, device := range devices {
			deviceTokens[i] = device.Token
		}

		userNotification := baseNotification
		userNotification.UserID = userID

		// Enqueue to RabbitMQ
		if err := s.pushQueue.EnqueuePush(ctx, userNotification, deviceTokens); err != nil {
			zap.L().Error("Failed to enqueue push for user",
				zap.String("user_id", userID),
				zap.Error(err),
			)
			continue
		}

		enqueuedCount++
		zap.L().Info("Bulk push enqueued for user",
			zap.String("user_id", userID),
			zap.Int("device_count", len(deviceTokens)),
		)
	}

	zap.L().Info("Bulk push enqueuing completed",
		zap.Int("enqueued_users", enqueuedCount),
		zap.Int("total_users", len(req.UserIDs)),
	)

	return nil
}

// ProcessPushFromQueue processes a single message from the queue
// This is called by the worker for each message consumed from RabbitMQ
func (s *pushService) ProcessPushFromQueue(ctx context.Context, delivery amqp.Delivery) error {
	var pushMessage queue.PushMessage
	if err := json.Unmarshal(delivery.Body, &pushMessage); err != nil {
		zap.L().Error("Failed to unmarshal push message",
			zap.Error(err),
		)
		// Nack and don't requeue - message is malformed
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, false); err != nil {
			zap.L().Error("Failed to nack malformed message", zap.Error(err))
		}
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	notification := pushMessage.Notification
	deviceTokens := pushMessage.DeviceTokens

	zap.L().Info("Processing push message from queue",
		zap.String("user_id", notification.UserID),
		zap.Int("device_count", len(deviceTokens)),
		zap.String("title", notification.Title),
		zap.Int("retry_count", pushMessage.RetryCount),
	)

	// Validate tokens if validation is enabled
	validTokens := make([]string, 0, len(deviceTokens))
	if s.cfg != nil && s.cfg.Queue.Validation.Enabled {
		for _, token := range deviceTokens {
			validationCtx, cancel := context.WithTimeout(ctx, s.cfg.Queue.Validation.Timeout)
			err := s.fcmClient.ValidateToken(validationCtx, token)
			cancel()

			if err != nil {
				maskedToken := "***"
				if len(token) > 20 {
					maskedToken = token[:10] + "..." + token[len(token)-10:]
				}
				zap.L().Warn("Token validation failed, skipping",
					zap.String("token", maskedToken),
					zap.Error(err),
				)
				continue
			}
			validTokens = append(validTokens, token)
		}

		if len(validTokens) == 0 {
			zap.L().Warn("No valid tokens found, moving to dead letter queue",
				zap.String("user_id", notification.UserID),
				zap.Int("original_count", len(deviceTokens)),
			)
			// All tokens invalid - move to dead letter queue
			if err := s.pushQueue.EnqueueRetry(ctx, pushMessage); err != nil {
				zap.L().Error("Failed to enqueue to retry/dead letter", zap.Error(err))
			}
			// Ack the message since we've handled it
			if err := s.pushQueue.GetRabbitMQClient().Ack(delivery.DeliveryTag, false); err != nil {
				zap.L().Error("Failed to ack message", zap.Error(err))
			}
			return fmt.Errorf("no valid tokens")
		}

		deviceTokens = validTokens
		zap.L().Debug("Token validation completed",
			zap.Int("original_count", len(pushMessage.DeviceTokens)),
			zap.Int("valid_count", len(validTokens)),
		)
	}

	// Update notification status
	notification.Status = "sending"

	// Send notifications via FCM
	successCount, failureCount, err := s.fcmClient.SendMultiple(ctx, deviceTokens, notification)
	if err != nil {
		zap.L().Error("Failed to send push notifications",
			zap.String("user_id", notification.UserID),
			zap.Int("device_count", len(deviceTokens)),
			zap.Error(err),
		)
		// Enqueue for retry
		if err := s.pushQueue.EnqueueRetry(ctx, pushMessage); err != nil {
			zap.L().Error("Failed to enqueue retry", zap.Error(err))
		}
		// Nack and requeue via retry queue
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, false); err != nil {
			zap.L().Error("Failed to nack message", zap.Error(err))
		}
		return fmt.Errorf("fcm send failed: %w", err)
	}

	// Check if all sends failed
	if failureCount == len(deviceTokens) {
		zap.L().Warn("All push notifications failed, enqueuing for retry",
			zap.String("user_id", notification.UserID),
			zap.Int("device_count", len(deviceTokens)),
		)
		// Enqueue for retry
		if err := s.pushQueue.EnqueueRetry(ctx, pushMessage); err != nil {
			zap.L().Error("Failed to enqueue retry", zap.Error(err))
		}
		// Nack - message will go to retry queue
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, false); err != nil {
			zap.L().Error("Failed to nack message", zap.Error(err))
		}
		return fmt.Errorf("all notifications failed")
	}

	// Success - ack the message
	zap.L().Info("Push notifications sent successfully",
		zap.String("user_id", notification.UserID),
		zap.Int("device_count", len(deviceTokens)),
		zap.Int("success_count", successCount),
		zap.Int("failure_count", failureCount),
	)

	if err := s.pushQueue.GetRabbitMQClient().Ack(delivery.DeliveryTag, false); err != nil {
		zap.L().Error("Failed to ack message", zap.Error(err))
		return err
	}

	return nil
}

// GetQueueStats returns statistics about the push queues
func (s *pushService) GetQueueStats(ctx context.Context) (map[string]int64, error) {
	return s.pushQueue.GetQueueStats(ctx)
}

// ProcessGatewayMessage processes messages from the API Gateway's push.queue
// API Gateway sends: {notification_id, user_id, push_token, name, template: {subject, body}, ...}
func (s *pushService) ProcessGatewayMessage(ctx context.Context, delivery amqp.Delivery) error {
	// Parse API Gateway message format
	var gatewayMessage map[string]interface{}
	if err := json.Unmarshal(delivery.Body, &gatewayMessage); err != nil {
		zap.L().Error("Failed to unmarshal gateway message",
			zap.Error(err),
		)
		// Nack and don't requeue - message is malformed
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, false); err != nil {
			zap.L().Error("Failed to nack malformed gateway message", zap.Error(err))
		}
		return fmt.Errorf("failed to unmarshal gateway message: %w", err)
	}

	// Extract data from gateway message
	notificationID, ok := gatewayMessage["notification_id"].(string)
	if !ok {
		zap.L().Error("Missing or invalid notification_id in gateway message")
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, false); err != nil {
			zap.L().Error("Failed to nack gateway message", zap.Error(err))
		}
		return fmt.Errorf("missing notification_id")
	}

	userID, ok := gatewayMessage["user_id"].(string)
	if !ok {
		zap.L().Error("Missing or invalid user_id in gateway message")
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, false); err != nil {
			zap.L().Error("Failed to nack gateway message", zap.Error(err))
		}
		return fmt.Errorf("missing user_id")
	}

	// Get template (may be nil)
	var template map[string]interface{}
	if templateVal, ok := gatewayMessage["template"]; ok {
		if templateMap, ok := templateVal.(map[string]interface{}); ok {
			template = templateMap
		}
	}

	// Extract title and body from template
	title := "Notification"
	body := "You have a new notification"
	if template != nil {
		if subject, ok := template["subject"].(string); ok && subject != "" {
			title = subject
		}
		if bodyContent, ok := template["body"].(string); ok && bodyContent != "" {
			body = bodyContent
		}
	}

	// Get device tokens from database
	devices, err := s.deviceRepo.GetByUserID(ctx, userID)
	if err != nil {
		zap.L().Warn("Failed to get devices from database, using push_token fallback",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	var deviceTokens []string
	if len(devices) > 0 {
		// Use tokens from database
		deviceTokens = make([]string, len(devices))
		for i, device := range devices {
			deviceTokens[i] = device.Token
		}
		zap.L().Info("Using device tokens from database",
			zap.String("user_id", userID),
			zap.Int("device_count", len(deviceTokens)),
		)
	} else {
		// Fallback to push_token from gateway message
		if pushToken, ok := gatewayMessage["push_token"].(string); ok && pushToken != "" {
			deviceTokens = []string{pushToken}
			zap.L().Info("Using push_token from gateway message",
				zap.String("user_id", userID),
			)
		} else {
			zap.L().Warn("No devices found and no push_token provided",
				zap.String("user_id", userID),
				zap.String("notification_id", notificationID),
			)
			// Ack the message since we can't process it
			if err := s.pushQueue.GetRabbitMQClient().Ack(delivery.DeliveryTag, false); err != nil {
				zap.L().Error("Failed to ack gateway message", zap.Error(err))
			}
			return fmt.Errorf("no device tokens available for user: %s", userID)
		}
	}

	// Extract data if present
	var data map[string]interface{}
	if dataVal, ok := gatewayMessage["data"]; ok {
		if dataMap, ok := dataVal.(map[string]interface{}); ok {
			data = dataMap
		}
	}

	// Create notification
	notification := models.PushNotification{
		ID:        notificationID,
		UserID:    userID,
		Title:     title,
		Body:      body,
		Data:      data,
		Status:    "queued",
		CreatedAt: time.Now(),
	}

	zap.L().Info("Processing gateway push message",
		zap.String("notification_id", notificationID),
		zap.String("user_id", userID),
		zap.Int("device_count", len(deviceTokens)),
		zap.String("title", title),
	)

	// Enqueue to internal push queue for processing
	if err := s.pushQueue.EnqueuePush(ctx, notification, deviceTokens); err != nil {
		zap.L().Error("Failed to enqueue push from gateway",
			zap.String("notification_id", notificationID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		// Nack and requeue
		if err := s.pushQueue.GetRabbitMQClient().Nack(delivery.DeliveryTag, false, true); err != nil {
			zap.L().Error("Failed to nack gateway message", zap.Error(err))
		}
		return fmt.Errorf("failed to enqueue push: %w", err)
	}

	// Ack the gateway message
	if err := s.pushQueue.GetRabbitMQClient().Ack(delivery.DeliveryTag, false); err != nil {
		zap.L().Error("Failed to ack gateway message", zap.Error(err))
		return err
	}

	zap.L().Info("Gateway push message enqueued successfully",
		zap.String("notification_id", notificationID),
		zap.String("user_id", userID),
	)

	return nil
}

func (s *pushService) SendDirect(ctx context.Context, token string, notification models.PushNotification) error {
	zap.L().Debug("🔧 Sending direct FCM message",
		zap.String("token", token),
		zap.String("title", notification.Title),
		zap.String("body", notification.Body),
	)

	err := s.fcmClient.Send(ctx, token, notification)
	if err != nil {
		zap.L().Error("💥 FCM direct send failed",
			zap.String("token", token),
			zap.String("error_type", fmt.Sprintf("%T", err)),
			zap.Error(err),
		)
		return err
	}

	zap.L().Info("✅ FCM direct send successful")
	return nil
}
