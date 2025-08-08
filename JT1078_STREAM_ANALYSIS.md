# JT1078 Video Stream Analysis and Decoding Guide

## Overview
This document describes how to receive, parse, and decode JT1078 video streams based on the official JT/T 808-2011 and JT/T 1078 specifications, and analysis of real-world captured data from a GPS tracking device.

**Based on**: Official JT/T 808-2011 and JT/T 1078 Protocol Standards  
**Date of Analysis**: August 1, 2025  
**Captured Data**: 2.3 MB over 30 seconds  
**Device SIM**: 018853512607  
**Channel**: 3 (likely front camera)

## Protocol Separation
It's crucial to understand that there are **two separate protocols** in JT1078 communication:

1. **JT808 Signaling Protocol**: Uses 0x7e identification bits for command/control messages
2. **JT1078 Video Stream Protocol**: Uses RTP-based format with 0x30 0x31 0x63 0x64 headers for video data

---

## 1. JT808 Signaling Protocol (Command/Control)

### 1.1 Message Structure (from Manual_JT808.md)
According to the official JT808 specification, each signaling message consists of:

```
[0x7e] [Message Header] [Message Body] [Check Code] [0x7e]
```

### 1.2 Message Header Format
| Start Byte | Field | Data Type | Description |
|------------|-------|-----------|-------------|
| 0 | Message ID | WORD | Command identifier |
| 2 | Message body attribute | WORD | Includes subcontract bit (bit 13) |
| 4 | Terminal phone number | BCD[6] | 6-byte BCD encoded |
| 10 | Message serial number | WORD | Circular accumulation from 0 |
| 12 | Message Packet Encapsulation | - | Present if subcontract bit is set |

### 1.3 Escape Sequences
The JT808 protocol uses escape sequences:
- `0x7e` → `0x7d 0x02`
- `0x7d` → `0x7d 0x01`

---

## 2. JT1078 Video Stream Protocol (RTP-based)

### 2.1 Official RTP Protocol Payload Format
According to Manual_jt1078.md Table 19, the JT1078 stream uses RTP format with this structure:

```
Bytes 0-3:   Fixed Header (0x30 0x31 0x63 0x64)
Bytes 4-5:   V/P/X/CC and PT fields
Bytes 6-7:   Sequence number
Bytes 8-13:  SIM card number (6 bytes BCD)
Byte 14:     Logic channel number
Byte 15:     Data type (0=I-frame, 1=P-frame, 2=B-frame, etc.)
Bytes 16-23: Timestamp (8 bytes)
Bytes 24-25: Last I-frame interval
Bytes 26-27: Last frame interval  
Byte 28:     Subpackage (fragmentation) indicator
Byte 29:     Subpackage number
Bytes 30+:   H.264 payload data
```

### 2.2 Official Subpackage Definitions
According to the JT1078 manual, the subpackage field (byte 28) should be:
- `0000 (0)`: Atomic packet (complete frame in single packet)
- `0001 (1)`: First subpackage of fragmented frame  
- `0010 (2)`: Last subpackage of fragmented frame
- `0011 (3)`: Middle subpackage of fragmented frame

---

## 3. Stream Data Acquisition

### 1.1 Capture Method
A TCP server was created on port 7800 to capture raw JT1078 stream data:

```go
// Basic TCP server structure
listener, _ := net.Listen("tcp", ":7800")
conn, _ := listener.Accept()

// Read raw bytes and store to file
buf := make([]byte, 65536)
for {
    n, _ := conn.Read(buf)
    rawFile.Write(buf[:n])
}
```

## 3. Stream Data Acquisition

### 3.1 Capture Method
A TCP server was created on port 7800 to capture raw JT1078 stream data:

```go
// Basic TCP server structure
listener, _ := net.Listen("tcp", ":7800")
conn, _ := listener.Accept()

// Read raw bytes and store to file
buf := make([]byte, 65536)
for {
    n, _ := conn.Read(buf)
    rawFile.Write(buf[:n])
}
```

### 3.2 Captured Data Summary
- **Total bytes**: 2,296,060 bytes
- **Duration**: 30.04 seconds  
- **Frames parsed**: 33 complete JT1078 packets
- **Frame types**: I-frames (DataType=0) and P-frames (DataType=1)
- **Average packet size**: ~950 bytes per fragment

---

## 4. Device-Specific Protocol Deviations

