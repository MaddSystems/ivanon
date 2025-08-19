// services/jt808_handler.go
package services

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"proxy/jt808"
	"proxy/models"
	"proxy/shared"
	"time"
)

// HandleJT808Message is the main router for incoming messages from devices.
func HandleJT808Message(conn net.Conn, data []byte, remoteAddr string) {
	msgID, phone, _, body, total, current, err := jt808.ParseJT808(data)
	if err != nil {
		shared.VPrint("Error parsing JT808 message: %v", err)
		return
	}

	shared.VPrint("JT808 Message - ID: 0x%04X, Phone: %s", msgID, phone)
	UpdateDeviceState(conn, phone, remoteAddr)

	switch msgID {
	case 0x0102: // Authentication
		handleAuthentication(phone, body)
	case 0x0001: // Terminal general response
		handleTerminalResponse(phone, body)
	case 0x0801: // Multimedia data upload
		handleMultimediaUpload(conn, phone, body, total, current)
	case 0x0805: // Camera command response
		handleCameraResponse(body)
	}
}

func handleAuthentication(phone string, body []byte) {
	device, exists := GetJT808Device(phone)
	if !exists {
		return
	}
	device.AuthCode = string(body)
	shared.VPrint("Authentication attempt tracked for device: %s", phone)
}

func handleTerminalResponse(phone string, body []byte) {
	if len(body) < 5 {
		return
	}
	replySerial := binary.BigEndian.Uint16(body[0:2])
	replyMsgID := binary.BigEndian.Uint16(body[2:4])
	result := body[4]
	fmt.Printf("\033[1;36mTerminal response - Phone: %s, Serial: %d, MsgID: 0x%04X, Result: %d\033[0m\n", phone, replySerial, replyMsgID, result)
}

func handleCameraResponse(body []byte) {
	if len(body) < 5 {
		return
	}
	replySerial := binary.BigEndian.Uint16(body[0:2])
	result := body[4]
	log.Printf("[CAMERA RESPONSE] Serial: %d, Result: %d", replySerial, result)
	if result != 0 {
		log.Printf("[CAMERA ERROR] Device rejected snapshot command - Error code: %d", result)
	}
}

func handleMultimediaUpload(conn net.Conn, phone string, body []byte, total, current uint16) {
	shared.VPrint("Multimedia upload - Phone: %s, Packet: %d/%d", phone, current, total)
	if current == 1 {
		handleFirstMultimediaPacket(conn, phone, body, total)
	} else {
		handleSubsequentMultimediaPacket(conn, phone, body, total, current)
	}
}

func handleFirstMultimediaPacket(conn net.Conn, phone string, body []byte, total uint16) {
	if len(body) < 36 {
		shared.VPrint("First multimedia packet too short: %d bytes", len(body))
		return
	}
	multimediaID := binary.BigEndian.Uint32(body[0:4])
	channelID := body[7]

	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()

	snapshot := &models.ImageSnapshot{
		MultimediaID:   multimediaID,
		DevicePhone:    phone,
		Channel:        int(channelID),
		ImageData:      make([]byte, 0),
		CaptureTime:    time.Now(),
		ExpectedChunks: int(total),
		LastChunkTime:  time.Now(),
	}
	shared.ActiveSnapshots[multimediaID] = snapshot
	shared.ImageChunks[multimediaID] = make(map[uint16]*models.ImageChunk)

	imageData := body[36:]
	if len(imageData) > 0 {
		shared.ImageChunks[multimediaID][1] = &models.ImageChunk{Data: imageData, Timestamp: time.Now()}
		snapshot.ReceivedChunks++
		snapshot.TotalSize += len(imageData)
	}

	if pending, exists := shared.EarlyChunks[phone]; exists {
		for _, chunk := range pending {
			if int(chunk.TotalPackets) == snapshot.ExpectedChunks {
				if _, alreadyExists := shared.ImageChunks[multimediaID][chunk.CurrentPacket]; !alreadyExists {
					shared.ImageChunks[multimediaID][chunk.CurrentPacket] = &models.ImageChunk{Data: chunk.Body, Timestamp: time.Now()}
					snapshot.ReceivedChunks++
					snapshot.TotalSize += len(chunk.Body)
				}
			}
		}
		delete(shared.EarlyChunks, phone)
	}

	if snapshot.ReceivedChunks >= snapshot.ExpectedChunks {
		AssembleAndCompleteSnapshot(snapshot)
	}
	SendMultimediaResponse(conn, phone, multimediaID, 0)
}

func handleSubsequentMultimediaPacket(conn net.Conn, phone string, body []byte, total, current uint16) {
	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()

	var activeSnapshot *models.ImageSnapshot
	for _, snap := range shared.ActiveSnapshots {
		if snap.DevicePhone == phone && !snap.Complete {
			activeSnapshot = snap
			break
		}
	}

	if activeSnapshot == nil {
		chunk := &models.PendingChunk{Body: body, TotalPackets: total, CurrentPacket: current}
		shared.EarlyChunks[phone] = append(shared.EarlyChunks[phone], chunk)
		return
	}

	multimediaID := activeSnapshot.MultimediaID
	if _, exists := shared.ImageChunks[multimediaID][current]; exists {
		shared.VPrint("Duplicate packet %d for multimedia ID %d", current, multimediaID)
		SendMultimediaResponse(conn, phone, multimediaID, 0)
		return
	}

	shared.ImageChunks[multimediaID][current] = &models.ImageChunk{Data: body, Timestamp: time.Now()}
	activeSnapshot.ReceivedChunks++
	activeSnapshot.TotalSize += len(body)
	activeSnapshot.LastChunkTime = time.Now()

	if activeSnapshot.ReceivedChunks >= activeSnapshot.ExpectedChunks {
		AssembleAndCompleteSnapshot(activeSnapshot)
	}
	SendMultimediaResponse(conn, phone, multimediaID, 0)
}
