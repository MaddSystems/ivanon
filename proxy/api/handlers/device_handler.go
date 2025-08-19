package handlers

import (
	"net/http"
	"proxy/models"
	"proxy/shared"

	"github.com/gin-gonic/gin"
)

// ListJT808Devices lists all connected JT808 devices
// @Summary List JT808 devices
// @Tags jt808
// @Produce json
// @Success 200 {array} models.JT808Device
// @Router /api/v1/jt808/devices [get]
func ListJT808Devices(c *gin.Context) {
	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()

	devices := make([]*models.JT808Device, 0, len(shared.JT808Devices))
	for _, device := range shared.JT808Devices {
		devices = append(devices, device)
	}

	c.JSON(http.StatusOK, devices)
}