### 4.1 Critical Finding: Non-Standard Subpackage Encoding
**IMPORTANT**: The analyzed device uses **non-standard SubPackage encoding** that differs from the official JT1078 specification:

| Official JT1078 Standard | Device Implementation | Description |
|--------------------------|----------------------|-------------|
| `0000 (0)` = Atomic | `0000 (0)` = Complete | ✓ Matches standard |
| `0001 (1)` = First | `0001 (1)` = First | ✓ Matches standard |
| `0010 (2)` = Last | `0010 (2)` = Last | ✓ Matches standard |
| `0011 (3)` = Middle | `0011 (3)` = Middle | ✓ Matches standard |

**Update**: Upon closer analysis, the device actually follows the official specification correctly. The confusion arose from initial parsing errors.

### 4.2 Packet Header Validation
Each packet properly starts with the official JT1078 header:
```hex
30 31 63 64  // Official RTP magic header for JT1078
```

---

## 5. Frame Fragmentation Analysis

## 5. Frame Fragmentation Analysis

### 5.1 Analysis of Captured Frames
From the stream analysis, we identified clear fragmentation patterns following the official JT1078 specification:

**Example P-Frame Reconstruction (Frames 1-5)**:
```
Frame 1: Seq=0, SubPkg=1, Size=950 bytes  → First fragment (official: 0001)
Frame 2: Seq=1, SubPkg=3, Size=950 bytes  → Middle fragment (official: 0011)  
Frame 3: Seq=2, SubPkg=3, Size=950 bytes  → Middle fragment (official: 0011)
Frame 4: Seq=3, SubPkg=3, Size=950 bytes  → Middle fragment (official: 0011)
Frame 5: Seq=4, SubPkg=2, Size=454 bytes  → Last fragment (official: 0010)
Total: 4254 bytes for complete P-frame
```

**Example I-Frame Reconstruction (Frames 6-33)**:
```
Frame 6:  Seq=5,  SubPkg=1, Size=950 bytes → First (contains SPS/PPS/IDR)
Frame 7:  Seq=6,  SubPkg=3, Size=950 bytes → Middle
Frame 8:  Seq=7,  SubPkg=3, Size=950 bytes → Middle
...
Frame 33: Seq=32, SubPkg=2, Size=219 bytes → Last
Total: 26,669 bytes for complete I-frame
```

### 5.2 H.264 Content Detection
The I-frame (Frame 6) contains proper H.264 NAL units:
```hex
00000001 67 42000a965402802d88  → SPS (Sequence Parameter Set)
00000001 68 ce3c80               → PPS (Picture Parameter Set)  
00000001 65 888040027c           → IDR frame data
```

---

## 6. Complete Decoding Process

### 6.1 Two-Stage Processing Architecture

**Stage 1: JT808 Signaling Processing**
```go
func processJT808Signaling(data []byte) {
    if data[0] == 0x7e && data[len(data)-1] == 0x7e {
        // Unescape the data
        unescaped := unescapeJT808(data[1:len(data)-1])
        
        // Parse message header
        msgID := binary.BigEndian.Uint16(unescaped[0:2])
        
        // Handle different commands (0x9101, 0x9102, etc.)
        switch msgID {
        case 0x9101: // Start video streaming
            handleStartVideo(unescaped)
        case 0x9102: // Stop video streaming  
            handleStopVideo(unescaped)
        }
    }
}
```

**Stage 2: JT1078 Video Stream Processing**
```go
func processJT1078Stream(data []byte) {
    if len(data) >= 30 && 
       data[0] == 0x30 && data[1] == 0x31 && 
       data[2] == 0x63 && data[3] == 0x64 {
        
        header := parseJT1078Header(data[:30])
        payload := data[30:]
        
        reconstructFrame(header, payload)
    }
}
```

### 6.3 H.264 Stream Generation
```go
func writeH264Frame(frameData []byte, isIFrame bool) {
    if isIFrame {
        // For I-frames, ensure SPS/PPS are written first
        // These are already embedded in the frame data
        h264File.Write(frameData)
    } else {
        // For P/B frames, write directly
        h264File.Write(frameData)  
    }
}
```

---

## 7. Implementation Guidelines

### 7.1 Recommended Architecture
```
Input Stream → [Protocol Detector] → [JT808 Handler] → Signaling Commands
                                  → [JT1078 Handler] → Video Stream
                                                    → Frame Reconstructor
                                                    → H.264 Output
                                                    → MP4 Conversion
```

