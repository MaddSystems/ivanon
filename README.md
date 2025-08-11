# Project Structure Overview

### HARDCODED ADRESSES

- video/index.html
```
wss://voip.armaddia.lat/video
```

- video/main.go
```
https://ivan-proxy.armaddia.lat
```

This repository is a telematics/GPS tracking infrastructure with the following main components:

- **proxy/** — JT808 protocol proxy with VoIP, MQTT, and REST API support
- **video/** — JT1078 video/audio streaming and protocol documentation
- **voice/monitor/** — JT1078 stream analyzer for audio/video quality and protocol compliance
- **voice/twoway/** — Two-way JT1078 audio streaming and relay (browser-to-device and device-to-browser, push-to-talk)
- **voice/test/** — JT1078 audio stream capture and analysis tool

## proxy/
*Go TCP proxy for JT808 protocol devices with VoIP and MQTT integration.*

- Handles JT808 GPS tracking protocol and VoIP extensions
- Integrates with MQTT for device assignment and data forwarding
- Provides REST API endpoints for device management and VoIP calls
- Supports Docker deployment

Key endpoints:
- `GET /api/v1/jt808/devices` — List connected JT808 devices
- `POST /api/v1/jt808/call/start` — Start VoIP call
- `POST /api/v1/jt808/call/control` — Control ongoing call (end=command 4)
- `GET /api/v1/jt808/calls` — List active calls

## video/
*JT1078 video/audio streaming documentation and scripts.*

- Protocol breakdowns for JT808/JT1078 video and audio streaming
- Example message structures for 0x9101 (start video) and 0x9102 (control/stop video)
- Python scripts for log parsing and protocol analysis

## voice/monitor/
*JT1078 stream analyzer for audio/video data.*

- Listens on TCP port 7800 for incoming JT1078 streams
- Serves a web dashboard on port 8081 for live protocol/frame analysis
- Reports bitrate, frame rate, inter-frame gap, packet loss, and protocol compliance

## voice/twoway/
*Two-way JT1078 audio streaming and relay (browser-to-device and device-to-browser, push-to-talk).* 

- Backend server for two-way audio (Go)
- WebSocket endpoints for browser audio receive (`/ws`) and transmit (`/transmit`)
- G.711A audio decoding/encoding and relay between browser and device
- Modern web UI for device selection, call control, push-to-talk, and real-time metrics

## voice/test/
*JT1078 audio stream capture and analyzer.*

- Captures and saves G.711A audio payloads from JT1078 streams
- Reports stream statistics (bitrate, frame rate, gaps, packet loss)
- Serves a web UI for live stats and frame breakdown

## Environment Variables

### Proxy Configuration
- `PLATFORM_HOST` — Backend server (<host>:<port>)
- `MQTT_BROKER_HOST` — MQTT broker (default: localhost)
- `AUDIO_SERVER_IP` — VoIP server IP (default: 127.0.0.1)
- `AUDIO_SERVER_PORT` — VoIP port (default: 7800)
- `VOIP_SERVER_URL` — VoIP service endpoint

## Build and Run Commands

### proxy
```bash
cd proxy
go mod tidy
go build -o proxy
# Run with environment variables
PLATFORM_HOST=your.platform.com:9999 ./proxy -l 0.0.0.0:1024
```

### voice/monitor
```bash
cd voice/monitor
go mod tidy
go build -o monitor
./monitor
# Access web UI at http://localhost:8081
```

### voice/twoway
```bash
cd voice/twoway
go mod tidy
go build -o twoway
./twoway
# Access web UI at http://localhost:8081
```

### voice/test
```bash
cd voice/test
go mod tidy
go build -o test
./test
# Access web UI at http://localhost:8081
```

## Protocol Details

- **JT808**: GPS vehicle tracking protocol (Chinese standard)
- **JT1078**: Audio/video streaming for vehicle surveillance
- **Codec**: G.711A (64kbps) or PCM-S16LE (8kHz, mono)
- **Network**: TCP streams, proprietary framing with 0x30 0x31 0x63 0x64 header

## File Structure Highlights

- `proxy/main.go` — Core proxy logic with MQTT and protocol handling
- `proxy/api.go` — REST API endpoints with Swagger documentation
- `video/` — Protocol docs and scripts for JT1078 video/audio
- `voice/monitor/main.go` — JT1078 streaming analyzer with web dashboard
- `voice/twoway/main.go` — Two-way audio relay and web UI
- `voice/test/main.go` — Audio stream capture and analyzer