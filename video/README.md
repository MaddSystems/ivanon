# JT1078 Video & Audio Streaming Server

A complete implementation of JT1078 video/audio streaming protocol with real-time web playback using WebCodecs API and Web Audio API.

## Architecture Overview

```
JT1078 Device → TCP Server → Frame Parser → WebSocket Clients → Web Browser
    (G.711A)      (Go)       (Reassembly)     (Dual WS)      (HTML5 Player)
```

## Core Components

### 1. Backend: `main.go` - Go TCP/WebSocket Server

#### **Protocol Implementation**
- **JT1078 Frame Parser**: Handles official RTP-based video streaming protocol
- **Dual Protocol Support**: Separates JT808 signaling from JT1078 video streams
- **Frame Extraction**: Proven algorithm based on `stream-capture` analysis

#### **TCP Server (Port 7800)**
```go
func startTCPServer() {
    // Listens for JT1078 device connections
    // Processes incoming video/audio frames
    // Handles frame fragmentation and reassembly
}
```

#### **Frame Processing Pipeline**
1. **Header Detection**: Searches for `0x30 0x31 0x63 0x64` signature
2. **Frame Parsing**: Extracts metadata (channel, type, sequence, timestamp)
3. **Data Type Classification**:
   - `DataType 0`: I-Frame (keyframe with SPS/PPS)
   - `DataType 1`: P-Frame (predictive frame)
   - `DataType 2`: B-Frame (bidirectional frame)
   - `DataType 3`: Audio Frame (G.711A encoded)

#### **Video Frame Reassembly**
```go
type VideoFrameAssembler struct {
    Channel         int
    SequenceNum     uint16
    FrameType       int
    Fragments       []Fragment
    JT1078Timestamp uint64  // Key for grouping fragments
}
```

**Fragmentation Handling**:
- `SubType 0`: Atomic (complete frame)
- `SubType 1`: First fragment
- `SubType 2`: Last fragment  
- `SubType 3`: Middle fragment

#### **Audio Processing**
```go
func processAudioFrame(frame *JT1078Frame) {
    // CRITICAL: Sends raw G.711A data (not converted PCM)
    // Maintains original JT1078 audio format
    // Duration calculated: payloadSize / 8000Hz
}
```

#### **WebSocket Endpoints**
- **`/video`** → Video frames only (client type: "video")
- **`/ws`** → Audio frames only (client type: "receiver")
- **`/transmit`** → Audio transmission (client type: "transmitter")

#### **API Proxy Endpoints**
- `POST /api/video/start` → Start video streaming
- `POST /api/video/control` → Control video (stop/pause)
- `GET /api/devices` → List available devices
- `POST /api/call/start` → Start audio call
- `POST /api/call/control` → Control audio call

### 2. Frontend: `index.html` - HTML5 Video Player

#### **WebCodecs H.264 Decoder**
```javascript
class VideoPlayer {
    setupVideoDecoder() {
        this.videoDecoder = new VideoDecoder({
            output: this.renderVideoFrame.bind(this),
            error: this.handleVideoError.bind(this)
        });
        
        // Configure for H.264 Baseline Profile
        this.videoDecoder.configure({
            codec: 'avc1.42001f',  // H.264 Baseline Level 3.1
            codedWidth: 1280,
            codedHeight: 720
        });
    }
}
```

#### **Dual WebSocket Architecture**
```javascript
// Video WebSocket (receives video frames)
this.videoWs = new WebSocket('ws://localhost:8081/video');
this.videoWs.onmessage = (event) => {
    this.handleStreamFrame(event.data, 'video');
};

// Audio WebSocket (receives audio data)  
this.audioWs = new WebSocket('ws://localhost:8081/ws');
this.audioWs.onmessage = (event) => {
    this.handleAudioWebSocketData(event.data);
};
```

#### **Frame Processing**
```javascript
handleStreamFrame(arrayBuffer, source) {
    // Parse JT1078 frame format: [8 byte header] + [payload]
    const view = new DataView(arrayBuffer);
    const channel = view.getUint8(0);
    const frameType = view.getUint8(1);
    const seqNum = view.getUint16(2, false);
    const dataLength = view.getUint32(4, false);
    const payloadData = new Uint8Array(arrayBuffer, 8, dataLength);
    
    if (frameType <= 2) {
        this.handleVideoFrame(payloadData, frameType, seqNum, channel);
    } else if (frameType === 3) {
        this.handleAudioFrame(payloadData, seqNum, channel);
    }
}
```

#### **Audio Format Handling**
```javascript
handleAudioWebSocketData(arrayBuffer) {
    // Parse: [4 bytes duration] + [G.711A data]
    const view = new DataView(arrayBuffer);
    const durationFloat = view.getFloat32(0, true);
    const g711aData = new Uint8Array(arrayBuffer, 4);
    
    // Process G.711A audio through Web Audio API
    this.processAudioFrame(g711aData);
}
```

#### **G.711A Decoder**
```javascript
decodeG711ALaw(alawData) {
    const pcm16 = new Int16Array(alawData.length);
    for (let i = 0; i < alawData.length; i++) {
        pcm16[i] = this.alawToPcm16Table[alawData[i]];
    }
    return pcm16;
}
```

#### **Audio Scheduler**
```javascript
class AudioScheduler {
    scheduleAudioFrame(audioData) {
        const decoded = this.decodeG711ALaw(audioData);
        const audioBuffer = this.audioContext.createBuffer(1, decoded.length, 8000);
        const channelData = audioBuffer.getChannelData(0);
        
        // Convert Int16 to Float32 [-1, 1]
        for (let i = 0; i < decoded.length; i++) {
            channelData[i] = decoded[i] / 32768.0;
        }
        
        // Schedule playback with proper timing
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);
        source.start(this.nextPlayTime);
    }
}
```

