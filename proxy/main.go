package main

// @title JT808 Proxy API
// @version 1.0
// @description API for managing JT808 devices and VoIP calls.
// @host ivan-proxy.armaddia.lat
// @BasePath /

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

var (
	activeConnections = make(map[string]net.Conn)
	imeiConnections   = make(map[string]ConnectionInfo) // maps IMEI to ConnectionInfo
	jt808Devices      = make(map[string]*JT808Device)   // maps phone number to device
	activeCalls       = make(map[string]*VoIPCall)      // maps call ID to call info
	connMutex         sync.Mutex
	voipServerURL     = os.Getenv("VOIP_SERVER_URL")
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

func proxyConnection(conn *net.TCPConn) {
	remoteAddr := conn.RemoteAddr().String()
	defer conn.Close()
	defer deregisterClient(remoteAddr, false) // Do not deregister device on connection close

	vPrint("New connection from: %s", remoteAddr)

	// Add to activeConnections
	connMutex.Lock()
	activeConnections[remoteAddr] = conn
	connMutex.Unlock()

	// Resolve and connect to remote server
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

	// Create channels to signal connection closure
	clientClosed := make(chan struct{})
	serverClosed := make(chan struct{})

	// Flag to track if we've processed the first JT808 message for this connection
	firstJT808 := true

	// Forward data from client to remote server
	go func() {
		defer close(clientClosed)
		buffer := make([]byte, 1024*1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					vPrint("Error reading from client: %v", err)
				}
				return
			}

			if n > 0 {
				// On first JT808 message, check if device needs deregistration
				if firstJT808 && buffer[0] == 0x7e {
					// Try to extract phone number from header
					_, phone, _, err := parseJT808Message(buffer[:n])
					if err == nil && phone != "" {
						connMutex.Lock()
						device, exists := jt808Devices[phone]
						if !exists {
							// New device, register and send deregistration
							device = &JT808Device{
								Conn:        conn,
								PhoneNumber: phone,
								LastSeen:    time.Now(),
								RemoteAddr:  remoteAddr,
							}
							jt808Devices[phone] = device
							connMutex.Unlock()
							// Send deregistration (logout) message to device
							sendDeregisterMessage(device)
							vPrint("[JT808] Forced deregistration (logout) sent for new device %s on connection %s", phone, remoteAddr)
							// Wait a short moment to allow device to process logout
							time.Sleep(200 * time.Millisecond)
						} else {
							// Device already registered, update connection details
							device.Conn = conn
							device.LastSeen = time.Now()
							device.RemoteAddr = remoteAddr
							connMutex.Unlock()
							vPrint("[JT808] Device %s already registered, updated connection %s", phone, remoteAddr)
						}
					}
					firstJT808 = false
				}

				// Forward data to the remote server
				_, err = rConn.Write(buffer[:n])
				if err != nil {
					vPrint("Error writing to remote server: %v", err)
					return
				}
				vPrint("From tracker to platform:\n%s", hex.Dump(buffer[:n]))
				// Check if this is a JT808 message
				if buffer[0] == 0x7e {
					handleJT808Message(conn, buffer[:n], remoteAddr)
				}
				// Prepare and publish MQTT message
				hexString := hex.EncodeToString(buffer[:n])
				trackerData := TrackerData{
					Payload:    hexString,
					RemoteAddr: remoteAddr,
				}
				byte_tracker_data_json, err := json.Marshal(trackerData)
				if err != nil {
					log.Printf("Error in byte_tracker_data_json creating JSON: %v", err)
					return
				}

				// Check if MQTT client is available before publishing
				if mqttClient != nil && mqttClient.IsConnected() {
					tracker_data_json := string(byte_tracker_data_json)
					if token := mqttClient.Publish("tracker/from-tcp", 0, false, tracker_data_json); token.Wait() && token.Error() != nil {
						vPrint("Error publishing to MQTT: %v", token.Error())
					}
				} else {
					vPrint("MQTT client not available or not connected")
				}
			}
		}
	}()

	// Forward data from remote server to client
	go func() {
		defer close(serverClosed)
		buffer := make([]byte, 1024*1024)
		for {
			n, err := rConn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					vPrint("Error reading from remote server: %v", err)
				}
				return
			}

			if n > 0 {
				_, err = conn.Write(buffer[:n])
				if err != nil {
					vPrint("Error writing to client: %v", err)
					return
				}
				vPrint("From platform to tracker:\n%s", hex.Dump(buffer[:min(32, n)]))
				// Track device state from platform-to-device messages
				if buffer[0] == 0x7e {
					trackDeviceFromPlatform(buffer[:n], conn, remoteAddr)
				}
			}
		}
	}()

	// Wait for either connection to close
	select {
	case <-clientClosed:
		vPrint("Client connection closed: %s", remoteAddr)
	case <-serverClosed:
		vPrint("Server connection closed for client: %s", remoteAddr)
	}
}

