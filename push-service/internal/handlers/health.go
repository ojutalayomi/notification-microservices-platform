package handlers

import (
	"net/http"
	"push-service/pkg/database"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"healthy"`
	Timestamp string `json:"timestamp" example:"2025-01-01T00:00:00Z"`
	Database  string `json:"database,omitempty" example:"healthy"`
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Returns the health status of the service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck godoc
// @Summary Readiness check endpoint
// @Description Returns the readiness status of the service including database connectivity
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /ready [get]
func ReadinessCheck(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dbStatus string

		if err := db.Pool.Ping(c.Request.Context()); err != nil {
			dbStatus = "unhealthy"
		} else {
			dbStatus = "healthy"
		}

		status := http.StatusOK
		if dbStatus != "healthy" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, HealthResponse{
			Status:    "ready",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Database:  dbStatus,
		})
	}
}
