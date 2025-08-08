package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	clientsMu sync.RWMutex
	clients   = make(map[*websocket.Conn]*ClientInfo)

	// TCP connections to devices
	deviceConnsMu sync.RWMutex
	deviceConns   = make(map[string]net.Conn) // remoteAddr -> connection

	// Debug flags
	debugSend    bool
	debugReceive bool
	sendCounter  int64
	recvCounter  int64

	// API proxy configuration
	apiBaseURL = "https://ivan-proxy.armaddia.lat"
)

type ClientInfo struct {
	RemoteAddr   string
	ConnectedAt  time.Time
	FramesSent   int64
	LastActivity time.Time
	ClientType   string     // "receiver" or "transmitter"
	WriteMutex   sync.Mutex // Add mutex for WebSocket writes
}

type AudioFrame struct {
	PCMData   []byte
	Duration  float32
	Timestamp time.Time
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

// API Proxy Handlers - NEW endpoints to match frontend expectations
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

// EXISTING API Proxy Handlers (keep for backward compatibility)
func apiProxyDevices(w http.ResponseWriter, r *http.Request) {
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

func apiProxyCallStart(w http.ResponseWriter, r *http.Request) {
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

func apiProxyCallControl(w http.ResponseWriter, r *http.Request) {
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

// Debug function to decode BCD format
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

// Debug function to analyze received JT1078 frame
func debugReceivedFrame(data []byte, frameNum int64) {
	if !debugReceive {
		return
	}

	fmt.Printf("\n🔍 [DEBUG-RECEIVE] Frame #%d (Length: %d bytes)\n", frameNum, len(data))
	fmt.Printf("Raw hex: %s\n", hex.EncodeToString(data[:min(50, len(data))]))
	if len(data) > 50 {
		fmt.Printf("... (truncated, showing first 50 bytes)\n")
	}

	if len(data) < 28 {
		fmt.Printf("❌ Frame too short for JT1078 header (need 28 bytes, got %d)\n", len(data))
		return
	}

	fmt.Println("Field-by-field decoding:")

	// Header signature
	header := data[0:4]
	fmt.Printf("      • Bytes 0-3: \"%02x %02x %02x %02x\"", header[0], header[1], header[2], header[3])
	if header[0] == 0x30 && header[1] == 0x31 && header[2] == 0x63 && header[3] == 0x64 {
		fmt.Printf(" ✓ VALID protocol header\n")
	} else {
		fmt.Printf(" ❌ INVALID protocol header (expected: 30 31 63 64)\n")
	}

	// Reserved bytes
	fmt.Printf("      • Bytes 4-5: \"%02x %02x\" → Reserved\n", data[4], data[5])

	// Sequence number
	seqNum := binary.BigEndian.Uint16(data[6:8])
	fmt.Printf("      • Bytes 6-7: \"%02x %02x\" → seq_num = %d\n", data[6], data[7], seqNum)

	// SIM card number (BCD format)
	if len(data) >= 14 {
		simBytes := data[8:14]
		simDecoded := decodeBCD(simBytes)
		fmt.Printf("      • Bytes 8-13: \"%s\" → BCD decoded SIM: %s\n",
			hex.EncodeToString(simBytes), simDecoded)
	}

	// Logic channel
	if len(data) >= 15 {
		logicChannel := data[14]
		fmt.Printf("      • Byte 14: \"%02x\" → Logic Channel: %d\n", logicChannel, logicChannel)
	}

	// Data type and subtype
	if len(data) >= 16 {
		typeField := data[15]
		dataType := (typeField & 0xF0) >> 4
		subType := typeField & 0x0F
		fmt.Printf("      • Byte 15: \"%02x\" → Binary %08b → dataType=%d, subType=%d",
			typeField, typeField, dataType, subType)

		switch dataType {
		case 0:
			fmt.Printf(" (Video I-frame)")
		case 1:
			fmt.Printf(" (Video P-frame)")
		case 2:
			fmt.Printf(" (Video B-frame)")
		case 3:
			fmt.Printf(" (Audio frame)")
		case 4:
			fmt.Printf(" (Transparent data)")
		default:
			fmt.Printf(" (Unknown)")
		}
		fmt.Println()
	}

	// Timestamp
	if len(data) >= 24 {
		timestampBytes := data[16:24]
		fmt.Printf("      • Bytes 16-23: \"%s\" → Timestamp in BCD format\n",
			hex.EncodeToString(timestampBytes))
	}

	// Find payload length and data
	offset := 24
	if len(data) >= offset+2 {
		// For audio frames, payload length is typically at offset 24
		payloadLen := binary.BigEndian.Uint16(data[offset : offset+2])
		fmt.Printf("      • Bytes %d-%d: \"%02x %02x\" → Payload length: %d bytes\n",
			offset, offset+1, data[offset], data[offset+1], payloadLen)

		if len(data) >= offset+2+int(payloadLen) && payloadLen > 0 {
			payloadStart := offset + 2
			payloadData := data[payloadStart : payloadStart+int(payloadLen)]
			fmt.Printf("      • Payload: %s", hex.EncodeToString(payloadData[:min(20, len(payloadData))]))
			if len(payloadData) > 20 {
				fmt.Printf("... (%d total bytes)", len(payloadData))
			}
			fmt.Println()
		}
	}

	fmt.Printf("════════════════════════════════════════════════════════════════\n")
}

// Fixed debug function to analyze sent frame
func debugSentFrame(data []byte, frameNum int64) {
	if !debugSend {
		return
	}

	fmt.Printf("\n📤 [DEBUG-SEND] Frame #%d (Length: %d bytes)\n", frameNum, len(data))
	fmt.Printf("Raw hex: %s\n", hex.EncodeToString(data[:min(50, len(data))]))
	if len(data) > 50 {
		fmt.Printf("... (truncated, showing first 50 bytes)\n")
	}

	if len(data) < 28 {
		fmt.Printf("❌ Frame too short for JT1078 header (need 28 bytes, got %d)\n", len(data))
		return
	}

	fmt.Println("Field-by-field construction:")

	// Header signature
	header := data[0:4]
	fmt.Printf("      • Bytes 0-3: \"%02x %02x %02x %02x\"", header[0], header[1], header[2], header[3])
	if header[0] == 0x30 && header[1] == 0x31 && header[2] == 0x63 && header[3] == 0x64 {
		fmt.Printf(" ✓ VALID protocol header\n")
	} else {
		fmt.Printf(" ❌ INVALID protocol header (expected: 30 31 63 64)\n")
	}

	// Reserved bytes
	fmt.Printf("      • Bytes 4-5: \"%02x %02x\" → Reserved\n", data[4], data[5])

	// Sequence number
	seqNum := binary.BigEndian.Uint16(data[6:8])
	fmt.Printf("      • Bytes 6-7: \"%02x %02x\" → seq_num = %d\n", data[6], data[7], seqNum)

	// SIM card number (BCD format)
	if len(data) >= 14 {
		simBytes := data[8:14]
		simDecoded := decodeBCD(simBytes)
		fmt.Printf("      • Bytes 8-13: \"%s\" → BCD decoded SIM: %s\n",
			hex.EncodeToString(simBytes), simDecoded)
	}

	// Logic channel
	if len(data) >= 15 {
		logicChannel := data[14]
		fmt.Printf("      • Byte 14: \"%02x\" → Logic Channel: %d\n", logicChannel, logicChannel)
	}

	// Data type and subtype
	if len(data) >= 16 {
		typeField := data[15]
		dataType := (typeField & 0xF0) >> 4
		subType := typeField & 0x0F
		fmt.Printf("      • Byte 15: \"%02x\" → Binary %08b → dataType=%d, subType=%d",
			typeField, typeField, dataType, subType)

		switch dataType {
		case 3:
			fmt.Printf(" (Audio frame) ✓")
		default:
			fmt.Printf(" (Expected audio frame=3) ❌")
		}
		fmt.Println()
	}

	// BCD Timestamp (NEW STRUCTURE - bytes 16-23)
	if len(data) >= 24 {
		timestampBytes := data[16:24]
		fmt.Printf("      • Bytes 16-23: \"%s\" → BCD Timestamp\n",
			hex.EncodeToString(timestampBytes))
	}

	// Data length (NEW STRUCTURE - bytes 24-25)
	if len(data) >= 26 {
		dataLen := binary.BigEndian.Uint16(data[24:26])
		fmt.Printf("      • Bytes 24-25: \"%02x %02x\" → Data length: %d bytes\n",
			data[24], data[25], dataLen)

		// Audio payload (NEW STRUCTURE - starts at byte 26)
		if len(data) >= 26+int(dataLen) && dataLen > 0 {
			payloadStart := 26
			payloadData := data[payloadStart : payloadStart+int(dataLen)]
			fmt.Printf("      • Audio Data: %s", hex.EncodeToString(payloadData[:min(20, len(payloadData))]))
			if len(payloadData) > 20 {
				fmt.Printf("... (%d total bytes)", len(payloadData))
			}
			fmt.Println()
		}
	}

	fmt.Printf("════════════════════════════════════════════════════════════════\n")
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
		WriteMutex:   sync.Mutex{}, // Initialize mutex
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

// Fixed broadcastFrames with proper WebSocket synchronization
func broadcastFrames() {
	ticker := time.NewTicker(40 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		frames := frameBuffer.GetFrames()
		if len(frames) == 0 {
			continue
		}

		var combinedPCM []byte
		var totalDuration float32

		for _, frame := range frames {
			combinedPCM = append(combinedPCM, frame.PCMData...)
			totalDuration += frame.Duration
		}

		if len(combinedPCM) == 0 {
			continue
		}

		if totalDuration < 0.035 {
			totalDuration = 0.04
		}

		durBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(durBytes, math.Float32bits(totalDuration))
		message := append(durBytes, combinedPCM...)

		// Send only to receiver clients with proper synchronization
		clientsMu.RLock()
		var clientsToSend []*websocket.Conn
		var clientInfos []*ClientInfo

		// Collect clients to send to
		for client, info := range clients {
			if info.ClientType == "receiver" {
				clientsToSend = append(clientsToSend, client)
				clientInfos = append(clientInfos, info)
			}
		}
		clientsMu.RUnlock()

		// Send to each client with individual mutex protection
		var disconnectedClients []*websocket.Conn
		for i, client := range clientsToSend {
			info := clientInfos[i]

			// Use individual client mutex to prevent concurrent writes
			info.WriteMutex.Lock()

			client.SetWriteDeadline(time.Now().Add(1 * time.Second))
			err := client.WriteMessage(websocket.BinaryMessage, message)

			info.WriteMutex.Unlock()

			if err != nil {
				disconnectedClients = append(disconnectedClients, client)
			} else {
				// Update metrics (thread-safe since we're not modifying the map)
				info.FramesSent++
				info.LastActivity = time.Now()
			}
		}

		// Clean up disconnected clients
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

// WebSocket handler for audio reception (existing)
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
			// Use client mutex for ping messages too
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

// WebSocket handler for audio transmission (new)
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
			// Debug the frame being sent
			sendCounter++
			debugSentFrame(data, sendCounter)

			// Forward JT1078 frame to all connected devices
			forwardToDevices(data)
		}
	}
}

// Forward audio frame to all connected TCP devices
func forwardToDevices(frameData []byte) {
	deviceConnsMu.RLock()
	defer deviceConnsMu.RUnlock()

	sentCount := 0
	for remoteAddr, deviceConn := range deviceConns {
		deviceConn.SetWriteDeadline(time.Now().Add(2 * time.Second))

		if _, err := deviceConn.Write(frameData); err != nil {
			log.Printf("Failed to send to device %s: %v", remoteAddr, err)
			// Device will be cleaned up by connection handler
		} else {
			sentCount++
		}
	}

	if debugSend && sentCount > 0 {
		fmt.Printf("📡 Forwarded frame to %d device(s)\n", sentCount)
	}
}

func main() {
	// Parse command line flags
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

	// Audio WebSocket endpoints
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/transmit", transmitHandler)

	// NEW API Proxy endpoints - SHORT PATHS (what frontend expects)
	http.HandleFunc("/api/devices", apiProxyDevicesShort)
	http.HandleFunc("/api/call/start", apiProxyCallStartShort)
	http.HandleFunc("/api/call/control", apiProxyCallControlShort)

	// EXISTING API Proxy endpoints - FULL PATHS (keep for backward compatibility)
	http.HandleFunc("/api/v1/jt808/devices", apiProxyDevices)
	http.HandleFunc("/api/v1/jt808/call/start", apiProxyCallStart)
	http.HandleFunc("/api/v1/jt808/call/control", apiProxyCallControl)

	// Static file server
	http.Handle("/", http.FileServer(http.Dir(".")))

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		clientsMu.RLock()
		receiverCount := 0
		transmitterCount := 0
		for _, info := range clients {
			if info.ClientType == "receiver" {
				receiverCount++
			} else {
				transmitterCount++
			}
		}
		clientsMu.RUnlock()

		deviceConnsMu.RLock()
		deviceCount := len(deviceConns)
		deviceConnsMu.RUnlock()

		fmt.Fprintf(w, "OK - Receivers: %d, Transmitters: %d, Devices: %d",
			receiverCount, transmitterCount, deviceCount)
	})

	fmt.Println("🚀 JT1078 Two-Way Audio Streamer with API Proxy")
	fmt.Println("📄 Web interface: http://localhost:8081")
	fmt.Println("🔊 TCP audio port: 7800")
	fmt.Println("📻 Reception: ws://localhost:8081/ws")
	fmt.Println("🎤 Transmission: ws://localhost:8081/transmit")
	fmt.Println("🔗 API Proxy endpoints:")
	fmt.Println("   • GET  /api/devices")
	fmt.Println("   • POST /api/call/start")
	fmt.Println("   • POST /api/call/control")
	fmt.Println("   • GET  /api/v1/jt808/devices (legacy)")
	fmt.Println("   • POST /api/v1/jt808/call/start (legacy)")
	fmt.Println("   • POST /api/v1/jt808/call/control (legacy)")

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

		// Process frames (incoming audio from device)
		for {
			frame, consumed, isAudio := extractFrame(buffer)
			if consumed == 0 {
				break
			}

			// Debug the received frame
			if frame != nil {
				recvCounter++
				fullFrame := buffer[:consumed]
				debugReceivedFrame(fullFrame, recvCounter)
			}

			buffer = buffer[consumed:]

			if frame != nil && isAudio {
				processAudioFrame(frame)
			}

			if len(buffer) > 32768 {
				buffer = buffer[len(buffer)-1024:]
			}
		}
	}

	log.Printf("TCP device disconnected: %s", remoteAddr)
}

type JT1078Frame struct {
	Header     []byte
	DataType   int
	DataLength uint16
	Payload    []byte
}

func extractFrame(buffer []byte) (*JT1078Frame, int, bool) {
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
			return nil, len(buffer) - 4, false
		}
		return nil, 0, false
	}

	if len(buffer)-headerIdx < 28 {
		return nil, 0, false
	}

	frameData := buffer[headerIdx:]
	frame := &JT1078Frame{
		Header: frameData[:28],
	}

	label3 := frameData[15]
	frame.DataType = int(label3&0xF0) >> 4

	offset := 16

	if frame.DataType != 4 {
		if len(frameData) < offset+8 {
			return nil, 0, false
		}
		offset += 8
	}

	if frame.DataType != 4 && frame.DataType != 3 {
		if len(frameData) < offset+4 {
			return nil, 0, false
		}
		offset += 4
	}

	if len(frameData) < offset+2 {
		return nil, 0, false
	}
	frame.DataLength = binary.BigEndian.Uint16(frameData[offset : offset+2])
	offset += 2

	if frame.DataLength > 8192 {
		return nil, headerIdx + 28, false
	}

	totalFrameSize := offset + int(frame.DataLength)
	if len(frameData) < totalFrameSize {
		return nil, 0, false
	}

	if frame.DataLength > 0 {
		frame.Payload = make([]byte, frame.DataLength)
		copy(frame.Payload, frameData[offset:offset+int(frame.DataLength)])
	}

	isAudio := frame.DataType == 3 && len(frame.Payload) >= 100
	return frame, headerIdx + totalFrameSize, isAudio
}

func processAudioFrame(frame *JT1078Frame) {
	if len(frame.Payload) == 0 {
		return
	}

	payloadSize := len(frame.Payload)
	if payloadSize > 320 {
		payloadSize = 320
	}

	audioPayload := frame.Payload[:payloadSize]
	pcmData := DecodeG711ALaw(audioPayload)

	if len(pcmData) == 0 {
		return
	}

	duration := float32(payloadSize) / 8000.0
	if duration < 0.02 {
		duration = 0.02
	} else if duration > 0.06 {
		duration = 0.04
	}

	audioFrame := &AudioFrame{
		PCMData:   pcmData,
		Duration:  duration,
		Timestamp: time.Now(),
	}

	frameBuffer.AddFrame(audioFrame)
}
