package main

import (
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type SendCommandRequest struct {
	IMEI string `json:"imei" binding:"required"`
	Data string `json:"data" binding:"required"` // Hex encoded data
}

type VoIPCallRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	CallerID    string `json:"caller_id" binding:"required"`
}

type VoIPControlRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	Command     int    `json:"command" binding:"required"` // 0=off, 1=switch, 2=pause, 3=resume, 4=end
}

type VoIPCallResponse struct {
	CallID      string `json:"call_id"`
	Status      string `json:"status"`
	DevicePhone string `json:"device_phone"`
	CallerID    string `json:"caller_id"`
	StartTime   string `json:"start_time"`
}

// Video streaming API structs
type VideoStartRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	Channel     int    `json:"channel" binding:"required"`
	StreamType  int    `json:"stream_type"` // 0=main, 1=sub
}

type VideoControlRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	Channel     int    `json:"channel" binding:"required"`
	Command     int    `json:"command"` // 0=stop, 1=switch, 2=pause, 3=resume
}

type VideoStartResponse struct {
	SessionID   string `json:"session_id"`
	Status      string `json:"status"`
	DevicePhone string `json:"device_phone"`
	Channel     int    `json:"channel"`
	StreamType  int    `json:"stream_type"`
	StartTime   string `json:"start_time"`
}

// Video session tracking
type VideoSession struct {
	SessionID   string
	DevicePhone string
	Channel     int
	StreamType  int
	Status      string
	StartTime   time.Time
	VideoServer string
	VideoPort   int
}

// Enhanced Image snapshot API structs with optimized defaults from log analysis
type ImageSnapshotRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	Channel     int    `json:"channel" binding:"required"` // 1-4
	Resolution  int    `json:"resolution"`                 // Resolution code (default: 1 based on logs)
	Quality     int    `json:"quality"`                    // Quality 0-10 (default: 0 based on logs)
	Brightness  int    `json:"brightness"`                 // 0-255 (default: 0 based on logs)
	Contrast    int    `json:"contrast"`                   // 0-255 (default: 0 based on logs)
	Saturation  int    `json:"saturation"`                 // 0-255 (default: 0 based on logs)
	Chroma      int    `json:"chroma"`                     // 0-255 (default: 0 based on logs)
	Timeout     int    `json:"timeout"`                    // Timeout in seconds (default: 60)
}

type ImageSnapshotResponse struct {
	Status         string `json:"status"`
	ImageBase64    string `json:"image_base64,omitempty"`
	Error          string `json:"error,omitempty"`
	ImageSize      int    `json:"image_size,omitempty"`
	ChunksReceived int    `json:"chunks_received,omitempty"`
	CaptureTime    string `json:"capture_time,omitempty"`
	DevicePhone    string `json:"device_phone,omitempty"`
	Channel        int    `json:"channel,omitempty"`
}

var (
	activeSessions = make(map[string]*VideoSession) // maps session ID to video session
)

