package main

import (
	"encoding/hex"
	"log"
	"net/http"
	"os"
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
	StreamType  int    `json:"stream_type"` // 0=main, 1=sub (0 is valid, so no required tag)
}

type VideoControlRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	Channel     int    `json:"channel" binding:"required"`
	Command     int    `json:"command"` // 0=stop, 1=switch, 2=pause, 3=resume (0 is valid, so no required tag)
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

var (
	activeSessions = make(map[string]*VideoSession) // maps session ID to video session
)

// @Summary Start a video stream
// @Description Initiates a video stream for a JT808 device. Channel must be 1-4. Stream type: 0=main stream (typically for storage), 1=sub-stream (used for live broadcast)
// @Tags jt808
// @Accept json
// @Produce json
// @Param request body VideoStartRequest true "Video start request"
// @Success 200 {object} VideoStartResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/jt808/video/start [post]
func StartVideoStream(c *gin.Context) {
	var req VideoStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	logMsg := "[JT808] /jt808/video/start called for device: " + req.DevicePhone + ", channel: " + strconv.Itoa(req.Channel)
	log.Println(logMsg)

	// Validate channel (1-4)
	if req.Channel < 1 || req.Channel > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel must be between 1 and 4"})
		return
	}

	// Validate stream type (0=main, 1=sub)
	if req.StreamType < 0 || req.StreamType > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Stream type must be 0 (main) or 1 (sub)"})
		return
	}

	device, exists := getJT808Device(req.DevicePhone)
	if !exists {
		log.Println("[JT808] Device not found: " + req.DevicePhone)
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if !device.Authenticated {
		log.Println("[JT808] Device not authenticated: " + req.DevicePhone)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device not authenticated"})
		return
	}

	// Check if device connection is still valid
	if device.Conn == nil {
		log.Println("[JT808] Device connection is nil: " + req.DevicePhone)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Device connection not available, please retry"})
		return
	}

	// Get video server configuration
	videoServerIP := os.Getenv("VIDEO_SERVER_IP")
	if videoServerIP == "" {
		videoServerIP = "3.13.95.26"
	}

	videoPortStr := os.Getenv("VIDEO_SERVER_PORT")
	videoPort := 7800
	if videoPortStr != "" {
		if port, err := strconv.Atoi(videoPortStr); err == nil {
			videoPort = port
		}
	}

	log.Println("videoServerIP: " + videoServerIP + ", videoPort: " + strconv.Itoa(videoPort))

	err := sendVideoStartCommand(req.DevicePhone, videoServerIP, videoPort, req.Channel, req.StreamType)
	if err != nil {
		log.Println("[JT808] Error sending video start command for device: " + req.DevicePhone + ", err: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sessionID := generateCallID() // Reuse the same ID generation function
	session := &VideoSession{
		SessionID:   sessionID,
		DevicePhone: req.DevicePhone,
		Channel:     req.Channel,
		StreamType:  req.StreamType,
		Status:      "initiated",
		StartTime:   time.Now(),
		VideoServer: videoServerIP,
		VideoPort:   videoPort,
	}

	connMutex.Lock()
	activeSessions[sessionID] = session
	connMutex.Unlock()

	response := VideoStartResponse{
		SessionID:   sessionID,
		Status:      "initiated",
		DevicePhone: req.DevicePhone,
		Channel:     req.Channel,
		StreamType:  req.StreamType,
		StartTime:   session.StartTime.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Control a video stream
// @Description Sends a control command to a video stream. Channel must be 1-4. Commands: 0=stop, 1=switch, 2=pause, 3=resume
// @Tags jt808
// @Accept json
// @Produce json
// @Param request body VideoControlRequest true "Video control request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/jt808/video/control [post]
func ControlVideoStream(c *gin.Context) {
	var req VideoControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[VIDEO CONTROL] Invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	log.Printf("[VIDEO CONTROL] Request received - Device: %s, Channel: %d, Command: %d",
		req.DevicePhone, req.Channel, req.Command)

	// Validate channel (1-4)
	if req.Channel < 1 || req.Channel > 4 {
		log.Printf("[VIDEO CONTROL] Invalid channel: %d", req.Channel)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel must be between 1 and 4"})
		return
	}

	// Validate command (0=stop, 1=switch, 2=pause, 3=resume)
	if req.Command < 0 || req.Command > 3 {
		log.Printf("[VIDEO CONTROL] Invalid command: %d", req.Command)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command. Use 0=stop, 1=switch, 2=pause, 3=resume"})
		return
	}

	device, exists := getJT808Device(req.DevicePhone)
	if !exists {
		log.Printf("[VIDEO CONTROL] Device not found: %s", req.DevicePhone)
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if !device.Authenticated {
		log.Printf("[VIDEO CONTROL] Device not authenticated: %s", req.DevicePhone)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device not authenticated"})
		return
	}

	err := sendVideoControlCommand(req.DevicePhone, req.Channel, byte(req.Command))
	if err != nil {
		log.Printf("[VIDEO CONTROL] Error sending command: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update session status if stopping
	if req.Command == 0 { // stop command
		connMutex.Lock()
		for _, session := range activeSessions {
			if session.DevicePhone == req.DevicePhone && session.Channel == req.Channel {
				session.Status = "stopped"
				log.Printf("[VIDEO CONTROL] Session %s marked as stopped", session.SessionID)
				break
			}
		}
		connMutex.Unlock()
	}

	log.Printf("[VIDEO CONTROL] Command sent successfully - Device: %s, Channel: %d, Command: %d",
		req.DevicePhone, req.Channel, req.Command)

	c.JSON(http.StatusOK, gin.H{
		"message": "Video control command sent successfully",
		"command": req.Command,
		"channel": req.Channel,
	})
}

// ListTrackers lists all connected trackers
// @Summary List all trackers
// @Description Returns a list of connected trackers
// @Tags trackers
// @Produce json
// @Success 200 {array} TrackerAssign
// @Router /api/v1/trackerlist [get]
func ListTrackers(c *gin.Context) {
	connMutex.Lock()
	trackers := make([]TrackerAssign, 0)
	for imei, connInfo := range imeiConnections {
		tracker := TrackerAssign{
			Imei:       imei,
			Protocol:   connInfo.Protocol,
			RemoteAddr: connInfo.RemoteAddr,
		}
		trackers = append(trackers, tracker)
	}
	connMutex.Unlock()
	c.JSON(http.StatusOK, trackers)
}

// SendCommand sends a hex-encoded command to a tracker by IMEI
// @Summary Send command to tracker
// @Description Sends a hex-encoded command to a tracker by IMEI
// @Tags trackers
// @Accept json
// @Produce json
// @Param request body SendCommandRequest true "Command request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/sendcommand [post]
func SendCommand(c *gin.Context) {
	var req SendCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	connMutex.Lock()
	connInfo, exists := imeiConnections[req.IMEI]
	connMutex.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tracker not found"})
		return
	}

	data, err := hex.DecodeString(req.Data)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hex data"})
		return
	}

	err = SendDataToConnection(connInfo.RemoteAddr, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Command sent successfully"})
}

// ListJT808Devices lists all connected JT808 devices
// @Summary List JT808 devices
// @Description Returns a list of connected JT808 devices
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

// StartVoIPCall initiates a VoIP call for a JT808 device
// @Summary Start a VoIP call
// @Description Initiates a VoIP call for a JT808 device
// @Tags jt808
// @Accept json
// @Produce json
// @Param request body VoIPCallRequest true "VoIP call request"
// @Success 200 {object} VoIPCallResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/jt808/call/start [post]
func StartVoIPCall(c *gin.Context) {
	var req VoIPCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	logMsg := "[JT808] /jt808/call/start called for device: " + req.DevicePhone
	log.Println(logMsg)

	device, exists := getJT808Device(req.DevicePhone)
	if !exists {
		log.Println("[JT808] Device not found: " + req.DevicePhone)
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if !device.Authenticated {
		log.Println("[JT808] Device not authenticated: " + req.DevicePhone)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device not authenticated"})
		return
	}

	if device.InCall {
		log.Println("[JT808] Device already in call: " + req.DevicePhone)
		c.JSON(http.StatusConflict, gin.H{"error": "Device already in call"})
		return
	}

	audioServerIP := os.Getenv("AUDIO_SERVER_IP")
	if audioServerIP == "" {
		audioServerIP = "127.0.0.1"
	}

	audioPortStr := os.Getenv("AUDIO_SERVER_PORT")
	audioPort := 7800
	if audioPortStr != "" {
		if port, err := strconv.Atoi(audioPortStr); err == nil {
			audioPort = port
		}
	}
	log.Println("audioServerIP: " + audioServerIP + ", audioPort: " + strconv.Itoa(audioPort))
	err := sendVoIPStartCommand(req.DevicePhone, audioServerIP, audioPort)
	if err != nil {
		log.Println("[JT808] Error sending VoIP start command for device: " + req.DevicePhone + ", err: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	callID := generateCallID()
	call := &VoIPCall{
		CallID:      callID,
		DevicePhone: req.DevicePhone,
		CallerID:    req.CallerID,
		Status:      "initiated",
		StartTime:   time.Now(),
		AudioServer: audioServerIP,
		AudioPort:   audioPort,
	}

	connMutex.Lock()
	activeCalls[callID] = call
	device.InCall = true
	connMutex.Unlock()

	response := VoIPCallResponse{
		CallID:      callID,
		Status:      "initiated",
		DevicePhone: req.DevicePhone,
		CallerID:    req.CallerID,
		StartTime:   call.StartTime.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// ControlVoIPCall sends a control command to a VoIP call
// @Summary Control a VoIP call
// @Description Sends a control command to a VoIP call (0=off, 1=switch, 2=pause, 3=resume, 4=end)
// @Tags jt808
// @Accept json
// @Produce json
// @Param request body VoIPControlRequest true "VoIP control request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/jt808/call/control [post]
func ControlVoIPCall(c *gin.Context) {
	var req VoIPControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.Command < 0 || req.Command > 4 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command. Use 0-4"})
		return
	}

	device, exists := getJT808Device(req.DevicePhone)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	err := sendVoIPControlCommand(req.DevicePhone, byte(req.Command))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Command == 4 {
		connMutex.Lock()
		device.InCall = false
		for _, call := range activeCalls {
			if call.DevicePhone == req.DevicePhone {
				call.Status = "ended"
				break
			}
		}
		connMutex.Unlock()
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Control command sent successfully",
		"command": req.Command,
	})
}

// GetCallStatus returns the current call status for a device
// @Summary Get call status
// @Description Returns the current call status for a device
// @Tags jt808
// @Param phone path string true "Device Phone Number"
// @Success 200 {object} VoIPCallResponse
// @Failure 404 {object} map[string]string
// @Router /api/v1/jt808/call/status/{phone} [get]
func GetCallStatus(c *gin.Context) {
	phone := c.Param("phone")
	connMutex.Lock()
	var currentCall *VoIPCall
	for _, call := range activeCalls {
		if call.DevicePhone == phone {
			currentCall = call
			break
		}
	}
	connMutex.Unlock()

	if currentCall == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active call found"})
		return
	}

	response := VoIPCallResponse{
		CallID:      currentCall.CallID,
		Status:      currentCall.Status,
		DevicePhone: currentCall.DevicePhone,
		CallerID:    currentCall.CallerID,
		StartTime:   currentCall.StartTime.Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, response)
}

// ListVoIPCalls lists all active VoIP calls
// @Summary List all VoIP calls
// @Description Returns a list of all active VoIP calls
// @Tags jt808
// @Produce json
// @Success 200 {array} VoIPCallResponse
// @Router /api/v1/jt808/calls [get]
func ListVoIPCalls(c *gin.Context) {
	connMutex.Lock()
	calls := make([]VoIPCallResponse, 0)
	for _, call := range activeCalls {
		response := VoIPCallResponse{
			CallID:      call.CallID,
			Status:      call.Status,
			DevicePhone: call.DevicePhone,
			CallerID:    call.CallerID,
			StartTime:   call.StartTime.Format(time.RFC3339),
		}
		calls = append(calls, response)
	}
	connMutex.Unlock()
	c.JSON(http.StatusOK, calls)
}

// ListVideoSessions lists all active video sessions
// @Summary List all video sessions
// @Description Returns a list of all active video sessions
// @Tags jt808
// @Produce json
// @Success 200 {array} VideoStartResponse
// @Router /api/v1/jt808/video/sessions [get]
func ListVideoSessions(c *gin.Context) {
	connMutex.Lock()
	sessions := make([]VideoStartResponse, 0)
	for _, session := range activeSessions {
		response := VideoStartResponse{
			SessionID:   session.SessionID,
			Status:      session.Status,
			DevicePhone: session.DevicePhone,
			Channel:     session.Channel,
			StreamType:  session.StreamType,
			StartTime:   session.StartTime.Format(time.RFC3339),
		}
		sessions = append(sessions, response)
	}
	connMutex.Unlock()
	c.JSON(http.StatusOK, sessions)
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Add CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://ivan-proxy.armaddia.lat"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Tracker endpoints
		v1.GET("/trackerlist", ListTrackers)
		v1.POST("/sendcommand", SendCommand)

		// JT808 device endpoints
		v1.GET("/jt808/devices", ListJT808Devices)

		// VoIP call endpoints
		v1.POST("/jt808/call/start", StartVoIPCall)
		v1.POST("/jt808/call/control", ControlVoIPCall)
		v1.GET("/jt808/call/status/:phone", GetCallStatus)
		v1.GET("/jt808/calls", ListVoIPCalls)

		// Video streaming endpoints - THESE WERE MISSING!
		v1.POST("/jt808/video/start", StartVideoStream)
		v1.POST("/jt808/video/control", ControlVideoStream)
		v1.GET("/jt808/video/sessions", ListVideoSessions)
	}
	return r
}
