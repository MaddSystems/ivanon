package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	_ "proxy/docs"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	localAddress  = flag.String("l", "0.0.0.0:1024", "Local address")
	remoteAddress = flag.String("r", os.Getenv("PLATFORM_HOST"), "Remote address")
	verbose       = flag.Bool("v", false, "Enable verbose logging")
	mqttClient    mqtt.Client
)

// Send a simulated JT808 deregistration (logout) message to the platform
func sendDeregisterMessage(device *JT808Device) {
	// JT808 standard logout message is 0x0003, no body
	msg := buildJT808Message(0x0003, device.PhoneNumber, generateSerial(), []byte{})
	if device.Conn != nil {
		device.Conn.Write(msg)
		vPrint("Sent deregistration (logout) message for device: %s", device.PhoneNumber)
	}
}

// Helper function to print verbose logs if enabled
func vPrint(format string, v ...interface{}) {
	if *verbose {
		log.Printf(format, v...)
	}
}

type TrackerData struct {
	Payload    string `json:"payload"`
	RemoteAddr string `json:"remoteaddr"`
}

type TrackerAssign struct {
	Imei       string `json:"imei"`
	Protocol   string `json:"protocol"`
	RemoteAddr string `json:"remoteaddr"`
}

type ConnectionInfo struct {
	RemoteAddr string
	Protocol   string
}

// JT808 VoIP structures
type JT808Device struct {
	Conn          net.Conn  `json:"-"`
	PhoneNumber   string    `json:"phone_number"`
	LastSeen      time.Time `json:"last_seen"`
	InCall        bool      `json:"in_call"`
	Authenticated bool      `json:"authenticated"`
	RemoteAddr    string    `json:"remote_addr"`
	AuthCode      string    `json:"auth_code"`
}

type VoIPCall struct {
	CallID      string
	DevicePhone string
	CallerID    string
	Status      string
	StartTime   time.Time
	AudioServer string
	AudioPort   int
}

type VoIPServerClient struct {
	serverURL string
}

// Enhanced Image capture related structs
type ImageSnapshot struct {
	MultimediaID   uint32    `json:"multimedia_id"`
	DevicePhone    string    `json:"device_phone"`
	Channel        int       `json:"channel"`
	ImageData      []byte    `json:"-"`
	CaptureTime    time.Time `json:"capture_time"`
	Complete       bool      `json:"complete"`
	ExpectedChunks int       `json:"expected_chunks"`
	ReceivedChunks int       `json:"received_chunks"`
	ChunkSequences []int     `json:"chunk_sequences"`
	LastChunkTime  time.Time `json:"last_chunk_time"`
	TotalSize      int       `json:"total_size"`
}

// Struct to hold chunks that arrive before the first packet
type pendingChunk struct {
	body          []byte
	totalPackets  uint16
	currentPacket uint16
}

// Enhanced chunk tracking
type ImageChunk struct {
	SequenceNum uint16
	Data        []byte
	Timestamp   time.Time
}