// Track device state from platform-to-device messages
func trackDeviceFromPlatform(data []byte, conn net.Conn, remoteAddr string) {
	msgID, phoneNumber, _, err := parseJT808Message(data)
	if err != nil {
		vPrint("[Platform->Device] Error parsing JT808 message: %v", err)
		return
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

// JT808 Message parsing and building functions
func parseJT808Message(data []byte) (msgID uint16, phoneNumber string, body []byte, err error) {
	if len(data) < 12 {
		return 0, "", nil, fmt.Errorf("message too short")
	}

	// Remove escape characters first
	unescaped := unescapeJT808Message(data)

	if len(unescaped) < 12 || unescaped[0] != 0x7e || unescaped[len(unescaped)-1] != 0x7e {
		return 0, "", nil, fmt.Errorf("invalid message format")
	}

	// Extract message content (without start/end markers and checksum)
	content := unescaped[1 : len(unescaped)-2]

	// Parse header
	msgID = binary.BigEndian.Uint16(content[0:2])
	bodyAttr := binary.BigEndian.Uint16(content[2:4])

	// Extract phone number (BCD format)
	phoneBCD := content[4:10]
	phoneNumber = bcdToString(phoneBCD)

	// Extract body
	bodyLength := bodyAttr & 0x3FF // Lower 10 bits
	if len(content) >= 12+int(bodyLength) {
		body = content[12 : 12+bodyLength]
	}

	return msgID, phoneNumber, body, nil
}

func unescapeJT808Message(data []byte) []byte {
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
		result += fmt.Sprintf("%02d", b)
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
	phoneBCD := stringToBCD(phoneNumber)
	buf.Write(phoneBCD)

	// Message serial number
	binary.Write(&buf, binary.BigEndian, msgSerial)

	// Message body
	buf.Write(body)

	// Calculate checksum
	content := buf.Bytes()
	checksum := calculateChecksum(content)

	// Build final message with markers
	var final bytes.Buffer
	final.WriteByte(0x7e)
	final.Write(escapeJT808Message(content))
	final.WriteByte(checksum)
	final.WriteByte(0x7e)

	return final.Bytes()
}

func stringToBCD(s string) []byte {
	// Pad to 12 digits if necessary
	for len(s) < 12 {
		s = "0" + s
	}

	bcd := make([]byte, 6)
	for i := 0; i < 6; i++ {
		high := s[i*2] - '0'
		low := s[i*2+1] - '0'
		bcd[i] = (high << 4) | low
	}
	return bcd
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

// JT808 VoIP specific functions
func buildVoIPStartMessage(phoneNumber string, audioServerIP string, audioPort int) []byte {
	var body bytes.Buffer

	// 1. Server IP (ASCII string, prefixed by length)
	ipStr := audioServerIP
	body.WriteByte(byte(len(ipStr))) // Length of IP string
	body.WriteString(ipStr)          // IP as ASCII

	// 2. Server video channel listening port (TCP, 2 bytes, big-endian)
	binary.Write(&body, binary.BigEndian, uint16(7800)) // TCP port: 7800 (0x1e78)

	// 3. Server video channel listening port (UDP, 2 bytes, big-endian)
	binary.Write(&body, binary.BigEndian, uint16(0)) // UDP port: 0 (0x0000)

	// 4. Logical channel (1 byte) - 0x24 as in the example
	body.WriteByte(0x24)

	// 5. Data type (1 byte) - 0x02 as in the example (two-way intercom)
	body.WriteByte(0x02)

	// 6. Stream type (1 byte) - 0x01 as in the example (sub stream)
	body.WriteByte(0x01)

	// 7. Talk type (1 byte) - 0x20 as in the example
	body.WriteByte(0x20)

	return buildJT808Message(0x9101, phoneNumber, generateSerial(), body.Bytes())
}

/*
	func buildVoIPStartMessage(phoneNumber string, audioServerIP string, audioPort int) []byte {
		var body bytes.Buffer

		// 1. Server IP (ASCII string, prefixed by length)
		ipStr := audioServerIP
		body.WriteByte(byte(len(ipStr))) // Length of IP string
		body.WriteString(ipStr)          // IP as ASCII

		// 2. Server TCP port (2 bytes, big-endian, set to 0 for UDP)
		binary.Write(&body, binary.BigEndian, uint16(0)) // TCP port: 0

		// 3. Server UDP port (2 bytes, big-endian)
		binary.Write(&body, binary.BigEndian, uint16(7800)) // UDP port: 7800

		// 4. Logical channel (1 byte) - 0x24 (36, two-way talk/monitor, Table 12-1)
		body.WriteByte(0x24)

		// 5. Data type (1 byte) - 0x03 (audio frame, Table 51)
		body.WriteByte(0x03)

		// 6. Stream type (1 byte) - 0x01 (sub stream, Table 56)
		body.WriteByte(0x01)

		// 7. Talk type (1 byte) - Remove or set to 0x00 (not defined in JT808, assume default)
		body.WriteByte(0x20) // Assume 0x00 if talk type is not required

		return buildJT808Message(0x9101, phoneNumber, generateSerial(), body.Bytes())
	}
*/
func buildVoIPControlMessage(phoneNumber string, command byte) []byte {
	var body bytes.Buffer

	// Logical channel (1 byte) - 0x24 as used in the VoIP start command
	body.WriteByte(0x24)

	// Control instruction (1 byte) - e.g., 4 for end call
	body.WriteByte(command)

	// Turn off AV types (1 byte) - 0: all (audio and video)
	body.WriteByte(0x00)

	// Switch stream type (1 byte) - 0: main stream
	body.WriteByte(0x00)

	return buildJT808Message(0x9102, phoneNumber, generateSerial(), body.Bytes())
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

func buildSetG711AParams(phoneNumber string) []byte {
	var body bytes.Buffer

	// Parameter count: 1
	body.WriteByte(0x01)

	// Parameter ID: 0x0074 (audio/video parameter settings, 4 bytes, big-endian)
	binary.Write(&body, binary.BigEndian, uint32(0x0074))

	// Parameter length: 11 bytes (audio param block)
	body.WriteByte(0x0B)

	// Audio Coding Format: 0x06 (G.711A, per Table 12)
	body.WriteByte(0x06)
	// Sample Rate: 0x00 (8 kHz)
	body.WriteByte(0x00)
	// Bit Depth: 0x00 (8-bit)
	body.WriteByte(0x00)
	// Channels: 0x00 (Mono)
	body.WriteByte(0x00)
	// Frame Length: 0x0014 (20 ms, 160 bytes for G.711A)
	binary.Write(&body, binary.BigEndian, uint16(0x0014))
	// Audio Output: 0x01 (Enable)
	body.WriteByte(0x01)
	// Bitrate: 0x0000FA00 (64 kbps, standard for G.711A)
	binary.Write(&body, binary.BigEndian, uint32(0x0000FA00))

	// Log the constructed message for debugging
	message := buildJT808Message(0x8103, phoneNumber, generateSerial(), body.Bytes())
	vPrint("Built G.711A parameter setting message for %s:\n%s", phoneNumber, hex.Dump(message))

	return message
}

func sendVoIPStartCommand(phoneNumber string, audioServerIP string, audioPort int) error {
	bmessage := buildSetG711AParams(phoneNumber) // Set G.711A parameters before starting VoIP
	fmt.Printf("\033[94m[COMMAND]\033[0m buildSetg711params: %s ", hex.Dump(bmessage))
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		return fmt.Errorf("device not found: %s", phoneNumber)
	}

	// Send the G.711A parameter setting message first
	_, err := device.Conn.Write(bmessage)
	if err != nil {
		return fmt.Errorf("failed to send G.711A param command: %v", err)
	}

	// Optionally, wait for a short time or for a response here

	message := buildVoIPStartMessage(phoneNumber, audioServerIP, audioPort)

	// Debug: print hex dump of the message being sent
	fmt.Printf("\033[31m[COMMAND]\033[0m VoIP Start Command to %s (IP: %s, Port: %d):\n%s\n",
		phoneNumber, audioServerIP, audioPort, hex.Dump(message))
	_, err = device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send VoIP start command: %v", err)
	}

	vPrint("VoIP start command sent to device: %s", phoneNumber)
	return nil
}

func sendVoIPControlCommand(phoneNumber string, command byte) error {
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		return fmt.Errorf("device not found: %s", phoneNumber)
	}

	message := buildVoIPControlMessage(phoneNumber, command)

	// Debug: print hex dump of the control command being sent
	fmt.Printf("[DEBUG] VoIP Control Command to %s (command: %d):\n%s\n", phoneNumber, command, hex.Dump(message))

	_, err := device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send VoIP control command: %v", err)
	}

	vPrint("VoIP control command (%d) sent to device: %s", command, phoneNumber)
	return nil
}