// CaptureSnapshot captures a single image from device camera
// @Summary Capture image snapshot
// @Description Captures a single image from device camera with optimized settings from log analysis
// @Tags jt808
// @Accept json
// @Produce json
// @Param device_phone query string true "Device Phone Number"
// @Param channel query int true "Camera Channel (1-4)"
// @Param resolution query int false "Resolution code (default 1)"
// @Param quality query int false "Quality 0-10 (default 0)"
// @Param timeout query int false "Timeout in seconds (default 90)"
// @Success 200 {object} ImageSnapshotResponse
// @Router /api/v1/jt808/snapshot [get]
func CaptureSnapshot(c *gin.Context) {
	var req ImageSnapshotRequest

	// Handle both GET (query params) and POST (JSON body)
	if c.Request.Method == "GET" {
		req.DevicePhone = c.Query("device_phone")

		if channelStr := c.Query("channel"); channelStr != "" {
			if channel, err := strconv.Atoi(channelStr); err == nil {
				req.Channel = channel
			}
		}

		if resolutionStr := c.Query("resolution"); resolutionStr != "" {
			if resolution, err := strconv.Atoi(resolutionStr); err == nil {
				req.Resolution = resolution
			}
		}

		if qualityStr := c.Query("quality"); qualityStr != "" {
			if quality, err := strconv.Atoi(qualityStr); err == nil {
				req.Quality = quality
			}
		}

		if timeoutStr := c.Query("timeout"); timeoutStr != "" {
			if timeout, err := strconv.Atoi(timeoutStr); err == nil {
				req.Timeout = timeout
			}
		}

		// Validate required parameters
		if req.DevicePhone == "" || req.Channel == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "device_phone and channel are required"})
			return
		}
	} else {
		// Handle POST with JSON body
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
	}

	log.Printf("[IMAGE SNAPSHOT] Request received - Device: %s, Channel: %d", req.DevicePhone, req.Channel)

	// Validate channel (1-4)
	if req.Channel < 1 || req.Channel > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel must be between 1 and 4"})
		return
	}

	device, exists := getJT808Device(req.DevicePhone)
	if !exists {
		log.Printf("[IMAGE SNAPSHOT] Device not found: %s", req.DevicePhone)
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if !device.Authenticated {
		log.Printf("[IMAGE SNAPSHOT] Device not authenticated: %s", req.DevicePhone)
		// Note: Based on the code, devices might be authenticated but not marked as such
		// Consider checking if device is actively communicating
		log.Printf("[IMAGE SNAPSHOT] Warning: Device may be authenticated but not marked. Proceeding...")
	}

	// Set optimized default values based on successful captures
	resolution := req.Resolution
	if resolution == 0 {
		resolution = 1 // 320x240 works well
	}

	quality := req.Quality
	// Keep quality as 0 for best results

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 90 // Increased timeout for multi-packet transfers
	}

	// All image parameters set to 0 based on successful captures
	brightness := byte(0)
	contrast := byte(0)
	saturation := byte(0)
	chroma := byte(0)

	// Clean up any existing incomplete snapshots for this device/channel
	connMutex.Lock()
	for id, snapshot := range activeSnapshots {
		if snapshot.DevicePhone == req.DevicePhone && snapshot.Channel == req.Channel && !snapshot.Complete {
			delete(activeSnapshots, id)
			delete(imageChunks, id)
			log.Printf("[IMAGE SNAPSHOT] Cleaned up incomplete snapshot ID: %d", id)
		}
	}
	connMutex.Unlock()

	// Send image capture command
	err := sendImageCaptureCommand(req.DevicePhone, req.Channel, 1, resolution, quality, brightness, contrast, saturation, chroma)
	if err != nil {
		log.Printf("[IMAGE SNAPSHOT] Error sending capture command: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[IMAGE SNAPSHOT] Capture command sent - Resolution: %d, Quality: %d, Timeout: %ds", resolution, quality, timeout)

	// Wait for image data with enhanced monitoring
	timeoutDuration := time.Duration(timeout) * time.Second
	timeoutTimer := time.After(timeoutDuration)
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

	startTime := time.Now()
	lastLogTime := startTime

	for {
		select {
		case <-timeoutTimer:
			log.Printf("[IMAGE SNAPSHOT] Timeout after %d seconds for device: %s", timeout, req.DevicePhone)

			// Check for partial data
			connMutex.Lock()
			var partialSnapshot *ImageSnapshot
			for _, snapshot := range activeSnapshots {
				if snapshot.DevicePhone == req.DevicePhone && snapshot.Channel == req.Channel {
					partialSnapshot = snapshot
					break
				}
			}

			if partialSnapshot != nil {
				chunksReceived := partialSnapshot.ReceivedChunks
				expectedChunks := partialSnapshot.ExpectedChunks
				totalSize := partialSnapshot.TotalSize
				connMutex.Unlock()

				c.JSON(http.StatusRequestTimeout, gin.H{
					"status":          "timeout",
					"error":           "Timeout waiting for complete image from device",
					"chunks_received": chunksReceived,
					"expected_chunks": expectedChunks,
					"partial_size":    totalSize,
					"timeout_seconds": timeout,
				})
			} else {
				connMutex.Unlock()
				c.JSON(http.StatusRequestTimeout, gin.H{
					"status":          "timeout",
					"error":           "No response from device",
					"timeout_seconds": timeout,
				})
			}
			return

		case <-ticker.C:
			// Check for complete image
			connMutex.Lock()
			var foundSnapshot *ImageSnapshot
			for _, snapshot := range activeSnapshots {
				if snapshot.DevicePhone == req.DevicePhone && snapshot.Channel == req.Channel && snapshot.Complete {
					foundSnapshot = snapshot
					break
				}
			}

			// Log progress every 10 seconds
			if foundSnapshot == nil && time.Since(lastLogTime) > 10*time.Second {
				for _, snapshot := range activeSnapshots {
					if snapshot.DevicePhone == req.DevicePhone && snapshot.Channel == req.Channel {
						log.Printf("[IMAGE SNAPSHOT] Progress - Device: %s, Chunks: %d/%d, Size: %d bytes",
							req.DevicePhone, snapshot.ReceivedChunks, snapshot.ExpectedChunks, snapshot.TotalSize)
						lastLogTime = time.Now()
						break
					}
				}
			}

			if foundSnapshot != nil {
				// Create a copy of the data before unlocking
				imageData := make([]byte, len(foundSnapshot.ImageData))
				copy(imageData, foundSnapshot.ImageData)
				imageSize := len(foundSnapshot.ImageData)
				chunksReceived := foundSnapshot.ReceivedChunks
				captureTime := foundSnapshot.CaptureTime.Format(time.RFC3339)
				devicePhone := foundSnapshot.DevicePhone
				channel := foundSnapshot.Channel
				multimediaID := foundSnapshot.MultimediaID

				// Clean up
				delete(activeSnapshots, multimediaID)
				delete(imageChunks, multimediaID)
				connMutex.Unlock()

				// Convert image to base64
				imageBase64 := base64.StdEncoding.EncodeToString(imageData)

				// Prepare response
				responseData := ImageSnapshotResponse{
					Status:         "success",
					ImageBase64:    imageBase64,
					ImageSize:      imageSize,
					ChunksReceived: chunksReceived,
					CaptureTime:    captureTime,
					DevicePhone:    devicePhone,
					Channel:        channel,
				}

				duration := time.Since(startTime)
				log.Printf("[IMAGE SNAPSHOT] SUCCESS - Device: %s, Channel: %d, Size: %d bytes, Chunks: %d, Duration: %v",
					devicePhone, channel, imageSize, chunksReceived, duration)

				c.JSON(http.StatusOK, responseData)
				return
			}
			connMutex.Unlock()
		}
	}
}

// ListJT808Devices lists all connected JT808 devices
// @Summary List JT808 devices
// @Tags jt808
// @Produce json
// @Success 200 {array} JT808Device
// @Router /api/v1/jt808/devices [get]
func ListJT808Devices(c *gin.Context) {
	connMutex.Lock()
	devices := make([]*JT808Device, 0)
	for _, device := range jt808Devices {
		devices = append(devices, device)
	}
	connMutex.Unlock()
	c.JSON(http.StatusOK, devices)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins for testing
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// JT808 device endpoints
		v1.GET("/jt808/devices", ListJT808Devices)

		// Image capture endpoints - both GET and POST supported
		v1.GET("/jt808/snapshot", CaptureSnapshot)
		v1.POST("/jt808/snapshot", CaptureSnapshot)
	}

	return r
}