var (
	activeConnections = make(map[string]net.Conn)
	imeiConnections   = make(map[string]ConnectionInfo)         // maps IMEI to ConnectionInfo
	jt808Devices      = make(map[string]*JT808Device)           // maps phone number to device
	activeCalls       = make(map[string]*VoIPCall)              // maps call ID to call info
	activeSnapshots   = make(map[uint32]*ImageSnapshot)         // maps multimedia ID to snapshot info
	imageChunks       = make(map[uint32]map[uint16]*ImageChunk) // maps multimedia ID to chunks
	earlyChunks       = make(map[string][]*pendingChunk)        // maps phone number to chunks that arrived before packet 1
	connMutex         sync.Mutex
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SendDataToConnection(remoteAddr string, data []byte) error {
	connMutex.Lock()
	conn, exists := activeConnections[remoteAddr]
	connMutex.Unlock()
	if !exists {
		return fmt.Errorf("no active connection for address: %s", remoteAddr)
	}
	_, err := conn.Write(data)
	if err != nil {
		vPrint("Error writing to connection %s: %v", remoteAddr, err)
		return err
	}
	return nil
}

func init() {
	// MQTT connection options
	opts := mqtt.NewClientOptions()
	mqttBrokerHost := os.Getenv("MQTT_BROKER_HOST")
	if mqttBrokerHost == "" {
		mqttBrokerHost = "localhost" // default value
	}
	opts.AddBroker(fmt.Sprintf("tcp://%s:1883", mqttBrokerHost))
	opts.SetClientID("proxy_service")
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		// Subscribe to all topics only after connection is established
		if token := client.Subscribe("tracker/assign", 0, handleAssignment); token.Wait() && token.Error() != nil {
			log.Printf("Failed to subscribe to tracker/assign: %v", token.Error())
		}
		if token := client.Subscribe("tracker/send", 0, func(client mqtt.Client, msg mqtt.Message) {
			var data TrackerData
			if err := json.Unmarshal(msg.Payload(), &data); err != nil {
				vPrint("Error unmarshaling MQTT message: %v", err)
				return
			}
			rawBytes, err := hex.DecodeString(data.Payload)
			if err != nil {
				vPrint("Error decoding hex payload: %v", err)
				return
			}
			if err := SendDataToConnection(data.RemoteAddr, rawBytes); err != nil {
				vPrint("Error sending data to connection: %v", err)
				return
			}
		}); token.Wait() && token.Error() != nil {
			log.Printf("Failed to subscribe to tracker/send: %v", token.Error())
		}
		if token := client.Subscribe("tracker/assign-imei2remoteaddr", 0, func(client mqtt.Client, msg mqtt.Message) {
			var data TrackerAssign
			if err := json.Unmarshal(msg.Payload(), &data); err != nil {
				vPrint("Error unmarshaling JSON: %v", err)
				return
			}
			imei := data.Imei
			protocol := data.Protocol
			vPrint("Assigning imei: %s to address: %s from protocol %s", imei, data.RemoteAddr, protocol)
			connMutex.Lock()
			imeiConnections[imei] = ConnectionInfo{
				RemoteAddr: data.RemoteAddr,
				Protocol:   protocol,
			}
			connMutex.Unlock()
		}); token.Wait() && token.Error() != nil {
			log.Printf("Failed to subscribe to tracker/assign-imei2remoteaddr: %v", token.Error())
		}
	})

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Failed to connect to MQTT broker: %v", token.Error())
	}
}

func handleAssignment(client mqtt.Client, msg mqtt.Message) {
	var assign TrackerAssign
	if err := json.Unmarshal(msg.Payload(), &assign); err != nil {
		log.Printf("Error unmarshaling assignment message: %v", err)
		return
	}

	if assign.Imei == "" || assign.RemoteAddr == "" {
		log.Printf("Invalid assignment message: missing IMEI or RemoteAddr")
		return
	}

	connMutex.Lock()
	defer connMutex.Unlock()

	// Verify the connection exists
	if _, exists := activeConnections[assign.RemoteAddr]; !exists {
		log.Printf("Cannot assign IMEI %s: no active connection for %s", assign.Imei, assign.RemoteAddr)
		return
	}

	// Update the IMEI mapping
	imeiConnections[assign.Imei] = ConnectionInfo{
		RemoteAddr: assign.RemoteAddr,
		Protocol:   assign.Protocol,
	}
	vPrint("Assigned IMEI %s to connection %s with protocol %s", assign.Imei, assign.RemoteAddr, assign.Protocol)
}

func main() {
	flag.Parse()
	fmt.Printf("Listening: %v\nProxying %v\n", *localAddress, *remoteAddress)
	// Start the HTTP server
	go func() {
		r := setupRouter() // Assumes setupRouter is defined in api.go
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
		if err := r.Run(":8080"); err != nil {
			vPrint("Error starting HTTP server: %v", err)
		}
	}()
	// Start cleanup routine for old snapshots
	go snapshotCleanupRoutine()
	addr, err := net.ResolveTCPAddr("tcp", *localAddress)
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Deregister all currently tracked devices on startup
	connMutex.Lock()
	for phone, device := range jt808Devices {
		go sendDeregisterMessage(device)
		delete(jt808Devices, phone)
	}
	connMutex.Unlock()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go proxyConnection(conn)
	}
}

// Cleanup routine for old snapshots and chunks
func snapshotCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		connMutex.Lock()
		now := time.Now()

		// Clean up old incomplete snapshots (older than 2 minutes)
		for id, snapshot := range activeSnapshots {
			if now.Sub(snapshot.LastChunkTime) > 2*time.Minute {
				delete(activeSnapshots, id)
				delete(imageChunks, id)
				vPrint("Cleaned up expired snapshot: %d for device %s", id, snapshot.DevicePhone)
			}
		}
		connMutex.Unlock()
	}
}

