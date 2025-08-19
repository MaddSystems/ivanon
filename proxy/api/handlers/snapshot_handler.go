package handlers

import (
	"encoding/base64"
	"log"
	"net/http"
	"proxy/models"
	"proxy/services"
	"proxy/shared"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// CaptureSnapshot captures a single image from a device camera
// @Summary Capture image snapshot
// @Description Captures a single image from device camera with optimized settings
// @Tags jt808
// @Accept json
// @Produce json
// @Param device_phone query string true "Device Phone Number"
// @Param channel query int true "Camera Channel (1-4)"
// @Param resolution query int false "Resolution code (default 1)"
// @Param quality query int false "Quality 0-10 (default 0)"
// @Param timeout query int false "Timeout in seconds (default 90)"
// @Success 200 {object} models.ImageSnapshotResponse
// @Router /api/v1/jt808/snapshot [get]
func CaptureSnapshot(c *gin.Context) {
	var req models.ImageSnapshotRequest

	// Handle GET request with query parameters
	req.DevicePhone = c.Query("device_phone")
	req.Channel, _ = strconv.Atoi(c.Query("channel"))
	req.Resolution, _ = strconv.Atoi(c.Query("resolution"))
	req.Quality, _ = strconv.Atoi(c.Query("quality"))
	req.Timeout, _ = strconv.Atoi(c.Query("timeout"))
	
	if req.DevicePhone == "" || req.Channel == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_phone and channel are required"})
		return
	}

	log.Printf("[IMAGE SNAPSHOT] Request received - Device: %s, Channel: %d", req.DevicePhone, req.Channel)

	if req.Channel < 1 || req.Channel > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel must be between 1 and 4"})
		return
	}

	device, exists := services.GetJT808Device(req.DevicePhone)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if !device.Authenticated {
		log.Printf("[IMAGE SNAPSHOT] Warning: Device %s not authenticated. Proceeding...", req.DevicePhone)
	}

	// Set optimized defaults
	resolution := req.Resolution
	if resolution == 0 {
		resolution = 1
	}
	quality := req.Quality // Keep as 0 for best results
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 90
	}

	services.CleanupIncompleteSnapshots(req.DevicePhone, req.Channel)

	// Send image capture command
	err := services.SendImageCaptureCommand(req.DevicePhone, req.Channel, 1, resolution, quality, 0, 0, 0, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[IMAGE SNAPSHOT] Capture command sent - Resolution: %d, Quality: %d, Timeout: %ds", resolution, quality, timeout)

	// Wait for image data
	timeoutDuration := time.Duration(timeout) * time.Second
	timeoutTimer := time.After(timeoutDuration)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	startTime := time.Now()

	for {
		select {
		case <-timeoutTimer:
			// Handle timeout, checking for partial data
			shared.ConnMutex.Lock()
			var partialSnapshot *models.ImageSnapshot
			for _, snapshot := range shared.ActiveSnapshots {
				if snapshot.DevicePhone == req.DevicePhone && snapshot.Channel == req.Channel {
					partialSnapshot = snapshot
					break
				}
			}
			if partialSnapshot != nil {
				resp := gin.H{
					"status":          "timeout",
					"error":           "Timeout waiting for complete image",
					"chunks_received": partialSnapshot.ReceivedChunks,
					"expected_chunks": partialSnapshot.ExpectedChunks,
				}
				shared.ConnMutex.Unlock()
				c.JSON(http.StatusRequestTimeout, resp)
			} else {
				shared.ConnMutex.Unlock()
				c.JSON(http.StatusRequestTimeout, gin.H{"status": "timeout", "error": "No response from device"})
			}
			return

		case <-ticker.C:
			// Check for complete image
			shared.ConnMutex.Lock()
			var foundSnapshot *models.ImageSnapshot
			var multimediaID uint32
			for id, snapshot := range shared.ActiveSnapshots {
				if snapshot.DevicePhone == req.DevicePhone && snapshot.Channel == req.Channel && snapshot.Complete {
					foundSnapshot = snapshot
					multimediaID = id
					break
				}
			}

			if foundSnapshot != nil {
				// Image is complete, prepare and send response
				imageData := make([]byte, len(foundSnapshot.ImageData))
				copy(imageData, foundSnapshot.ImageData)

				// Clean up
				delete(shared.ActiveSnapshots, multimediaID)
				delete(shared.ImageChunks, multimediaID)
				shared.ConnMutex.Unlock()

				imageBase64 := base64.StdEncoding.EncodeToString(imageData)
				duration := time.Since(startTime)
				log.Printf("[IMAGE SNAPSHOT] SUCCESS - Device: %s, Size: %d bytes, Duration: %v", req.DevicePhone, len(imageData), duration)

				c.JSON(http.StatusOK, models.ImageSnapshotResponse{
					Status:         "success",
					ImageBase64:    imageBase64,
					ImageSize:      len(imageData),
					ChunksReceived: foundSnapshot.ReceivedChunks,
					CaptureTime:    foundSnapshot.CaptureTime.Format(time.RFC3339),
					DevicePhone:    foundSnapshot.DevicePhone,
					Channel:        foundSnapshot.Channel,
				})
				return
			}
			shared.ConnMutex.Unlock()
		}
	}
}
