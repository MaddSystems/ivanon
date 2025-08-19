package jt808

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"proxy/shared"
)

// ParseJT808 decodes a raw JT808 message frame.
func ParseJT808(data []byte) (msgID uint16, phoneNumber string, msgSerial uint16, body []byte, totalPackets, currentPacket uint16, err error) {
	if len(data) < 2 || data[0] != 0x7e || data[len(data)-1] != 0x7e {
		return 0, "", 0, nil, 0, 0, fmt.Errorf("invalid message format or missing 0x7e markers")
	}

	unescaped := unescapeJT808Data(data[1 : len(data)-1])
	if len(unescaped) < 13 { // Minimum length: header(12) + checksum(1)
		return 0, "", 0, nil, 0, 0, fmt.Errorf("message too short after unescaping")
	}

	content := unescaped[:len(unescaped)-1]
	receivedChecksum := unescaped[len(unescaped)-1]
	if receivedChecksum != calculateChecksum(content) {
		shared.VPrint("Warning: checksum mismatch")
	}

	msgID = binary.BigEndian.Uint16(content[0:2])
	bodyAttr := binary.BigEndian.Uint16(content[2:4])
	phoneNumber = bcdToString(content[4:10])
	msgSerial = binary.BigEndian.Uint16(content[10:12])

	headerOffset := 12
	if (bodyAttr & 0x2000) != 0 { // Check for sub-packaging
		if len(content) < 16 {
			return 0, "", 0, nil, 0, 0, fmt.Errorf("sub-packaged message header too short")
		}
		totalPackets = binary.BigEndian.Uint16(content[12:14])
		currentPacket = binary.BigEndian.Uint16(content[14:16])
		headerOffset = 16
	} else {
		totalPackets = 1
		currentPacket = 1
	}

	body = content[headerOffset:]
	return msgID, phoneNumber, msgSerial, body, totalPackets, currentPacket, nil
}

// BuildJT808Message constructs a complete JT808 message frame.
func BuildJT808Message(msgID uint16, phoneNumber string, msgSerial uint16, body []byte, isSubPacket bool, totalPackets, currentPacket uint16) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, msgID)

	bodyAttr := uint16(len(body))
	if isSubPacket {
		bodyAttr |= 0x2000
	}
	binary.Write(&buf, binary.BigEndian, bodyAttr)

	phoneBCD, _ := hex.DecodeString(phoneNumber)
	buf.Write(phoneBCD)

	binary.Write(&buf, binary.BigEndian, msgSerial)

	if isSubPacket {
		binary.Write(&buf, binary.BigEndian, totalPackets)
		binary.Write(&buf, binary.BigEndian, currentPacket)
	}

	buf.Write(body)

	content := buf.Bytes()
	checksum := calculateChecksum(content)
	escapedContent := escapeJT808Data(append(content, checksum))

	var final bytes.Buffer
	final.WriteByte(0x7e)
	final.Write(escapedContent)
	final.WriteByte(0x7e)
	return final.Bytes()
}

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

func escapeJT808Data(data []byte) []byte {
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

func bcdToString(bcd []byte) string {
	return hex.EncodeToString(bcd)
}

func BuildGeneralResponse(phoneNumber string, replyMsgSerial uint16, replyMsgID uint16, result byte) []byte {
	var body bytes.Buffer
	binary.Write(&body, binary.BigEndian, replyMsgSerial)
	binary.Write(&body, binary.BigEndian, replyMsgID)
	body.WriteByte(result)
	return BuildJT808Message(0x8001, phoneNumber, shared.GenerateSerial(), body.Bytes(), false, 0, 0)
}

func BuildImageCaptureMessage(phoneNumber string, channel, count, resolution, quality int, brightness, contrast, saturation, chroma byte) []byte {
	var body bytes.Buffer
	body.WriteByte(byte(channel))
	binary.Write(&body, binary.BigEndian, uint16(count))
	binary.Write(&body, binary.BigEndian, uint16(0)) // Time
	body.WriteByte(0x00)                              // Save flag
	body.WriteByte(byte(resolution))
	body.WriteByte(byte(quality))
	body.WriteByte(brightness)
	body.WriteByte(contrast)
	body.WriteByte(saturation)
	body.WriteByte(chroma)
	return BuildJT808Message(0x8801, phoneNumber, shared.GenerateSerial(), body.Bytes(), false, 0, 0)
}