// Deregister a client connection
func deregisterClient(remoteAddr string, deregisterDevice bool) {
	connMutex.Lock()
	defer connMutex.Unlock()

	// First, delete from activeConnections
	if conn, exists := activeConnections[remoteAddr]; exists {
		conn.Close()
		delete(activeConnections, remoteAddr)
	}

	// Find and remove any IMEI that maps to this remoteAddr
	var imeisToDelete []string
	for imei, connInfo := range imeiConnections {
		if connInfo.RemoteAddr == remoteAddr {
			imeisToDelete = append(imeisToDelete, imei)
		}
	}

	// Delete the found IMEIs
	for _, imei := range imeisToDelete {
		delete(imeiConnections, imei)
		vPrint("Deregistered IMEI %s due to connection close from %s", imei, remoteAddr)
	}

	// Find and remove any JT808 device that maps to this remoteAddr, if requested
	if deregisterDevice {
		var phonesToDelete []string
		for phone, device := range jt808Devices {
			if device.RemoteAddr == remoteAddr {
				phonesToDelete = append(phonesToDelete, phone)
			}
		}

		// Delete the found JT808 devices
		for _, phone := range phonesToDelete {
			delete(jt808Devices, phone)
			vPrint("Deregistered JT808 device %s due to connection close from %s", phone, remoteAddr)
		}
	}

	// End any active calls for this connection
	var callsToEnd []string
	for callID, call := range activeCalls {
		if device, exists := jt808Devices[call.DevicePhone]; exists && device.RemoteAddr == remoteAddr {
			callsToEnd = append(callsToEnd, callID)
		}
	}

	// End the found calls
	for _, callID := range callsToEnd {
		if call, exists := activeCalls[callID]; exists {
			call.Status = "disconnected"
			vPrint("Ended call %s due to device disconnection", callID)
		}
	}
}

// Track device state from platform-to-device messages
func trackDeviceFromPlatform(data []byte, conn net.Conn, remoteAddr string) {
	msgID, phoneNumber, _, body, _, _, err := parseJT808(data)
	if err != nil {
		vPrint("[Platform->Device] Error parsing JT808 message: %v", err)
		return
	}

	// Handle platform responses to mark devices as authenticated
	if msgID == 0x8001 && len(body) >= 5 { // Platform General Response
		// Parse the response: serial(2) + msgID(2) + result(1)
		responseSerial := binary.BigEndian.Uint16(body[0:2])
		responseMsgID := binary.BigEndian.Uint16(body[2:4])
		result := body[4]

		vPrint("[Platform->Device] Response 0x8001 - Serial: %d, ResponseTo: 0x%04X, Result: %d", responseSerial, responseMsgID, result)

		if responseMsgID == 0x0102 && result == 0 { // Successful authentication response
			connMutex.Lock()
			if device, exists := jt808Devices[phoneNumber]; exists {
				device.Authenticated = true
				vPrint("[Platform->Device] Device %s authenticated successfully (serial: %d)", phoneNumber, responseSerial)
			}
			connMutex.Unlock()
		} else if responseMsgID == 0x0100 && result == 0 { // Successful registration response
			connMutex.Lock()
			if _, exists := jt808Devices[phoneNumber]; exists {
				// Registration successful - device can now authenticate
				vPrint("[Platform->Device] Device %s registered successfully (serial: %d)", phoneNumber, responseSerial)
			}
			connMutex.Unlock()
		}
	}

	if phoneNumber != "" {
		connMutex.Lock()
		device, exists := jt808Devices[phoneNumber]
		if !exists {
			device = &JT808Device{
				Conn:        conn,
				PhoneNumber: phoneNumber,
				LastSeen:    time.Now(),
				RemoteAddr:  remoteAddr,
			}
			jt808Devices[phoneNumber] = device
		} else {
			device.Conn = conn
			device.LastSeen = time.Now()
			device.RemoteAddr = remoteAddr
		}
		connMutex.Unlock()
		vPrint("[Platform->Device] Tracked device %s (msgID: 0x%04X) on connection %s", phoneNumber, msgID, remoteAddr)
	}
}

