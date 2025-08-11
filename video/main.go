package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	clientsMu sync.RWMutex
	clients   = make(map[*websocket.Conn]*ClientInfo)

	// TCP connections to devices
	deviceConnsMu sync.RWMutex
	deviceConns   = make(map[string]net.Conn)

	// Debug flags
	debugSend    bool
	debugReceive bool
	sendCounter  int64
	recvCounter  int64
	audioCounter int64 // Track audio frames processed

	// API proxy configuration
	apiBaseURL = "https://ivan-proxy.armaddia.lat"

	// Video frame reassembly - CRITICAL FIX: Use timestamp-based grouping
	videoFramesMu sync.RWMutex
	videoFrames   = make(map[string]*VideoFrameAssembler) // key: channel_timestamp

	// SPS/PPS storage for H.264
	spsPpsStoreMu sync.RWMutex
	spsPpsStore   = make(map[int]*SPSPPSData) // channel -> SPS/PPS data
)

type SPSPPSData struct {
	SPS []byte
	PPS []byte
}

type ClientInfo struct {
	RemoteAddr   string
	ConnectedAt  time.Time
	FramesSent   int64
	LastActivity time.Time
	ClientType   string // "receiver", "transmitter", "video"
	WriteMutex   sync.Mutex
}

type AudioFrame struct {
	PCMData   []byte
	Duration  float32
	Timestamp time.Time
}

type VideoFrameAssembler struct {
	Channel         int
	SequenceNum     uint16
	FrameType       int        // 0=I-frame, 1=P-frame, 2=B-frame
	Fragments       []Fragment // Store fragments with their metadata
	TotalSize       int
	ReceivedSize    int
	LastUpdate      time.Time
	IsComplete      bool
	JT1078Timestamp uint64 // JT1078 timestamp for grouping fragments
}

type Fragment struct {
	SubType     int
	SequenceNum uint16
	Data        []byte
}

type VideoFrame struct {
	Channel     int
	SequenceNum uint16
	FrameType   int
	Data        []byte
	Timestamp   time.Time
}

type FrameBuffer struct {
	frames  []*AudioFrame
	mutex   sync.Mutex
	maxSize int
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:    1024,
	WriteBufferSize:   4096,
	EnableCompression: false,
}

var frameBuffer = &FrameBuffer{
	frames:  make([]*AudioFrame, 0),
	maxSize: 15,
}

var alawToPcmTable [256]int16

func init() {
	for i := 0; i < 256; i++ {
		alawToPcmTable[i] = computeG711ALawToPCM16(byte(i))
	}
}

func computeG711ALawToPCM16(aLawByte byte) int16 {
	aLawByte ^= 0x55
	t := int((aLawByte&0x0F)<<4) + 8
	seg := int((aLawByte & 0x70) >> 4)
	if seg >= 1 {
		t += 0x100
	}
	if seg > 1 {
		t <<= (seg - 1)
	}
	if (aLawByte & 0x80) == 0 {
		return int16(t)
	}
	return int16(-t)
}

func DecodeG711ALaw(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	pcm := make([]byte, len(data)*2)
	for i, b := range data {
		sample := alawToPcmTable[b]

		if sample < 80 && sample > -80 {
			sample = 0
		}

		if sample > 30000 {
			sample = 30000
		} else if sample < -30000 {
			sample = -30000
		}

		pcm[2*i] = byte(sample)
		pcm[2*i+1] = byte(sample >> 8)
	}
	return pcm
}

