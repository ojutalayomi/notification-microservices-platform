package handlers

import (
	"net/http"
	"push-service/internal/models"
	"push-service/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterDeviceRequest represents the device registration request
// @Description Device registration request
type RegisterDeviceRequest struct {
	UserID   string `json:"user_id" binding:"required" example:"user123"`
	Token    string `json:"token" binding:"required" example:"fcm_token_here"`
	Platform string `json:"platform" binding:"required,oneof=ios android web" example:"android"`
}

// RegisterDeviceResponse represents the device registration response
// @Description Device registration response
type RegisterDeviceResponse struct {
	Message string                `json:"message" example:"Device registered successfully"`
	Device  models.DeviceResponse `json:"device"`
}

// GetUserDevicesResponse represents the user devices response
// @Description User devices response
type GetUserDevicesResponse struct {
	UserID  string                  `json:"user_id" example:"user123"`
	Devices []models.DeviceResponse `json:"devices"`
	Count   int                     `json:"count" example:"2"`
}

type DeviceHandler struct {
	deviceService service.DeviceService
}

func NewDeviceHandler(deviceService service.DeviceService) *DeviceHandler {
	return &DeviceHandler{deviceService: deviceService}
}

// RegisterDevice godoc
// @Summary Register a new device
// @Description Register a device token for push notifications
// @Tags devices
// @Accept json
// @Produce json
// @Param request body models.CreateDeviceRequest true "Device registration request"
// @Success 201 {object} RegisterDeviceResponse
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 500 {object} map[string]string "Failed to register device"
// @Router /v1/devices [post]
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
	var req models.CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	device, err := h.deviceService.RegisterDevice(c.Request.Context(), req)
	if err != nil {
		zap.L().Error("Failed to register device", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Device registered successfully",
		"device":  device,
	})
}

// UnregisterDevice godoc
// @Summary Unregister a device
// @Description Unregister a device token (soft delete)
// @Tags devices
// @Accept json
// @Produce json
// @Param token path string true "Device token"
// @Success 200 {object} map[string]string "Device unregistered successfully"
// @Failure 400 {object} map[string]string "Device token is required"
// @Failure 500 {object} map[string]string "Failed to unregister device"
// @Router /v1/devices/{token} [delete]
func (h *DeviceHandler) UnregisterDevice(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device token is required"})
		return
	}

	err := h.deviceService.UnregisterDevice(c.Request.Context(), token)
	if err != nil {
		zap.L().Error("Failed to unregister device", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unregister device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device unregistered successfully"})
}

// GetUserDevices godoc
// @Summary Get user devices
// @Description Get all registered devices for a user
// @Tags devices
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {object} GetUserDevicesResponse
// @Failure 400 {object} map[string]string "User ID is required"
// @Failure 500 {object} map[string]string "Failed to get user devices"
// @Router /v1/devices [get]
func (h *DeviceHandler) GetUserDevices(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	devices, err := h.deviceService.GetUserDevices(c.Request.Context(), userID)
	if err != nil {
		zap.L().Error("Failed to get user devices", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user devices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"devices": devices,
		"count":   len(devices),
	})
}
