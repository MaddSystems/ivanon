# JT1078 Two-Way Audio (twoway)

This folder implements a two-way audio communication system for JT1078-compliant devices, with both backend and frontend components.

## main.go
- **Backend server for two-way audio**
- Listens for JT1078 audio streams from devices on TCP port 7800
- Provides WebSocket endpoints for browser audio receive (`/ws`) and transmit (`/transmit`)
- Proxies API calls for device management and call control (e.g., `/api/devices`, `/api/call/start`)
- Handles G.711 A-law audio decoding/encoding and relays audio between browser and device
- Serves static files (including the frontend UI)

## index.html
- **Frontend web interface for two-way audio**
- Lets users select a device, start/stop calls, and view device/call status
- Receives live audio from the device and plays it in the browser
- Allows microphone audio to be sent to the device (push-to-talk)
- Shows real-time metrics for buffer, latency, and connection status
- Provides a modern, user-friendly UI for managing and testing two-way audio

## Usage
1. Start the backend: `go run main.go`
2. Open `index.html` in your browser (or visit the served web UI)
3. Select a device, start a call, and use the push-to-talk and receive features for two-way audio

This tool is intended for testing, integration, and demonstration of real-time two-way audio with JT1078 devices.