// Extract SPS/PPS from payload (proven algorithm from stream-capture)
func extractSPSPPS(payload []byte, channel int) {
	spsPpsStoreMu.Lock()
	defer spsPpsStoreMu.Unlock()

	if _, exists := spsPpsStore[channel]; !exists {
		spsPpsStore[channel] = &SPSPPSData{}
	}

	for i := 0; i < len(payload)-4; {
		// Find NAL unit start code (00 00 01 or 00 00 00 01)
		if !(payload[i] == 0 && payload[i+1] == 0 && (payload[i+2] == 1 || (payload[i+2] == 0 && payload[i+3] == 1))) {
			i++
			continue
		}

		// Determine start position and NAL type
		startOffset := 3
		if payload[i+2] == 0 {
			startOffset = 4
		}
		nalStart := i + startOffset
		if nalStart >= len(payload) {
			break
		}
		nalType := payload[nalStart] & 0x1F

		// If it's SPS or PPS, find its end and store it
		if nalType == 7 || nalType == 8 {
			nalEnd := len(payload)
			for j := nalStart + 1; j < len(payload)-4; j++ {
				if payload[j] == 0 && payload[j+1] == 0 && (payload[j+2] == 1 || (payload[j+2] == 0 && payload[j+3] == 1)) {
					nalEnd = j
					break
				}
			}
			// Re-add the 4-byte start code for the file
			nalData := append([]byte{0x00, 0x00, 0x00, 0x01}, payload[nalStart:nalEnd]...)
			if nalType == 7 {
				spsPpsStore[channel].SPS = nalData
				if debugReceive {
					fmt.Printf("📍 Found SPS for channel %d (length: %d bytes)\n", channel, len(nalData))
				}
			} else {
				spsPpsStore[channel].PPS = nalData
				if debugReceive {
					fmt.Printf("📍 Found PPS for channel %d (length: %d bytes)\n", channel, len(nalData))
				}
			}
			i = nalEnd
		} else {
			i++
		}
	}
}