### 7.2 Key Implementation Points

1. **Protocol Detection**: Always check first 4 bytes to distinguish between protocols
   - `0x7e xx xx xx` = JT808 signaling
   - `0x30 0x31 0x63 0x64` = JT1078 video stream

2. **Escape Handling**: Only apply JT808 escape sequences to signaling data, never to video stream

3. **Subpackage Handling**: Follow official JT1078 specification:
   - `0` = Atomic (complete frame)
   - `1` = First fragment  
   - `2` = Last fragment
   - `3` = Middle fragment

4. **Sequence Validation**: Use sequence numbers to detect lost packets and reorder fragments

5. **Buffer Management**: Implement proper cleanup for incomplete frames to prevent memory leaks

### 7.3 Common Pitfalls to Avoid

1. **Don't mix protocols**: JT808 and JT1078 have different structures and processing rules
2. **Don't apply escape sequences to video data**: Only JT808 signaling uses escape sequences
3. **Don't assume standard subpackage values**: Always verify against specification
4. **Don't ignore sequence numbers**: They're critical for proper frame reconstruction

---

## 8. Conclusion

The JT1078 protocol implementation requires careful separation of signaling (JT808) and video stream (JT1078) protocols. The captured device follows the official specifications correctly, with proper RTP-based video streaming format. Success depends on:

1. Correct protocol identification and routing
2. Proper implementation of the official subpackage fragmentation scheme  
3. Accurate H.264 frame reconstruction
4. Appropriate handling of escape sequences only for signaling data

This analysis provides the foundation for implementing a robust JT1078 video decoder that can handle real-world device variations while adhering to the official protocol standards.

### 6.3 H.264 Stream Generation
   if frameType == 0 {
       extractAndWriteSPS(frameData)
       extractAndWritePPS(frameData)
   }
   writeNALWithStartCode(h264File, frameData)
   ```

### 4.2 Key Implementation Points

- **Sequence Number Validation**: Check for gaps in sequence numbers
- **Timestamp Consistency**: Group fragments by timestamp
- **NAL Unit Detection**: Look for `0x00000001` start codes
- **Parameter Set Extraction**: Extract SPS (0x67) and PPS (0x68) from I-frames

---

## 5. Expected H.264 Output

### 5.1 Proper H.264 File Structure
A correctly reconstructed H.264 file should contain:

```
[Start Code] [SPS NAL Unit]
[Start Code] [PPS NAL Unit]  
[Start Code] [IDR Frame Data]
[Start Code] [P-Frame Data]
[Start Code] [P-Frame Data]
...
```

### 5.2 Quality Validation
- **Resolution**: 1280x720 (extracted from SPS)
- **Profile**: H.264 Baseline/High profile
- **Frame Rate**: Variable, based on device capture rate
- **Bitrate**: Approximately 1-3 Mbps

---

## 6. Conversion to MP4

### 6.1 Using FFmpeg
Once a proper H.264 file is generated:

```bash
# Basic conversion
ffmpeg -i captured_video.h264 -c copy output.mp4

# With frame rate specification  
ffmpeg -r 25 -i captured_video.h264 -c:v libx264 output.mp4

# With timing correction
ffmpeg -fflags +genpts -i captured_video.h264 -c copy output.mp4
```

### 6.2 Common Issues and Solutions
- **Missing SPS/PPS**: Extract from I-frames and prepend to stream
- **Incorrect timestamps**: Use `-fflags +genpts` to regenerate
- **Frame ordering**: Ensure I-frames come before dependent P-frames

---

## 7. Next Steps for Implementation

### 7.1 Minimal Viable Decoder
Create a focused program that:
1. Reads JT1078 stream data
2. Parses headers correctly
3. Reconstructs frames using device-specific SubPackage mapping
4. Outputs raw H.264 file
5. Converts to MP4 using FFmpeg

### 7.2 Testing Strategy
- Use the captured `raw_stream.bin` (2.3MB) as test data
- Validate output against known H.264 structure
- Compare with reference video `dvr-ch-03.mp4`

---

## 8. Files Referenced

- `/stream-capture/raw_stream.bin` - 2.3MB captured stream
- `/stream-capture/stream_parse.log` - Detailed frame analysis
- `/stream-capture/main.go` - Stream capture and parsing code

This analysis provides the foundation for building a robust JT1078 to H.264/MP4 converter with proper understanding of the protocol specifics and device behavior.
