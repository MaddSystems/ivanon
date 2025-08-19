package services

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"proxy/jt808"
	"proxy/models"
	"proxy/shared"
	"time"
)

// SendImageCaptureCommand sends a snapshot command to a device.
func SendImageCaptureCommand(phone string, channel, count, res, qual int, bright, cont, sat, chroma byte) error {
	device, exists := GetJT808Device(phone)
	if !exists {
		return fmt.Errorf("device not found: %s", phone)
	}
	if device.Conn == nil {
		return fmt.Errorf("device connection is nil: %s", phone)
	}

	message := jt808.BuildImageCaptureMessage(phone, channel, count, res, qual, bright, cont, sat, chroma)
	_, err := device.Conn.Write(message)
	if err != nil {
		return fmt.Errorf("failed to send image capture command: %v", err)
	}
	log.Printf("[IMAGE COMMAND] Successfully sent snapshot command to device: %s", phone)
	return nil
}

// SnapshotCleanupRoutine periodically cleans up old, incomplete snapshots.
func SnapshotCleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		shared.ConnMutex.Lock()
		now := time.Now()
		for id, snapshot := range shared.ActiveSnapshots {
			if !snapshot.Complete && now.Sub(snapshot.LastChunkTime) > 2*time.Minute {
				delete(shared.ActiveSnapshots, id)
				delete(shared.ImageChunks, id)
				shared.VPrint("Cleaned up expired snapshot: %d for device %s", id, snapshot.DevicePhone)
			}
		}
		shared.ConnMutex.Unlock()
	}
}

// CleanupIncompleteSnapshots removes any pending snapshot data for a specific device and channel.
func CleanupIncompleteSnapshots(devicePhone string, channel int) {
	shared.ConnMutex.Lock()
	defer shared.ConnMutex.Unlock()
	for id, snapshot := range shared.ActiveSnapshots {
		if snapshot.DevicePhone == devicePhone && snapshot.Channel == channel && !snapshot.Complete {
			delete(shared.ActiveSnapshots, id)
			delete(shared.ImageChunks, id)
			log.Printf("[IMAGE SNAPSHOT] Cleaned up incomplete snapshot ID: %d", id)
		}
	}
}

// AssembleAndCompleteSnapshot assembles image chunks and marks the snapshot as complete.
func AssembleAndCompleteSnapshot(snapshot *models.ImageSnapshot) {
	var completeImageData []byte
	multimediaID := snapshot.MultimediaID
	for i := uint16(1); i <= uint16(snapshot.ExpectedChunks); i++ {
		if chunk, ok := shared.ImageChunks[multimediaID][i]; ok {
			completeImageData = append(completeImageData, chunk.Data...)
		} else {
			shared.VPrint("ERROR: Missing chunk %d when assembling multimedia ID %d", i, multimediaID)
			return // Cannot complete assembly
		}
	}
	snapshot.ImageData = completeImageData
	snapshot.Complete = true
	snapshot.TotalSize = len(completeImageData)
	log.Printf("Image capture COMPLETE - ID: %d, Device: %s, FinalSize: %d bytes", multimediaID, snapshot.DevicePhone, len(completeImageData))
}

// SendMultimediaResponse sends an acknowledgment for a received multimedia packet.
func SendMultimediaResponse(conn net.Conn, phone string, multimediaID uint32, result byte) {
	var body bytes.Buffer
	binary.Write(&body, binary.BigEndian, multimediaID)
	body.WriteByte(result) // 0 for success
	message := jt808.BuildJT808Message(0x8800, phone, shared.GenerateSerial(), body.Bytes(), false, 0, 0)

	_, err := conn.Write(message)
	if err != nil {
		shared.VPrint("Failed to send multimedia response: %v", err)
	}
}