// Video API Endpoints (keep existing)
func apiProxyVideoStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading video start request: %v", err)
		http.Error(w, "Error reading request", http.StatusBadRequest)
		return
	}

	log.Printf("API Proxy: Starting video with payload: %s", string(body))

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/jt808/video/start", apiBaseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Printf("Error starting video: %v", err)
		http.Error(w, fmt.Sprintf("Error starting video: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading video start response: %v", err)
		http.Error(w, fmt.Sprintf("Error reading response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(responseBody)
	log.Printf("API Proxy: Video start response forwarded successfully")
}

func apiProxyVideoControl(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading video control request: %v", err)
		http.Error(w, "Error reading request", http.StatusBadRequest)
		return
	}

	log.Printf("API Proxy: Video control with payload: %s", string(body))

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/jt808/video/control", apiBaseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Printf("Error controlling video: %v", err)
		http.Error(w, fmt.Sprintf("Error controlling video: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading video control response: %v", err)
		http.Error(w, fmt.Sprintf("Error reading response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(responseBody)
	log.Printf("API Proxy: Video control response forwarded successfully")
}

// Keep existing API proxy handlers
func apiProxyDevicesShort(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("API Proxy: Getting devices from %s/api/v1/jt808/devices", apiBaseURL)

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/jt808/devices", apiBaseURL))
	if err != nil {
		log.Printf("Error fetching devices: %v", err)
		http.Error(w, fmt.Sprintf("Error fetching devices: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading devices response: %v", err)
		http.Error(w, fmt.Sprintf("Error reading response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(body)
	log.Printf("API Proxy: Devices response forwarded successfully")
}

func apiProxyCallStartShort(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading call start request: %v", err)
		http.Error(w, "Error reading request", http.StatusBadRequest)
		return
	}

	log.Printf("API Proxy: Starting call with payload: %s", string(body))

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/jt808/call/start", apiBaseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Printf("Error starting call: %v", err)
		http.Error(w, fmt.Sprintf("Error starting call: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading call start response: %v", err)
		http.Error(w, fmt.Sprintf("Error reading response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(responseBody)
	log.Printf("API Proxy: Call start response forwarded successfully")
}

func apiProxyCallControlShort(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading call control request: %v", err)
		http.Error(w, "Error reading request", http.StatusBadRequest)
		return
	}

	log.Printf("API Proxy: Call control with payload: %s", string(body))

	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/jt808/call/control", apiBaseURL),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Printf("Error controlling call: %v", err)
		http.Error(w, fmt.Sprintf("Error controlling call: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading call control response: %v", err)
		http.Error(w, fmt.Sprintf("Error reading response: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)
	w.Write(responseBody)
	log.Printf("API Proxy: Call control response forwarded successfully")
}

func decodeBCD(data []byte) string {
	result := ""
	for _, b := range data {
		high := (b & 0xF0) >> 4
		low := b & 0x0F
		if high <= 9 {
			result += fmt.Sprintf("%d", high)
		}
		if low <= 9 {
			result += fmt.Sprintf("%d", low)
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (fb *FrameBuffer) AddFrame(frame *AudioFrame) {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	if len(fb.frames) >= fb.maxSize {
		fb.frames = fb.frames[1:]
	}
	fb.frames = append(fb.frames, frame)
}

func (fb *FrameBuffer) GetFrames() []*AudioFrame {
	fb.mutex.Lock()
	defer fb.mutex.Unlock()

	if len(fb.frames) < 3 {
		return nil
	}

	frames := make([]*AudioFrame, len(fb.frames))
	copy(frames, fb.frames)
	fb.frames = fb.frames[:0]
	return frames
}

func addClient(conn *websocket.Conn, remoteAddr string, clientType string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[conn] = &ClientInfo{
		RemoteAddr:   remoteAddr,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		ClientType:   clientType,
		WriteMutex:   sync.Mutex{},
	}
	log.Printf("%s client connected: %s", clientType, remoteAddr)
}

func removeClient(conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	if info, exists := clients[conn]; exists {
		log.Printf("%s client disconnected: %s", info.ClientType, info.RemoteAddr)
		delete(clients, conn)
	}
}

// WebSocket handler for video reception
func videoHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Video WebSocket upgrade error: %v", err)
		return
	}

	addClient(conn, r.RemoteAddr, "video")
	defer func() {
		conn.Close()
		removeClient(conn)
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			clientsMu.RLock()
			info, exists := clients[conn]
			clientsMu.RUnlock()

			if !exists {
				return
			}

			info.WriteMutex.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			info.WriteMutex.Unlock()

			if err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func broadcastFrames() {
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		frames := frameBuffer.GetFrames()
		if len(frames) == 0 {
			continue
		}

		if debugSend {
			log.Printf("[AUDIO-DEBUG] Broadcasting %d audio frames from buffer", len(frames))
		}

		var combinedG711A []byte
		var totalDuration float32

		for _, frame := range frames {
			combinedG711A = append(combinedG711A, frame.PCMData...) // Now contains G.711A data
			totalDuration += frame.Duration
		}

		if len(combinedG711A) == 0 {
			if debugSend {
				log.Printf("[AUDIO-DEBUG] No G.711A data to send after combining frames")
			}
			continue
		}

		if totalDuration < 0.035 {
			totalDuration = 0.04
		}

		if debugSend {
			log.Printf("[AUDIO-DEBUG] Sending audio: CombinedG711ASize=%d, TotalDuration=%.3fs",
				len(combinedG711A), totalDuration)
		}

		durBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(durBytes, math.Float32bits(totalDuration))
		message := append(durBytes, combinedG711A...)

		clientsMu.RLock()
		var clientsToSend []*websocket.Conn
		var clientInfos []*ClientInfo

		for client, info := range clients {
			if info.ClientType == "receiver" {
				clientsToSend = append(clientsToSend, client)
				clientInfos = append(clientInfos, info)
			}
		}
		clientsMu.RUnlock()

		if debugSend {
			log.Printf("[AUDIO-DEBUG] Found %d receiver clients to send audio data", len(clientsToSend))
		}

		var disconnectedClients []*websocket.Conn
		sentCount := 0
		for i, client := range clientsToSend {
			info := clientInfos[i]

			info.WriteMutex.Lock()
			client.SetWriteDeadline(time.Now().Add(1 * time.Second))
			err := client.WriteMessage(websocket.BinaryMessage, message)
			info.WriteMutex.Unlock()

			if err != nil {
				disconnectedClients = append(disconnectedClients, client)
				if debugSend {
					log.Printf("[AUDIO-DEBUG] Failed to send audio to client %d: %v", i, err)
				}
			} else {
				info.FramesSent++
				info.LastActivity = time.Now()
				sentCount++
			}
		}

		if debugSend && sentCount > 0 {
			log.Printf("[AUDIO-DEBUG] Successfully sent audio data to %d clients (message size: %d bytes)",
				sentCount, len(message))
		}

		if len(disconnectedClients) > 0 {
			clientsMu.Lock()
			for _, client := range disconnectedClients {
				client.Close()
				delete(clients, client)
			}
			clientsMu.Unlock()
		}
	}
}

// CRITICAL FIX: Broadcast video frames with SPS/PPS
func broadcastVideoFrame(videoFrame *VideoFrame) {
	clientsMu.RLock()
	var videoClients []*websocket.Conn
	var clientInfos []*ClientInfo

	for client, info := range clients {
		if info.ClientType == "video" {
			videoClients = append(videoClients, client)
			clientInfos = append(clientInfos, info)
		}
	}
	clientsMu.RUnlock()

	if len(videoClients) == 0 {
		return
	}

	// Create video message with SPS/PPS if needed
	message := createVideoMessageWithSPSPPS(videoFrame)

	var disconnectedClients []*websocket.Conn
	for i, client := range videoClients {
		info := clientInfos[i]

		info.WriteMutex.Lock()
		client.SetWriteDeadline(time.Now().Add(1 * time.Second))
		err := client.WriteMessage(websocket.BinaryMessage, message)
		info.WriteMutex.Unlock()

		if err != nil {
			disconnectedClients = append(disconnectedClients, client)
		} else {
			info.FramesSent++
			info.LastActivity = time.Now()
		}
	}

	if len(disconnectedClients) > 0 {
		clientsMu.Lock()
		for _, client := range disconnectedClients {
			client.Close()
			delete(clients, client)
		}
		clientsMu.Unlock()
	}
}

// CRITICAL FIX: Include SPS/PPS with I-frames
func createVideoMessageWithSPSPPS(videoFrame *VideoFrame) []byte {
	frameData := videoFrame.Data

	// For I-frames, prepend SPS/PPS if available and not already present
	if videoFrame.FrameType == 0 { // I-frame
		spsPpsStoreMu.RLock()
		spsPpsData, exists := spsPpsStore[videoFrame.Channel]
		spsPpsStoreMu.RUnlock()

		if exists && spsPpsData.SPS != nil && spsPpsData.PPS != nil {
			// Check if frame already contains SPS/PPS
			hasSPS := false
			hasPPS := false

			// Quick check for SPS/PPS NAL units
			for i := 0; i < len(frameData)-5 && i < 100; i++ {
				if frameData[i] == 0 && frameData[i+1] == 0 &&
					(frameData[i+2] == 1 || (frameData[i+2] == 0 && frameData[i+3] == 1)) {
					nalStart := i + 3
					if frameData[i+2] == 0 {
						nalStart = i + 4
					}
					if nalStart < len(frameData) {
						nalType := frameData[nalStart] & 0x1F
						if nalType == 7 {
							hasSPS = true
						} else if nalType == 8 {
							hasPPS = true
						}
					}
				}
			}

			// Prepend SPS/PPS if not present
			if !hasSPS || !hasPPS {
				// Add Access Unit Delimiter first
				aud := []byte{0x00, 0x00, 0x00, 0x01, 0x09, 0x10}
				totalLen := len(aud) + len(spsPpsData.SPS) + len(spsPpsData.PPS) + len(frameData)
				newData := make([]byte, totalLen)
				offset := 0
				copy(newData[offset:], aud)
				offset += len(aud)
				copy(newData[offset:], spsPpsData.SPS)
				offset += len(spsPpsData.SPS)
				copy(newData[offset:], spsPpsData.PPS)
				offset += len(spsPpsData.PPS)
				copy(newData[offset:], frameData)
				frameData = newData

				if debugSend {
					fmt.Printf("📹 Added SPS/PPS to I-frame for channel %d\n", videoFrame.Channel)
				}
			}
		}
	}

	// Message format: [channel:1][frameType:1][seqNum:2][dataLength:4][data...]
	messageSize := 1 + 1 + 2 + 4 + len(frameData)
	message := make([]byte, messageSize)

	message[0] = byte(videoFrame.Channel)
	message[1] = byte(videoFrame.FrameType)
	binary.BigEndian.PutUint16(message[2:4], videoFrame.SequenceNum)
	binary.BigEndian.PutUint32(message[4:8], uint32(len(frameData)))
	copy(message[8:], frameData)

	return message
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	addClient(conn, r.RemoteAddr, "receiver")
	defer func() {
		conn.Close()
		removeClient(conn)
	}()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	go func() {
		for range pingTicker.C {
			clientsMu.RLock()
			info, exists := clients[conn]
			clientsMu.RUnlock()

			if !exists {
				return
			}

			info.WriteMutex.Lock()
			err := conn.WriteMessage(websocket.PingMessage, nil)
			info.WriteMutex.Unlock()

			if err != nil {
				return
			}
		}
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func transmitHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Transmit WebSocket upgrade error: %v", err)
		return
	}

	addClient(conn, r.RemoteAddr, "transmitter")
	defer func() {
		conn.Close()
		removeClient(conn)
	}()

	log.Printf("Audio transmitter connected: %s", r.RemoteAddr)

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Transmit read error: %v", err)
			break
		}

		if messageType == websocket.BinaryMessage {
			forwardToDevices(data)
		}
	}
}

func forwardToDevices(frameData []byte) {
	deviceConnsMu.RLock()
	defer deviceConnsMu.RUnlock()

	sentCount := 0
	for remoteAddr, deviceConn := range deviceConns {
		deviceConn.SetWriteDeadline(time.Now().Add(2 * time.Second))

		if _, err := deviceConn.Write(frameData); err != nil {
			log.Printf("Failed to send to device %s: %v", remoteAddr, err)
		} else {
			sentCount++
		}
	}

	if debugSend && sentCount > 0 {
		fmt.Printf("📡 Forwarded frame to %d device(s)\n", sentCount)
	}
}

func main() {
	flag.BoolVar(&debugSend, "ds", false, "Debug send: show details of frames being sent to devices")
	flag.BoolVar(&debugReceive, "dr", false, "Debug receive: show details of frames received from devices")
	flag.Parse()

	if debugSend {
		fmt.Println("🐛 DEBUG SEND MODE ENABLED - Will show details of outgoing frames")
	}
	if debugReceive {
		fmt.Println("🐛 DEBUG RECEIVE MODE ENABLED - Will show details of incoming frames")
	}

	go broadcastFrames()
	go startTCPServer()
	go cleanupVideoFrames()

	// Audio WebSocket endpoints
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/transmit", transmitHandler)

	// Video WebSocket endpoint
	http.HandleFunc("/video", videoHandler)

	// Video API endpoints
	http.HandleFunc("/api/video/start", apiProxyVideoStart)
	http.HandleFunc("/api/video/control", apiProxyVideoControl)

	// Audio API endpoints (existing)
	http.HandleFunc("/api/devices", apiProxyDevicesShort)
	http.HandleFunc("/api/call/start", apiProxyCallStartShort)
	http.HandleFunc("/api/call/control", apiProxyCallControlShort)

	// Static file server
	http.Handle("/", http.FileServer(http.Dir(".")))

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		clientsMu.RLock()
		receiverCount := 0
		transmitterCount := 0
		videoCount := 0
		for _, info := range clients {
			switch info.ClientType {
			case "receiver":
				receiverCount++
			case "transmitter":
				transmitterCount++
			case "video":
				videoCount++
			}
		}
		clientsMu.RUnlock()

		deviceConnsMu.RLock()
		deviceCount := len(deviceConns)
		deviceConnsMu.RUnlock()

		fmt.Fprintf(w, "OK - Audio Receivers: %d, Transmitters: %d, Video: %d, Devices: %d",
			receiverCount, transmitterCount, videoCount, deviceCount)
	})

	fmt.Println("🚀 JT1078 Two-Way Audio + Video Streamer with API Proxy")
	fmt.Println("📄 Web interface: http://localhost:8081")
	fmt.Println("🔊 TCP audio/video port: 7800")
	fmt.Println("📻 Audio Reception: ws://localhost:8081/ws")
	fmt.Println("🎤 Audio Transmission: ws://localhost:8081/transmit")
	fmt.Println("📺 Video Reception: ws://localhost:8081/video")
	fmt.Println("🔗 API Proxy endpoints:")
	fmt.Println("   • GET  /api/devices")
	fmt.Println("   • POST /api/call/start")
	fmt.Println("   • POST /api/call/control")
	fmt.Println("   • POST /api/video/start")
	fmt.Println("   • POST /api/video/control")

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func startTCPServer() {
	ln, err := net.Listen("tcp", "0.0.0.0:7800")
	if err != nil {
		log.Fatalf("TCP listen error: %v", err)
	}
	defer ln.Close()

	log.Println("TCP server listening on 0.0.0.0:7800")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	defer conn.Close()

	// Register device connection
	deviceConnsMu.Lock()
	deviceConns[remoteAddr] = conn
	deviceConnsMu.Unlock()

	defer func() {
		deviceConnsMu.Lock()
		delete(deviceConns, remoteAddr)
		deviceConnsMu.Unlock()
	}()

	buffer := make([]byte, 0, 16384)
	readBuffer := make([]byte, 8192)

	log.Printf("New TCP device connection: %s", remoteAddr)

	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, err := conn.Read(readBuffer)

		if n > 0 {
			buffer = append(buffer, readBuffer[:n]...)
		}

		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("Read timeout from device %s", remoteAddr)
			}
			break
		}

		// Process frames
		for {
			frame, consumed, frameType := extractFrame(buffer)
			if consumed == 0 {
				break
			}

			buffer = buffer[consumed:]

			if frame != nil {
				if debugSend && frameType == "audio" {
					log.Printf("[AUDIO-DEBUG] Dispatching audio frame for processing")
				}
				switch frameType {
				case "audio":
					processAudioFrame(frame)
				case "video":
					processVideoFrame(frame)
				}
			}

			if len(buffer) > 32768 {
				buffer = buffer[len(buffer)-1024:]
			}
		}
	}

	log.Printf("TCP device disconnected: %s", remoteAddr)
}

type JT1078Frame struct {
	Header          []byte
	Channel         int
	SequenceNum     uint16
	DataType        int
	SubType         int
	DataLength      uint16
	Payload         []byte
	Timestamp       time.Time
	JT1078Timestamp uint64
}

// CRITICAL FIX: Use proven extraction algorithm from stream-capture
func extractFrame(buffer []byte) (*JT1078Frame, int, string) {
	headerIdx := -1
	for i := 0; i <= len(buffer)-4; i++ {
		if buffer[i] == 0x30 && buffer[i+1] == 0x31 &&
			buffer[i+2] == 0x63 && buffer[i+3] == 0x64 {
			headerIdx = i
			break
		}
	}

	if headerIdx == -1 {
		if len(buffer) > 4 {
			return nil, len(buffer) - 4, ""
		}
		return nil, 0, ""
	}

	if len(buffer)-headerIdx < 28 {
		return nil, 0, ""
	}

	frameData := buffer[headerIdx:]

	// Parse data type from label3 byte (15) - PROVEN ALGORITHM
	label3 := frameData[15]
	dataType := int(label3&0xF0) >> 4
	subPackageType := int(label3 & 0x0F)

	// Calculate offset based on data type (CRITICAL FIX from stream-capture)
	offset := 16 // Start after the basic header

	// For non-transparent data, skip timestamp (8 bytes)
	if dataType != 4 {
		if len(frameData) < offset+8 {
			return nil, 0, ""
		}
		offset += 8
	}

	// For video frames (not audio, not transparent), skip additional header (4 bytes)
	if dataType != 4 && dataType != 3 {
		if len(frameData) < offset+4 {
			return nil, 0, ""
		}
		offset += 4
	}

	// Get payload length
	if len(frameData) < offset+2 {
		return nil, 0, ""
	}
	payloadLength := binary.BigEndian.Uint16(frameData[offset : offset+2])
	offset += 2

	// Sanity check payload length
	if payloadLength > 8192 {
		return nil, headerIdx + 28, ""
	}

	totalFrameSize := offset + int(payloadLength)
	if len(frameData) < totalFrameSize {
		return nil, 0, "" // Need more data
	}

	frame := &JT1078Frame{
		Header:      frameData[:16],
		SequenceNum: binary.BigEndian.Uint16(frameData[6:8]),
		Channel:     int(frameData[14]),
		DataType:    dataType,
		SubType:     subPackageType,
		DataLength:  payloadLength,
		Timestamp:   time.Now(),
	}

	// Extract JT1078 timestamp for grouping (CRITICAL for video reconstruction)
	if dataType != 4 && len(frameData) >= 24 {
		frame.JT1078Timestamp = binary.BigEndian.Uint64(frameData[16:24])
	}

	// Extract payload
	if payloadLength > 0 {
		frame.Payload = make([]byte, payloadLength)
		copy(frame.Payload, frameData[offset:offset+int(payloadLength)])
	}

	frameType := ""
	switch frame.DataType {
	case 0, 1, 2: // Video frames (I, P, B)
		frameType = "video"
	case 3: // Audio frame
		frameType = "audio"
		if debugSend {
			log.Printf("[AUDIO-DEBUG] Detected audio frame: DataType=%d, PayloadSize=%d, Channel=%d",
				frame.DataType, len(frame.Payload), frame.Channel)
		}
	}

	return frame, headerIdx + totalFrameSize, frameType
}

func processAudioFrame(frame *JT1078Frame) {
	if len(frame.Payload) == 0 {
		if debugSend {
			log.Printf("[AUDIO-DEBUG] Empty audio payload, skipping frame")
		}
		return
	}

	audioCounter++

	payloadSize := len(frame.Payload)
	if payloadSize > 320 {
		payloadSize = 320
	}

	if debugSend {
		log.Printf("[AUDIO-DEBUG] Processing audio frame #%d: PayloadSize=%d, Channel=%d, SeqNum=%d, DataType=%d",
			audioCounter, payloadSize, frame.Channel, frame.SequenceNum, frame.DataType)
	}

	// CRITICAL FIX: Send raw G.711A audio data (not converted PCM)
	// This matches the JT1078 specification and stream-capture implementation
	audioPayload := frame.Payload[:payloadSize]

	// Calculate duration based on G.711A sample rate (8000 Hz, 1 byte per sample)
	duration := float32(payloadSize) / 8000.0
	if duration < 0.02 {
		duration = 0.02
	} else if duration > 0.06 {
		duration = 0.04
	}

	audioFrame := &AudioFrame{
		PCMData:   audioPayload, // Now contains raw G.711A data, not PCM
		Duration:  duration,
		Timestamp: time.Now(),
	}

	frameBuffer.AddFrame(audioFrame)

	if debugSend {
		log.Printf("[AUDIO-DEBUG] Added audio frame to buffer: G711ASize=%d, Duration=%.3fs, BufferSize=%d",
			len(audioPayload), duration, len(frameBuffer.frames))
	}
}

// CRITICAL FIX: Implement proven video frame processing from stream-capture
func processVideoFrame(frame *JT1078Frame) {
	if len(frame.Payload) == 0 {
		return
	}

	// Extract SPS/PPS from I-frames
	if frame.DataType == 0 { // I-frame
		extractSPSPPS(frame.Payload, frame.Channel)
	}

	if debugReceive {
		fmt.Printf("📺 [VIDEO] Ch:%d, Type:%d, Sub:%d, Seq:%d, TS:%d, Size:%d\n",
			frame.Channel, frame.DataType, frame.SubType, frame.SequenceNum,
			frame.JT1078Timestamp, len(frame.Payload))
	}

	// For atomic frames (subType 0), send immediately
	if frame.SubType == 0 {
		videoFrame := &VideoFrame{
			Channel:     frame.Channel,
			SequenceNum: frame.SequenceNum,
			FrameType:   frame.DataType,
			Data:        frame.Payload,
			Timestamp:   time.Now(),
		}

		broadcastVideoFrame(videoFrame)
		return
	}

	// CRITICAL FIX: Use timestamp-based grouping (proven from stream-capture)
	videoFramesMu.Lock()
	defer videoFramesMu.Unlock()

	// Use JT1078 timestamp + channel as key (NOT sequence number)
	timestampKey := fmt.Sprintf("%d_%d", frame.Channel, frame.JT1078Timestamp)

	assembler, exists := videoFrames[timestampKey]
	if !exists {
		assembler = &VideoFrameAssembler{
			Channel:         frame.Channel,
			SequenceNum:     frame.SequenceNum,
			FrameType:       frame.DataType,
			Fragments:       []Fragment{},
			LastUpdate:      time.Now(),
			JT1078Timestamp: frame.JT1078Timestamp,
		}
		videoFrames[timestampKey] = assembler
	}

	// Add fragment with metadata
	assembler.Fragments = append(assembler.Fragments, Fragment{
		SubType:     frame.SubType,
		SequenceNum: frame.SequenceNum,
		Data:        frame.Payload,
	})
	assembler.ReceivedSize += len(frame.Payload)
	assembler.LastUpdate = time.Now()

	// Check if complete (has last fragment)
	hasLast := false
	hasFirst := false
	for _, frag := range assembler.Fragments {
		if frag.SubType == 1 {
			hasFirst = true
		}
		if frag.SubType == 2 {
			hasLast = true
		}
	}

	isComplete := hasLast && hasFirst

	if isComplete {
		// Reconstruct frame using proven chronological order
		videoFrame := &VideoFrame{
			Channel:     assembler.Channel,
			SequenceNum: assembler.SequenceNum,
			FrameType:   assembler.FrameType,
			Data:        reconstructVideoData(assembler),
			Timestamp:   time.Now(),
		}

		delete(videoFrames, timestampKey)
		go broadcastVideoFrame(videoFrame)

		if debugReceive {
			fmt.Printf("✅ Complete frame assembled: Ch:%d, Type:%d, Size:%d\n",
				videoFrame.Channel, videoFrame.FrameType, len(videoFrame.Data))
		}
	}
}

// CRITICAL FIX: Proven reconstruction algorithm from stream-capture
func reconstructVideoData(assembler *VideoFrameAssembler) []byte {
	if len(assembler.Fragments) == 1 {
		return assembler.Fragments[0].Data
	}

	// Sort fragments by SubType in chronological order: 1 (first) -> 3 (middle) -> 2 (last)
	sort.Slice(assembler.Fragments, func(i, j int) bool {
		// Custom sort: 1 < 3 < 2
		typeOrder := map[int]int{1: 0, 3: 1, 2: 2}
		orderI := typeOrder[assembler.Fragments[i].SubType]
		orderJ := typeOrder[assembler.Fragments[j].SubType]

		if orderI != orderJ {
			return orderI < orderJ
		}
		// If same type, sort by sequence number
		return assembler.Fragments[i].SequenceNum < assembler.Fragments[j].SequenceNum
	})

	// Assemble data
	var result []byte
	for _, frag := range assembler.Fragments {
		result = append(result, frag.Data...)
	}

	return result
}

func cleanupVideoFrames() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		videoFramesMu.Lock()
		now := time.Now()
		for key, assembler := range videoFrames {
			if now.Sub(assembler.LastUpdate) > 3*time.Second {
				delete(videoFrames, key)
				if debugReceive {
					fmt.Printf("🗑️ Cleaned up incomplete frame: Ch:%d, Fragments:%d\n",
						assembler.Channel, len(assembler.Fragments))
				}
			}
		}
		videoFramesMu.Unlock()
	}
}
