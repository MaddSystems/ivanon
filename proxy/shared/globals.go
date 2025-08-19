package shared

import (
	"net"
	"proxy/models"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	// Concurrency control
	ConnMutex sync.Mutex

	// Connection and device tracking
	ActiveConnections = make(map[string]net.Conn)
	ImeiConnections   = make(map[string]models.ConnectionInfo)
	JT808Devices      = make(map[string]*models.JT808Device)

	// Feature-specific state
	ActiveSnapshots = make(map[uint32]*models.ImageSnapshot)
	ImageChunks     = make(map[uint32]map[uint16]*models.ImageChunk)
	EarlyChunks     = make(map[string][]*models.PendingChunk)

	// MQTT Client
	MQTTClient mqtt.Client
)
