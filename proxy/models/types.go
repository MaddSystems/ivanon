package models

import (
	"net"
	"time"
)

// --- API Request/Response Structs ---

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

type ImageSnapshotRequest struct {
	DevicePhone string `json:"device_phone" binding:"required"`
	Channel     int    `json:"channel" binding:"required"` // 1-4
	Resolution  int    `json:"resolution"`                 // Resolution code (default: 1)
	Quality     int    `json:"quality"`                    // Quality 0-10 (default: 0)
	Brightness  int    `json:"brightness"`                 // 0-255 (default: 0)
	Contrast    int    `json:"contrast"`                   // 0-255 (default: 0)
	Saturation  int    `json:"saturation"`                 // 0-255 (default: 0)
	Chroma      int    `json:"chroma"`                     // 0-255 (default: 0)
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

// --- Internal State Management Structs ---

type JT808Device struct {
	Conn          net.Conn  `json:"-"`
	PhoneNumber   string    `json:"phone_number"`
	LastSeen      time.Time `json:"last_seen"`
	InCall        bool      `json:"in_call"`
	Authenticated bool      `json:"authenticated"`
	RemoteAddr    string    `json:"remote_addr"`
	AuthCode      string    `json:"auth_code"`
}

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

type ImageSnapshot struct {
	MultimediaID   uint32
	DevicePhone    string
	Channel        int
	ImageData      []byte
	CaptureTime    time.Time
	Complete       bool
	ExpectedChunks int
	ReceivedChunks int
	LastChunkTime  time.Time
	TotalSize      int
}

type ImageChunk struct {
	SequenceNum uint16
	Data        []byte
	Timestamp   time.Time
}

type PendingChunk struct {
	Body          []byte
	TotalPackets  uint16
	CurrentPacket uint16
}

type ConnectionInfo struct {
	RemoteAddr string
	Protocol   string
}

// --- MQTT Message Structs ---

type TrackerData struct {
	Payload    string `json:"payload"`
	RemoteAddr string `json:"remoteaddr"`
}

type TrackerAssign struct {
	Imei       string `json:"imei"`
	Protocol   string `json:"protocol"`
	RemoteAddr string `json:"remoteaddr"`
}