// Add these functions to main.go after the existing VoIP functions

// Build JT808 0x9101 video start command
func buildVideoStartMessage(phoneNumber string, videoServerIP string, videoPort int, channel int, streamType int) []byte {
	var body bytes.Buffer

	// 1. Server IP (ASCII string, prefixed by length)
	ipStr := videoServerIP
	body.WriteByte(byte(len(ipStr))) // Length of IP string
	body.WriteString(ipStr)          // IP as ASCII

	// 2. Server TCP port (2 bytes, big-endian)
	binary.Write(&body, binary.BigEndian, uint16(videoPort))

	// 3. Server UDP port (2 bytes, big-endian) - set to 0 for TCP
	binary.Write(&body, binary.BigEndian, uint16(0))

	// 4. Logical channel (1 byte) - channel number (1-4)
	body.WriteByte(byte(channel))

	// 5. Data type (1 byte) - 0x00 for video
	body.WriteByte(0x00)

	// 6. Stream type (1 byte) - 0=main stream, 1=sub stream
	body.WriteByte(byte(streamType))

	return buildJT808Message(0x9101, phoneNumber, generateSerial(), body.Bytes())
}

// Build JT808/JT1078 0x9102 video control command
func buildVideoControlMessage(phoneNumber string, channel int, command byte) []byte {
	var body bytes.Buffer

	// Logical channel (1 byte) - channel number (1-4)
	body.WriteByte(byte(channel))

	// Control instruction (1 byte) - 0=stop, 1=switch, 2=pause, 3=resume
	body.WriteByte(command)

	// Turn off AV types (1 byte) - 0: video only, 1: audio only, 2: both
	body.WriteByte(0x00) // video only

	// Switch stream type (1 byte) - 0: main stream, 1: sub stream
	body.WriteByte(0x01) // sub stream for live broadcast

	return buildJT808Message(0x9102, phoneNumber, generateSerial(), body.Bytes())
}