// forwarder handles the TCP stream for one direction, ensuring proper message framing.
func forwarder(src net.Conn, dest net.Conn, processFunc func(data []byte, conn net.Conn, remoteAddr string), conn net.Conn, remoteAddr string) {
	buffer := make([]byte, 0, 4096)
	readBuf := make([]byte, 2048)

	for {
		n, err := src.Read(readBuf)
		if err != nil {
			if err != io.EOF {
				vPrint("Error reading from %s: %v", src.RemoteAddr(), err)
			}
			break
		}

		if n > 0 {
			// Immediately forward the raw data to the destination
			if _, err := dest.Write(readBuf[:n]); err != nil {
				vPrint("Error writing to %s: %v", dest.RemoteAddr(), err)
				break
			}

			// Buffer the data for our own processing
			buffer = append(buffer, readBuf[:n]...)

			// Process all complete 0x7e-delimited messages in the buffer
			for {
				startIdx := bytes.IndexByte(buffer, 0x7e)
				if startIdx == -1 {
					// No start marker found, clear buffer as it's likely garbage
					buffer = buffer[:0]
					break
				}

				// If there's garbage before the start marker, discard it
				if startIdx > 0 {
					buffer = buffer[startIdx:]
				}

				// Look for the end marker after the start marker
				endIdx := bytes.IndexByte(buffer[1:], 0x7e)
				if endIdx == -1 {
					// No end marker yet, wait for more data
					break
				}
				endIdx += 1 // Adjust index to be relative to the buffer

				// We have a complete message
				msg := buffer[:endIdx+1]
				msgCopy := make([]byte, len(msg))
				copy(msgCopy, msg)

				// Call the provided function to process the message
				go processFunc(msgCopy, conn, remoteAddr)

				// Remove the processed message from the buffer
				buffer = buffer[endIdx+1:]
			}
		}
	}
}

func processClientData(data []byte, conn net.Conn, remoteAddr string) {
	// Prepare and publish MQTT message
	hexString := hex.EncodeToString(data)
	trackerData := TrackerData{
		Payload:    hexString,
		RemoteAddr: remoteAddr,
	}
	byte_tracker_data_json, err := json.Marshal(trackerData)
	if err != nil {
		log.Printf("Error in byte_tracker_data_json creating JSON: %v", err)
		return
	}

	if mqttClient != nil && mqttClient.IsConnected() {
		tracker_data_json := string(byte_tracker_data_json)
		if token := mqttClient.Publish("tracker/from-tcp", 0, false, tracker_data_json); token.Wait() && token.Error() != nil {
			vPrint("Error publishing to MQTT: %v", token.Error())
		}
	} else {
		vPrint("MQTT client not available or not connected")
	}

	vPrint("From tracker to platform:\n%s", hex.Dump(data))
	handleJT808Message(conn, data, remoteAddr)
}

func processPlatformData(data []byte, conn net.Conn, remoteAddr string) {
	vPrint("From platform to tracker:\n%s", hex.Dump(data[:min(32, len(data))]))
	trackDeviceFromPlatform(data, conn, remoteAddr)
}

func proxyConnection(conn *net.TCPConn) {
	remoteAddr := conn.RemoteAddr().String()
	defer conn.Close()
	defer deregisterClient(remoteAddr, false)

	vPrint("New connection from: %s", remoteAddr)

	connMutex.Lock()
	activeConnections[remoteAddr] = conn
	connMutex.Unlock()

	rAddr, err := net.ResolveTCPAddr("tcp", *remoteAddress)
	if err != nil {
		vPrint("Failed to resolve remote address: %v", err)
		return
	}

	rConn, err := net.DialTCP("tcp", nil, rAddr)
	if err != nil {
		vPrint("Failed to connect to remote server: %v", err)
		return
	}
	defer rConn.Close()

	done := make(chan struct{})

	go func() {
		forwarder(conn, rConn, processClientData, conn, remoteAddr)
		close(done) // Signal that one direction is closed
	}()

	forwarder(rConn, conn, processPlatformData, conn, remoteAddr)

	<-done // Wait for the client-to-server forwarder to finish
	vPrint("Connection closed for: %s", remoteAddr)
}

