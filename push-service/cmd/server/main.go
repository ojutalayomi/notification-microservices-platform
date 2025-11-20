// @title           Push Notification Service API
// @version         1.0
// @description     A microservice for sending push notifications via Firebase Cloud Messaging (FCM) with RabbitMQ queue support
// @description     Features:
// @description     - Device registration and management
// @description     - Queue-based push notification processing
// @description     - Token validation
// @description     - Rich notifications (title, body, image, link)
// @description     - Retry mechanism with dead letter queue
// @description     - Queue statistics

// @contact.name   API Support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @schemes   http https
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "push-service/docs/swagger"
	"push-service/internal/config"
	"push-service/internal/handlers"
	"push-service/internal/platform/fcm"
	"push-service/internal/queue"
	"push-service/internal/repository"
	"push-service/internal/service"
	"push-service/pkg/database"
	"push-service/pkg/logger"
	"push-service/pkg/rabbitmq"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	if err := logger.InitGlobal(cfg.Log.Level, cfg.Log.Format); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.L().Sync()

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		logger.L().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize RabbitMQ
	rabbitmqClient, err := rabbitmq.NewRabbitMQClient(&cfg.RabbitMQ)
	if err != nil {
		logger.L().Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rabbitmqClient.Close()

	// Initialize FCM client
	fcmClient, err := fcm.NewFCMClient(&cfg.FCM)
	if err != nil {
		logger.L().Fatal("Failed to initialize FCM client", zap.Error(err))
	}

	// Create Gin router
	router := setupRouter(db, rabbitmqClient, fcmClient, cfg)

	// Create server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.L().Info("Starting server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Start queue worker
	go startPushWorker(rabbitmqClient, fcmClient, db, cfg)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.L().Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.L().Info("Server exited properly")
}

func setupRouter(db *database.DB, rabbitmqClient *rabbitmq.RabbitMQClient, fcmClient fcm.FCMClient, cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware())

	// Initialize repositories and services
	deviceRepo := repository.NewDeviceRepository(db.Pool)
	pushQueue, err := queue.NewPushQueue(rabbitmqClient, &cfg.Queue)
	if err != nil {
		logger.L().Fatal("Failed to initialize push queue", zap.Error(err))
	}

	deviceService := service.NewDeviceService(deviceRepo, fcmClient, cfg)
	pushService := service.NewPushService(deviceRepo, fcmClient, pushQueue, cfg)

	deviceHandler := handlers.NewDeviceHandler(deviceService)
	pushHandler := handlers.NewPushHandler(pushService)

	// Health check
	router.GET("/health", handlers.HealthCheck)
	router.GET("/ready", handlers.ReadinessCheck(db))

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := router.Group("/v1")
	{
		v1.POST("/devices", deviceHandler.RegisterDevice)
		v1.DELETE("/devices/:token", deviceHandler.UnregisterDevice)
		v1.GET("/devices", deviceHandler.GetUserDevices)
		v1.POST("/push/send", pushHandler.SendPush)
		v1.POST("/push/send-bulk", pushHandler.SendBulkPush)
		v1.GET("/queue/stats", pushHandler.GetQueueStats)
		v1.POST("/push/test-direct", pushHandler.TestDirectSend)
	}

	return router
}

func startPushWorker(rabbitmqClient *rabbitmq.RabbitMQClient, fcmClient fcm.FCMClient, db *database.DB, cfg *config.Config) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize repositories and services for worker
	deviceRepo := repository.NewDeviceRepository(db.Pool)
	pushQueue, err := queue.NewPushQueue(rabbitmqClient, &cfg.Queue)
	if err != nil {
		logger.L().Fatal("Failed to initialize push queue in worker", zap.Error(err))
	}
	pushService := service.NewPushService(deviceRepo, fcmClient, pushQueue, cfg)

	logger.L().Info("Starting push worker...",
		zap.Int("prefetch_count", cfg.Queue.Worker.PrefetchCount),
	)

	// Start consuming messages from internal queue
	msgs, err := pushQueue.ConsumePush(ctx)
	if err != nil {
		logger.L().Fatal("Failed to start consuming messages from internal queue", zap.Error(err))
	}

	// Process internal queue messages in a goroutine
	go func() {
		for delivery := range msgs {
			// Process each message
			if err := pushService.ProcessPushFromQueue(ctx, delivery); err != nil {
				logger.L().Error("Failed to process push message from queue",
					zap.Error(err),
					zap.Uint64("delivery_tag", delivery.DeliveryTag),
				)
			}
		}
	}()

	// Start consuming messages from API Gateway queue
	gatewayMsgs, err := pushQueue.ConsumeFromGateway(ctx)
	if err != nil {
		logger.L().Fatal("Failed to start consuming messages from gateway queue", zap.Error(err))
	}

	// Process gateway messages in a goroutine
	go func() {
		for delivery := range gatewayMsgs {
			// Process each gateway message
			if err := pushService.ProcessGatewayMessage(ctx, delivery); err != nil {
				logger.L().Error("Failed to process gateway message",
					zap.Error(err),
					zap.Uint64("delivery_tag", delivery.DeliveryTag),
				)
			}
		}
	}()

	logger.L().Info("Push workers started (internal and gateway queues)")

	// Wait for context cancellation (graceful shutdown)
	<-ctx.Done()
	logger.L().Info("Push worker shutting down...")
}

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log after request completion
		logger.L().Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