// Send video start command to device
func sendVideoStartCommand(phoneNumber string, videoServerIP string, videoPort int, channel int, streamType int) error {
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		return fmt.Errorf("device not found: %s", phoneNumber)
	}

	message := buildVideoStartMessage(phoneNumber, videoServerIP, videoPort, channel, streamType)

	// Debug: print hex dump of the message being sent
	fmt.Printf("\033[32m[VIDEO COMMAND]\033[0m Video Start Command to %s (IP: %s, Port: %d, Channel: %d, StreamType: %d):\n%s\n",
		phoneNumber, videoServerIP, videoPort, channel, streamType, hex.Dump(message))

	_, err := device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send video start command: %v", err)
	}

	vPrint("Video start command sent to device: %s, channel: %d", phoneNumber, channel)
	return nil
}

// Send video control command to device
func sendVideoControlCommand(phoneNumber string, channel int, command byte) error {
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		return fmt.Errorf("device not found: %s", phoneNumber)
	}

	message := buildVideoControlMessage(phoneNumber, channel, command)

	// Debug: print hex dump of the control command being sent
	fmt.Printf("\033[33m[VIDEO DEBUG]\033[0m Video Control Command to %s (channel: %d, command: %d):\n%s\n",
		phoneNumber, channel, command, hex.Dump(message))

	_, err := device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send video control command: %v", err)
	}

	vPrint("Video control command (%d) sent to device: %s, channel: %d", command, phoneNumber, channel)
	return nil
}