// JT808 Message parsing and building functions
func parseJT808(data []byte) (msgID uint16, phoneNumber string, msgSerial uint16, body []byte, totalPackets, currentPacket uint16, err error) {
	if len(data) < 2 || data[0] != 0x7e || data[len(data)-1] != 0x7e {
		return 0, "", 0, nil, 0, 0, fmt.Errorf("invalid message format or missing 0x7e markers")
	}

	unescaped := unescapeJT808Data(data[1 : len(data)-1])

	if len(unescaped) < 1 {
		return 0, "", 0, nil, 0, 0, fmt.Errorf("message too short after unescaping")
	}
	content := unescaped[:len(unescaped)-1]
	receivedChecksum := unescaped[len(unescaped)-1]
	calculatedChecksum := calculateChecksum(content)
	if receivedChecksum != calculatedChecksum {
		vPrint("Warning: checksum mismatch. Got: %02X, Calculated: %02X", receivedChecksum, calculatedChecksum)
	}

	if len(content) < 12 {
		return 0, "", 0, nil, 0, 0, fmt.Errorf("header too short: %d bytes", len(content))
	}

	msgID = binary.BigEndian.Uint16(content[0:2])
	bodyAttr := binary.BigEndian.Uint16(content[2:4])
	phoneBCD := content[4:10]
	phoneNumber = bcdToString(phoneBCD)
	msgSerial = binary.BigEndian.Uint16(content[10:12])

	headerOffset := 12
	isSubPackaged := (bodyAttr & 0x2000) != 0

	if isSubPackaged {
		if len(content) < 16 {
			return 0, "", 0, nil, 0, 0, fmt.Errorf("sub-packaged message header too short: %d bytes", len(content))
		}
		totalPackets = binary.BigEndian.Uint16(content[12:14])
		currentPacket = binary.BigEndian.Uint16(content[14:16])
		headerOffset = 16
	} else {
		totalPackets = 1
		currentPacket = 1
	}

	if len(content) < headerOffset {
		return 0, "", 0, nil, 0, 0, fmt.Errorf("content too short for header: have %d, need %d", len(content), headerOffset)
	}
	body = content[headerOffset:]

	return msgID, phoneNumber, msgSerial, body, totalPackets, currentPacket, nil
}

// Helper function to unescape JT808 data (without the 0x7e markers)
func unescapeJT808Data(data []byte) []byte {
	var result []byte
	for i := 0; i < len(data); i++ {
		if data[i] == 0x7d && i+1 < len(data) {
			if data[i+1] == 0x01 {
				result = append(result, 0x7d)
				i++
			} else if data[i+1] == 0x02 {
				result = append(result, 0x7e)
				i++
			} else {
				result = append(result, data[i])
			}
		} else {
			result = append(result, data[i])
		}
	}
	return result
}

func bcdToString(bcd []byte) string {
	var result string
	for _, b := range bcd {
		result += fmt.Sprintf("%02x", b) // Corrected BCD to string conversion
	}
	return result
}

func buildJT808Message(msgID uint16, phoneNumber string, msgSerial uint16, body []byte) []byte {
	var buf bytes.Buffer

	// Message ID
	binary.Write(&buf, binary.BigEndian, msgID)

	// Message body attributes (length + flags)
	bodyAttr := uint16(len(body))
	binary.Write(&buf, binary.BigEndian, bodyAttr)

	// Phone number (BCD format)
	phoneBCD, _ := hex.DecodeString(phoneNumber)
	buf.Write(phoneBCD)

	// Message serial number
	binary.Write(&buf, binary.BigEndian, msgSerial)

	// Message body
	buf.Write(body)

	// Calculate checksum
	content := buf.Bytes()
	checksum := calculateChecksum(content)

	// Escape content and add checksum
	escapedContent := escapeJT808Message(append(content, checksum))

	// Build final message with markers
	var final bytes.Buffer
	final.WriteByte(0x7e)
	final.Write(escapedContent)
	final.WriteByte(0x7e)

	return final.Bytes()
}

func escapeJT808Message(data []byte) []byte {
	var result []byte
	for _, b := range data {
		if b == 0x7e {
			result = append(result, 0x7d, 0x02)
		} else if b == 0x7d {
			result = append(result, 0x7d, 0x01)
		} else {
			result = append(result, b)
		}
	}
	return result
}

func calculateChecksum(data []byte) byte {
	var checksum byte
	for _, b := range data {
		checksum ^= b
	}
	return checksum
}

func buildGeneralResponse(phoneNumber string, replyMsgSerial uint16, replyMsgID uint16, result byte) []byte {
	var body bytes.Buffer

	// Reply message serial number
	binary.Write(&body, binary.BigEndian, replyMsgSerial)

	// Reply message ID
	binary.Write(&body, binary.BigEndian, replyMsgID)

	// Result (0 = success, 1 = failure, 2 = message error, 3 = not supported)
	body.WriteByte(result)

	return buildJT808Message(0x8001, phoneNumber, generateSerial(), body.Bytes())
}

func generateSerial() uint16 {
	b := make([]byte, 2)
	rand.Read(b)
	return binary.BigEndian.Uint16(b)
}

func generateCallID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Register JT808 device
func getJT808Device(phoneNumber string) (*JT808Device, bool) {
	connMutex.Lock()
	defer connMutex.Unlock()

	device, exists := jt808Devices[phoneNumber]
	return device, exists
}