## Data Flow

### Video Stream Flow
```
JT1078 Device → TCP:7800 → processVideoFrame() → videoFrames[timestamp] 
    → reconstructVideoData() → broadcastVideoFrame() → WS:/video 
    → handleStreamFrame() → VideoDecoder → Canvas Rendering
```

### Audio Stream Flow
```
JT1078 Device → TCP:7800 → processAudioFrame() → frameBuffer 
    → broadcastFrames() → WS:/ws → handleAudioWebSocketData() 
    → G.711A Decoder → AudioContext → Speaker Output
```

## Frame Formats

### JT1078 Video Frame (from device)
```
Bytes 0-3:   0x30 0x31 0x63 0x64 (header signature)
Bytes 4-5:   V/P/X/CC and PT fields  
Bytes 6-7:   Sequence number
Bytes 8-13:  SIM card number (BCD)
Byte 14:     Logic channel number
Byte 15:     DataType (high 4 bits) + SubType (low 4 bits)
Bytes 16-23: JT1078 timestamp (8 bytes)
Bytes 24-25: Additional header for video
Bytes 26-27: Payload length
Bytes 28+:   H.264/G.711A payload
```

### WebSocket Video Message (to browser)
```
Byte 0:      Channel
Byte 1:      Frame type (0=I, 1=P, 2=B)
Bytes 2-3:   Sequence number (big endian)
Bytes 4-7:   Data length (big endian)
Bytes 8+:    H.264 payload data
```

### WebSocket Audio Message (to browser)
```
Bytes 0-3:   Duration (float32, little endian)
Bytes 4+:    Raw G.711A audio data
```

## JT1078 Video Stream Activation

To start a real-time video stream, send a **0x9101** command to the device:

| Field       | Type   | Description                                 |
|-------------|--------|---------------------------------------------|
| 0           | BYTE   | Server IP address length (n)               |
| 1           | STRING | Server IP address (n bytes)                |
| 1+n         | WORD   | Server TCP port (7800)                     |
| 3+n         | WORD   | Server UDP port                            |
| 5+n         | BYTE   | Logical channel number                     |
| 6+n         | BYTE   | Data type (0=AV, 1=Video, 2=Talk, 3=Monitor)|
| 7+n         | BYTE   | Stream type (0=Main, 1=Sub)                |

**Example Usage**:
- **Live Video**: Data type = 1 (video) or 0 (audio+video), Stream type = 1 (sub-stream)
- **Stop Stream**: Send **0x9102** control command

## Key Features

### ✅ **Video Features**
- **H.264 Decoding**: Real-time H.264 video using WebCodecs API
- **Frame Reassembly**: Handles fragmented video frames correctly
- **SPS/PPS Extraction**: Automatic codec parameter detection
- **Multi-Channel Support**: Supports multiple camera channels
- **Resolution**: 1280x720 @ variable FPS

### ✅ **Audio Features**  
- **G.711A Decoding**: Real-time G.711A audio decoding
- **Synchronized Playback**: Proper audio timing and scheduling
- **Buffer Management**: Audio frame buffering for smooth playback
- **Autoplay Handling**: Manages browser autoplay policies

### ✅ **Protocol Compliance**
- **JT1078 Standard**: Follows official RTP-based protocol
- **Fragmentation**: Proper handling of SubType fragmentation
- **Timestamp Grouping**: Uses JT1078 timestamps for frame assembly
- **Multi-Device Support**: Handles multiple concurrent device connections

## Configuration

### Device Connection
- **TCP Port**: 7800 (for JT1078 device connections)
- **Web Port**: 8081 (for browser interface)

### Video Settings
- **Codec**: H.264 Baseline Profile Level 3.1
- **Resolution**: 1280x720 (configurable via SPS)
- **Frame Rate**: Variable (device dependent)

### Audio Settings
- **Codec**: G.711A (A-law PCM)
- **Sample Rate**: 8000 Hz
- **Channels**: 1 (mono)
- **Frame Size**: ~320 bytes (40ms duration)

## Usage

### Start Server
```bash
cd /home/systemd/ivanguard/video
go run main.go -ds  # -ds enables debug logs
```

### Access Web Interface
```
http://localhost:8081
```

### Device Commands
1. **Select Device**: Choose from available GPS tracking devices
2. **Select Channel**: Choose camera channel (1-4)
3. **Start Video**: Initiates JT1078 video streaming
4. **Stop Video**: Terminates streaming

## Debug Features

### Backend Debug Flags
- **`-ds`**: Debug send operations (audio/video transmission)
- **`-dr`**: Debug receive operations (frame parsing)

### Frontend Debug Console
- Real-time frame statistics
- WebSocket connection status
- Audio/video metrics
- Error logging

## Technical Notes

### **Critical Implementation Details**

1. **Audio Format**: Backend sends **raw G.711A**, not converted PCM
2. **Frame Grouping**: Uses **JT1078 timestamps**, not sequence numbers
3. **WebSocket Separation**: Video and audio use different endpoints
4. **Fragment Assembly**: Follows official SubType specification
5. **SPS/PPS Handling**: Extracted from I-frames for decoder configuration

### **Browser Compatibility**
- **Chrome**: Full WebCodecs + Web Audio support
- **Edge**: Full support  
- **Firefox**: Limited WebCodecs support
- **Safari**: Limited support

### **Performance Characteristics**
- **Latency**: ~100-200ms end-to-end
- **Throughput**: Supports multiple concurrent 720p streams
- **Memory**: Efficient frame buffering with cleanup
- **CPU**: Hardware-accelerated H.264 decoding when available