func handleJT808Message(conn net.Conn, data []byte, remoteAddr string) {
	msgID, phoneNumber, body, err := parseJT808Message(data)
	if err != nil {
		vPrint("Error parsing JT808 message: %v", err)
		return
	}

	// Print JT808 Message - ID, Phone, Body Length
	vPrint("JT808 Message - ID: 0x%04X, Phone: %s, Body Length: %d", msgID, phoneNumber, len(body))

	// Debug: Always print registration message info in purple, even if body is too short
	if msgID == 0x0100 {
		fmt.Printf("\033[35m[JT808 REG DEBUG] Registration msgID=0x0100: bodyLen=%d\033[0m\n", len(body))
		if len(body) > 0 {
			fmt.Printf("\033[35m[JT808 REG DEBUG] Registration body hex: %s\033[0m\n", hex.EncodeToString(body))
		}
		if len(body) >= 25 {
			manufID := body[4:9]    // 5 bytes
			termModel := body[9:17] // 8 bytes
			termID := body[17:24]   // 7 bytes
			fmt.Printf("\033[32m[JT808 REG] Manufacturer ID: %s | Terminal Model: %s | Terminal ID: %s\033[0m\n",
				string(manufID),
				string(bytes.Trim(termModel, " ")), // remove padding spaces
				string(termID),
			)
		}
	}

	// Handle different JT808 message types
	switch msgID {
	case 0x0100: // Registration
		if len(body) >= 25 {
			manufID := string(body[4:9])
			termModel := string(bytes.Trim(body[9:17], " "))
			termID := string(body[17:24])
			licensePlateColor := body[24]
			licensePlate := string(body[25:])

			vPrint("[JT808 REG] Registering device: %s | Manufacturer: %s | Model: %s | TerminalID: %s | PlateColor: %d | Plate: %s",
				phoneNumber, manufID, termModel, termID, licensePlateColor, licensePlate)
			handleRegistration(conn, phoneNumber, body, remoteAddr)
		}
	case 0x0102: // Authentication
		handleAuthentication(conn, phoneNumber, body, remoteAddr)
	case 0x0002: // Heartbeat
		handleHeartbeat(conn, phoneNumber, body, remoteAddr)
	case 0x0200: // Location report
		handleLocationReport(conn, phoneNumber, body, remoteAddr)
	case 0x0001: // Terminal general response
		handleTerminalResponse(conn, phoneNumber, body, remoteAddr)
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
	// Use fixed authentication code expected by the device
	authCode := "bsjgps"

	// Register device with auth code
	device := &JT808Device{
		Conn:          conn,
		PhoneNumber:   phoneNumber,
		LastSeen:      time.Now(),
		InCall:        false,
		Authenticated: false,
		RemoteAddr:    remoteAddr,
		AuthCode:      authCode,
	}

	connMutex.Lock()
	jt808Devices[phoneNumber] = device
	connMutex.Unlock()

	vPrint("Device registered: %s from %s with auth code: %s", phoneNumber, remoteAddr, authCode)

	// Send registration response (0x8100)
	var respBody bytes.Buffer
	binary.Write(&respBody, binary.BigEndian, generateSerial()) // Response serial number
	respBody.WriteByte(0)                                       // Result: 0 = success
	respBody.WriteString(authCode)                              // Authentication code

	response := buildJT808Message(0x8100, phoneNumber, generateSerial(), respBody.Bytes())
	conn.Write(response)

	vPrint("Registration response sent to device: %s with auth code: %s", phoneNumber, authCode)
}

func handleAuthentication(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	// Extract the authentication code from the message body
	authCode := string(body)

	// Verify authentication code
	device, exists := getJT808Device(phoneNumber)
	if !exists {
		vPrint("Authentication failed: Device not found: %s", phoneNumber)
		// Send general response with failure
		response := buildGeneralResponse(phoneNumber, generateSerial(), 0x0102, 1)
		conn.Write(response)
		return
	}

	if device.AuthCode != authCode {
		vPrint("Authentication failed: Invalid auth code for device: %s. Expected: %s, Got: %s", phoneNumber, device.AuthCode, authCode)
		// Send general response with failure
		response := buildGeneralResponse(phoneNumber, generateSerial(), 0x0102, 1)
		conn.Write(response)
		return
	}

	// Update device as authenticated
	device.Authenticated = true
	device.LastSeen = time.Now()

	// Send general response with success
	response := buildGeneralResponse(phoneNumber, generateSerial(), 0x0102, 0)
	conn.Write(response)

	vPrint("Authentication successful for device: %s", phoneNumber)
}

func handleHeartbeat(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	// Update last seen time
	if device, exists := getJT808Device(phoneNumber); exists {
		device.LastSeen = time.Now()
	}

	// Send general response
	response := buildGeneralResponse(phoneNumber, generateSerial(), 0x0002, 0)
	conn.Write(response)

	vPrint("Heartbeat response sent to device: %s", phoneNumber)
}

func handleLocationReport(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	// Update last seen time
	if device, exists := getJT808Device(phoneNumber); exists {
		device.LastSeen = time.Now()
	}

	// Send general response
	response := buildGeneralResponse(phoneNumber, generateSerial(), 0x0200, 0)
	conn.Write(response)

	vPrint("Location report response sent to device: %s", phoneNumber)
}

func handleTerminalResponse(conn net.Conn, phoneNumber string, body []byte, remoteAddr string) {
	if len(body) < 5 {
		return
	}

	replySerial := binary.BigEndian.Uint16(body[0:2])
	replyMsgID := binary.BigEndian.Uint16(body[2:4])
	result := body[4]

	// Print the response in hex (raw message)
	if *verbose {
		// Print in light blue using ANSI escape code \033[1;36m (cyan)
		fmt.Printf("\033[1;36mTerminal response - Serial: %d, MsgID: 0x%04X, Result: %d\033[0m\n", replySerial, replyMsgID, result)
		//fmt.Printf("\033[1;36mTerminal response raw hex: %s\033[0m\n", hex.EncodeToString(body))
	} else {
		log.Printf("Terminal response - Serial: %d, MsgID: 0x%04X, Result: %d", replySerial, replyMsgID, result)
	}

	// Handle VoIP responses
	if replyMsgID == 0x9101 { // Response to VoIP start
		if result == 0 {
			vPrint("VoIP call successfully started for device: %s", phoneNumber)
		} else {
			vPrint("VoIP call failed for device: %s, result: %d", phoneNumber, result)
		}
	}
	if replyMsgID == 0x8103 && result != 0 {
		vPrint("Failed to set G.711A parameters for %s: Result=%d", phoneNumber, result)
	}
}

// End a VoIP call for a device
func endVoIPCall(devicePhone string) error {
	device, exists := getJT808Device(devicePhone)
	if !exists {
		return fmt.Errorf("device not found: %s", devicePhone)
	}

	// JT808 VoIP control command: 4 = hangup/end call
	message := buildVoIPControlMessage(devicePhone, 4)

	// Debug: print hex dump of the hangup command
	fmt.Printf("[DEBUG] VoIP End Call Command to %s:\n%s\n", devicePhone, hex.Dump(message))

	_, err := device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send VoIP end call command: %v", err)
	}

	vPrint("VoIP end call command sent to device: %s", devicePhone)

	// Update state
	connMutex.Lock()
	device.InCall = false
	for _, call := range activeCalls {
		if call.DevicePhone == devicePhone {
			call.Status = "ended"
		}
	}
	connMutex.Unlock()

	return nil
}