// Build 0x8801 Camera Immediate Shooting Command according to JT808 Table 40
func buildImageCaptureMessage(phoneNumber string, channel, count, resolution, quality int, brightness, contrast, saturation, chroma byte) []byte {
	var body bytes.Buffer

	body.WriteByte(byte(channel))
	binary.Write(&body, binary.BigEndian, uint16(count))
	binary.Write(&body, binary.BigEndian, uint16(0))
	body.WriteByte(0x00)
	body.WriteByte(byte(resolution))
	body.WriteByte(byte(quality))
	body.WriteByte(brightness)
	body.WriteByte(contrast)
	body.WriteByte(saturation)
	body.WriteByte(chroma)

	return buildJT808Message(0x8801, phoneNumber, generateSerial(), body.Bytes())
}

// Send image capture command to device with enhanced validation and debugging
func sendImageCaptureCommand(phoneNumber string, channel, count, resolution, quality int, brightness, contrast, saturation, chroma byte) error {
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		return fmt.Errorf("device not found: %s", phoneNumber)
	}

	if device.Conn == nil {
		return fmt.Errorf("device connection is nil: %s", phoneNumber)
	}

	if !device.Authenticated {
		log.Printf("[IMAGE COMMAND WARNING] Device %s may not be authenticated yet", phoneNumber)
	}
	message := buildImageCaptureMessage(phoneNumber, channel, count, resolution, quality, brightness, contrast, saturation, chroma)

	fmt.Printf("\033[95m[IMAGE COMMAND]\033[0m Sending to %s:\n", phoneNumber)
	fmt.Printf("  Channel: %d, Count: %d, Resolution: %d, Quality: %d\n", channel, count, resolution, quality)
	fmt.Printf("  Message hex:\n%s\n", hex.Dump(message))

	_, err := device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send image capture command: %v", err)
	}

	log.Printf("[IMAGE COMMAND] Successfully sent snapshot command to device: %s, channel: %d", phoneNumber, channel)
	return nil
}

// Enhanced handleJT808Message with better debugging for snapshot issues
func handleJT808Message(conn net.Conn, data []byte, remoteAddr string) {
	msgID, phoneNumber, msgSerial, body, totalPackets, currentPacket, err := parseJT808(data)
	if err != nil {
		vPrint("Error parsing JT808 message: %v", err)
		return
	}

	vPrint("JT808 Message - ID: 0x%04X, Phone: %s, Serial: %d, Body Length: %d", msgID, phoneNumber, msgSerial, len(body))

	// Handle different JT808 message types
	switch msgID {
	case 0x0100: // Registration
		handleRegistration(conn, phoneNumber, body, remoteAddr)
	case 0x0102: // Authentication
		handleAuthentication(conn, phoneNumber, body, remoteAddr)
	case 0x0002: // Heartbeat
		handleHeartbeat(conn, phoneNumber, remoteAddr)
	case 0x0200: // Location report
		handleLocationReport(conn, phoneNumber, remoteAddr)
	case 0x0704: // Location Data Batch Upload
		handleLocationBatch(conn, phoneNumber, body, remoteAddr)
	case 0x0001: // Terminal general response
		handleTerminalResponse(phoneNumber, body)
	case 0x0801: // Multimedia data upload
		handleMultimediaUpload(conn, phoneNumber, body, totalPackets, currentPacket)
	case 0x0805: // Camera command response
		if len(body) >= 5 {
			replySerial := binary.BigEndian.Uint16(body[0:2])
			result := body[4]
			log.Printf("[CAMERA RESPONSE] Serial: %d, Result: %d", replySerial, result)
			if result != 0 {
				log.Printf("[CAMERA ERROR] Device rejected snapshot command - Error code: %d", result)
			}
		}
	default:
		// Update device state for all other messages
		if phoneNumber != "" {
			connMutex.Lock()
			device, exists := jt808Devices[phoneNumber]
			if !exists {
				device = &JT808Device{
					Conn:        conn,
					PhoneNumber: phoneNumber,
					LastSeen:    time.Now(),
					RemoteAddr:  remoteAddr,
				}
				jt808Devices[phoneNumber] = device
			} else {
				device.Conn = conn
				device.LastSeen = time.Now()
				device.RemoteAddr = remoteAddr
			}
			connMutex.Unlock()
		}
	}
}

