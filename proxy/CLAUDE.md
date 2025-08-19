# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is an IoT device proxy system written in Go that handles:
- GPS tracker communication via JT808 protocol
- Video streaming from devices
- Real-time image snapshots
- TCP proxy for device data
- REST API for device management
- MQTT integration for data forwarding

## Architecture

### Core Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   TCP Listener  │    │   HTTP Server   │    │   MQTT Client   │
│   (Port 1024)   │────│   API (8080)    │────│   (Broker)      │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
   │   TCP Proxy     │    │   Handlers      │    │   Services      │
   │   Service       │    │   (REST API)    │    │   (Business)    │
   └─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Services

- **tcp_proxy.go**: Core TCP proxy for device connections
- **jt808_handler.go**: JT808 protocol handling for GPS trackers
- **mqtt.go**: MQTT client for data forwarding to cloud
- **client_manager.go**: Manages active device connections (TCP & HTTP clients)
- **snapshot.go**: Handles image snapshot requests and processing

### Protocols

- **JT808**: GPS tracker communication protocol
- **HTTP/REST**: API layer for device management
- **MQTT**: Message queuing for telemetry data
- **TCP**: Raw device connection proxy

## API Endpoints

The REST API is documented with Swagger at `/swagger/index.html` when running.

Key endpoints:
- `GET /devices` - List active devices
- `POST /command` - Send commands to devices (hex encoded)
- Video streaming endpoints
- Image snapshot endpoints
- VoIP call management

## Development Commands

### Build
```bash
go build -o proxy main.go
./proxy -l 0.0.0.0:1024 -r target-host:port
```

### Run with Environment Variables
```bash
export PLATFORM_HOST=your-target-host:port
./proxy
```

### Flags
- `-l`: Local listen address (default: 0.0.0.0:1024)
- `-r`: Remote target address (or use PLATFORM_HOST env)
- `-v`: Enable verbose logging

### Ports
- `1024`: Primary TCP proxy port for device connections
- `8080`: HTTP API server for device management

### Dependencies
Run `go mod tidy` to ensure all dependencies are present.

### Database Schema
No database - uses in-memory structs for active connections with automatic cleanup routines.

## Key Data Structures

- `JT808Device`: Active GPS tracker connection state
- `VideoSession`: Video streaming session management
- `ImageSnapshot`: Multi-part image assembly for device snapshots
- `TrackerData`: MQTT message format for telemetry

## Testing

Test device connection with telnet:
```bash
telnet localhost 1024
```

Test API endpoints:
```bash
curl http://localhost:8080/devices
curl -X POST http://localhost:8080/command -H "Content-Type: application/json" -d '{"imei":"123456789","data":"0123456789ABCDEF"}'
```

## Project Structure

```
├── main.go              # Entry point with CLI flags
├── go.mod              # Dependencies
├── models/             # Data structures and types
├── services/           # Core business logic
├── api/                # REST API handlers and routing
├── docs/               # Swagger documentation
├── jt808/              # JT808 protocol implementation
└── shared/             # Common utilities and globals
```

## Environment Configuration

- `PLATFORM_HOST`: Remote address for TCP proxy
- No config file - all configuration via flags/environment