package services

import (
	"encoding/binary"
	"fmt"
	"net"
	"proxy/models"
	"proxy/shared"
	"time"
)

// GetJT808Device retrieves a device by its phone number.
func GetJT808Device(phoneNumber string) (*models.JT808Device, bool) {
	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()
	device, exists := shared.JT808Devices[phoneNumber]
	return device, exists
}

// UpdateDeviceState creates or updates a device's record upon receiving any message.
func UpdateDeviceState(conn net.Conn, phoneNumber string, remoteAddr string) {
	if phoneNumber == "" {
		return
	}
	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()
	if device, exists := shared.JT808Devices[phoneNumber]; exists {
		device.Conn = conn
		device.LastSeen = time.Now()
		device.RemoteAddr = remoteAddr
	} else {
		shared.JT808Devices[phoneNumber] = &models.JT808Device{
			Conn:        conn,
			PhoneNumber: phoneNumber,
			LastSeen:    time.Now(),
			RemoteAddr:  remoteAddr,
		}
	}
}

// TrackDeviceFromPlatform updates device state based on messages from the platform.
func TrackDeviceFromPlatform(data []byte) {
	// A simple way to extract phone number without full parsing for performance
	if len(data) < 12 {
		return
	}
	// TODO: Properly unescape and parse for accuracy
	phoneNumber := bcdToString(data[5:11]) // Approximate position
	msgID := binary.BigEndian.Uint16(data[1:3])
	body := data[13 : len(data)-2]

	if msgID == 0x8001 && len(body) >= 5 {
		responseMsgID := binary.BigEndian.Uint16(body[2:4])
		result := body[4]
		if (responseMsgID == 0x0102 || responseMsgID == 0x0100) && result == 0 {
			shared.ConnMutex.Lock()
			if device, exists := shared.JT808Devices[phoneNumber]; exists {
				device.Authenticated = true
				shared.VPrint("[Platform->Device] Device %s authenticated successfully.", phoneNumber)
			}
			shared.ConnMutex.Unlock()
		}
	}
}

func bcdToString(bcd []byte) string {
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x", bcd[0], bcd[1], bcd[2], bcd[3], bcd[4], bcd[5])
}

// DeregisterClient cleans up all records associated with a closed connection.
func DeregisterClient(remoteAddr string) {
	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()

	// Close and remove from active connections
	if conn, exists := shared.ActiveConnections[remoteAddr]; exists {
		conn.Close()
		delete(shared.ActiveConnections, remoteAddr)
	}

	// Remove any IMEI mappings for this address
	for imei, connInfo := range shared.ImeiConnections {
		if connInfo.RemoteAddr == remoteAddr {
			delete(shared.ImeiConnections, imei)
		}
	}

	// Remove any JT808 device mappings for this address
	for phone, device := range shared.JT808Devices {
		if device.RemoteAddr == remoteAddr {
			delete(shared.JT808Devices, phone)
			shared.VPrint("Deregistered JT808 device %s due to connection close from %s", phone, remoteAddr)
		}
	}
}