func handleRegistration(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	device := &JT808Device{
		Conn:        conn,
		PhoneNumber: phoneNumber,
		LastSeen:    time.Now(),
		RemoteAddr:  remoteAddr,
	}

	connMutex.Lock()
	jt808Devices[phoneNumber] = device
	connMutex.Unlock()

	vPrint("Device registration tracked: %s from %s", phoneNumber, remoteAddr)
}

func handleAuthentication(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	authCode := string(body)
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		vPrint("Authentication attempt from unknown device: %s", phoneNumber)
		return
	}
	device.AuthCode = authCode
	device.LastSeen = time.Now()
	vPrint("Authentication attempt tracked for device: %s", phoneNumber)
}

func handleHeartbeat(conn net.Conn, phoneNumber string, remoteAddr string) {
	if device, exists := getJT808Device(phoneNumber); exists {
		device.LastSeen = time.Now()
	}
	vPrint("Heartbeat received from device: %s", phoneNumber)
}

func handleLocationReport(conn net.Conn, phoneNumber string, remoteAddr string) {
	if device, exists := getJT808Device(phoneNumber); exists {
		device.LastSeen = time.Now()
	}
	vPrint("Location report received from device: %s", phoneNumber)
}

func handleLocationBatch(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	if device, exists := getJT808Device(phoneNumber); exists {
		device.LastSeen = time.Now()
	}
	vPrint("Location batch received from device: %s, size: %d bytes", phoneNumber, len(body))
}

func handleTerminalResponse(phoneNumber string, body []byte) {
	if len(body) < 5 {
		return
	}
	replySerial := binary.BigEndian.Uint16(body[0:2])
	replyMsgID := binary.BigEndian.Uint16(body[2:4])
	result := body[4]
	fmt.Printf("\033[1;36mTerminal response - Phone: %s, Serial: %d, MsgID: 0x%04X, Result: %d\033[0m\n", phoneNumber, replySerial, replyMsgID, result)
}

func handleMultimediaUpload(conn net.Conn, phoneNumber string, body []byte, totalPackets, currentPacket uint16) {
	vPrint("Multimedia upload - Phone: %s, Packet: %d/%d, BodySize: %d", phoneNumber, currentPacket, totalPackets, len(body))

	if currentPacket == 1 {
		if len(body) < 36 {
			vPrint("First multimedia packet too short for metadata: %d bytes", len(body))
			sendMultimediaResponse(conn, phoneNumber, 0, 1) // Error
			return
		}

		multimediaID := binary.BigEndian.Uint32(body[0:4])
		multimediaType := body[4]
		format := body[5]
		channelID := body[7]
		imageData := body[36:]

		vPrint("First multimedia packet - ID: %d, Channel: %d, TotalPackets: %d", multimediaID, channelID, totalPackets)

		if multimediaType != 0 || format != 0 {
			vPrint("Ignoring multimedia upload - not a JPEG image (Type: %d, Format: %d)", multimediaType, format)
			sendMultimediaResponse(conn, phoneNumber, multimediaID, 0)
			return
		}

		connMutex.Lock()
		if _, exists := activeSnapshots[multimediaID]; exists {
			delete(imageChunks, multimediaID)
		}
		snapshot := &ImageSnapshot{
			MultimediaID:   multimediaID,
			DevicePhone:    phoneNumber,
			Channel:        int(channelID),
			ImageData:      make([]byte, 0),
			CaptureTime:    time.Now(),
			ExpectedChunks: int(totalPackets),
			LastChunkTime:  time.Now(),
		}
		activeSnapshots[multimediaID] = snapshot
		imageChunks[multimediaID] = make(map[uint16]*ImageChunk)

		if len(imageData) > 0 {
			imageChunks[multimediaID][currentPacket] = &ImageChunk{Data: imageData, Timestamp: time.Now()}
			snapshot.ReceivedChunks++
			snapshot.TotalSize += len(imageData) // Correctly account for first chunk size
		}

		// Process any chunks that arrived before this one
		if pending, exists := earlyChunks[phoneNumber]; exists {
			vPrint("Processing %d buffered early chunks for device %s", len(pending), phoneNumber)
			remainingPending := make([]*pendingChunk, 0)
			for _, chunk := range pending {
				if int(chunk.totalPackets) == snapshot.ExpectedChunks {
					if _, alreadyExists := imageChunks[multimediaID][chunk.currentPacket]; !alreadyExists {
						imageChunks[multimediaID][chunk.currentPacket] = &ImageChunk{Data: chunk.body, Timestamp: time.Now()}
						snapshot.ReceivedChunks++
						snapshot.TotalSize += len(chunk.body)
						// Send delayed response for the buffered chunk
						go sendMultimediaResponse(conn, phoneNumber, multimediaID, 0)
					}
				} else {
					remainingPending = append(remainingPending, chunk)
				}
			}
			if len(remainingPending) > 0 {
				earlyChunks[phoneNumber] = remainingPending
			} else {
				delete(earlyChunks, phoneNumber)
			}
		}

		if snapshot.ReceivedChunks >= snapshot.ExpectedChunks {
			snapshot.Complete = true // Mark complete if all chunks (including early ones) are now present
		}
		connMutex.Unlock()
		sendMultimediaResponse(conn, phoneNumber, multimediaID, 0)
		return
	}

	// --- Handle subsequent packets (currentPacket > 1) ---
	connMutex.Lock()
	var snapshot *ImageSnapshot
	var mostRecentTime time.Time
	var foundID uint32

	// Find the active snapshot for this device
	for id, snap := range activeSnapshots {
		if snap.DevicePhone == phoneNumber && !snap.Complete && snap.LastChunkTime.After(mostRecentTime) {
			snapshot = snap
			foundID = id
			mostRecentTime = snap.LastChunkTime
		}
	}

	if snapshot == nil {
		// Snapshot not created yet, buffer this chunk
		vPrint("Early packet %d/%d received for device %s, buffering.", currentPacket, totalPackets, phoneNumber)
		chunk := &pendingChunk{body: body, totalPackets: totalPackets, currentPacket: currentPacket}
		earlyChunks[phoneNumber] = append(earlyChunks[phoneNumber], chunk)
		connMutex.Unlock()
		return
	}

	// Snapshot exists, process the chunk
	multimediaID := foundID
	if _, exists := imageChunks[multimediaID][currentPacket]; exists {
		vPrint("Duplicate packet %d received for multimedia ID %d, ignoring", currentPacket, multimediaID)
		connMutex.Unlock()
		sendMultimediaResponse(conn, phoneNumber, multimediaID, 0)
		return
	}

	imageChunks[multimediaID][currentPacket] = &ImageChunk{Data: body, Timestamp: time.Now()}
	snapshot.ReceivedChunks++
	snapshot.TotalSize += len(body)
	snapshot.LastChunkTime = time.Now()
	vPrint("Stored chunk %d/%d for ID: %d. Total received: %d", currentPacket, snapshot.ExpectedChunks, multimediaID, snapshot.ReceivedChunks)

	if snapshot.ReceivedChunks >= snapshot.ExpectedChunks {
		vPrint("All %d chunks received, assembling image - ID: %d", snapshot.ExpectedChunks, multimediaID)
		var completeImageData []byte
		for i := uint16(1); i <= uint16(snapshot.ExpectedChunks); i++ {
			if chunk, ok := imageChunks[multimediaID][i]; ok {
				completeImageData = append(completeImageData, chunk.Data...)
			} else {
				vPrint("ERROR: Missing chunk %d when trying to assemble multimedia ID %d", i, multimediaID)
				connMutex.Unlock()
				sendMultimediaResponse(conn, phoneNumber, multimediaID, 0) // Acknowledge the current packet
				return
			}
		}

		snapshot.ImageData = completeImageData
		snapshot.Complete = true
		snapshot.TotalSize = len(completeImageData)
		vPrint("Image capture COMPLETE - ID: %d, Device: %s, FinalSize: %d bytes", multimediaID, phoneNumber, len(completeImageData))
	}
	connMutex.Unlock()
	sendMultimediaResponse(conn, phoneNumber, multimediaID, 0)
}

func sendMultimediaResponse(conn net.Conn, phoneNumber string, multimediaID uint32, result byte) {
	var body bytes.Buffer
	binary.Write(&body, binary.BigEndian, multimediaID)
	// For successful reception of packets, we list the packet IDs received.
	// Since we process one by one, we'll just acknowledge success.
	if result == 0 {
		body.WriteByte(0)
	} else {
		// A real implementation might request re-transmission of certain packets.
		// Here, we just signal failure.
		body.WriteByte(result)
	}

	message := buildJT808Message(0x8800, phoneNumber, generateSerial(), body.Bytes())

	if result == 0 {
		vPrint("Sending multimedia upload SUCCESS response - ID: %d", multimediaID)
	} else {
		vPrint("Sending multimedia upload ERROR response - ID: %d, Error: %d", multimediaID, result)
	}

	_, err := conn.Write(message)
	if err != nil {
		vPrint("Failed to send multimedia response: %v", err)
	}
}
