package services

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"proxy/models"
	"proxy/shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// InitializeMQTT sets up and connects the MQTT client.
func InitializeMQTT() {
	opts := mqtt.NewClientOptions()
	mqttBrokerHost := os.Getenv("MQTT_BROKER_HOST")
	if mqttBrokerHost == "" {
		mqttBrokerHost = "localhost"
	}
	opts.AddBroker(fmt.Sprintf("tcp://%s:1883", mqttBrokerHost))
	opts.SetClientID("proxy_service")
	opts.SetOnConnectHandler(onMQTTConnect)

	shared.MQTTClient = mqtt.NewClient(opts)
	if token := shared.MQTTClient.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Failed to connect to MQTT broker: %v", token.Error())
	}
}

// onMQTTConnect sets up subscriptions once the client is connected.
func onMQTTConnect(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
	// Subscribe to tracker/send topic
	if token := client.Subscribe("tracker/send", 0, handleTrackerSend); token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe to tracker/send: %v", token.Error())
	}
	// Subscribe to tracker/assign-imei2remoteaddr topic
	if token := client.Subscribe("tracker/assign-imei2remoteaddr", 0, handleImeiAssignment); token.Wait() && token.Error() != nil {
		log.Printf("Failed to subscribe to tracker/assign-imei2remoteaddr: %v", token.Error())
	}
}

// handleTrackerSend handles messages to send data to a specific device.
func handleTrackerSend(client mqtt.Client, msg mqtt.Message) {
	var data models.TrackerData
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		shared.VPrint("Error unmarshaling MQTT message: %v", err)
		return
	}
	rawBytes, err := hex.DecodeString(data.Payload)
	if err != nil {
		shared.VPrint("Error decoding hex payload: %v", err)
		return
	}

	shared.ConnMutex.Lock()
	conn, exists := shared.ActiveConnections[data.RemoteAddr]
	shared.ConnMutex.Unlock()
	if !exists {
		shared.VPrint("No active connection for address: %s", data.RemoteAddr)
		return
	}
	if _, err := conn.Write(rawBytes); err != nil {
		shared.VPrint("Error writing to connection %s: %v", data.RemoteAddr, err)
	}
}

// handleImeiAssignment handles messages that map an IMEI to a connection.
func handleImeiAssignment(client mqtt.Client, msg mqtt.Message) {
	var data models.TrackerAssign
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		shared.VPrint("Error unmarshaling JSON: %v", err)
		return
	}

	shared.VPrint("Assigning imei: %s to address: %s from protocol %s", data.Imei, data.RemoteAddr, data.Protocol)
	shared.ConnMutex.Lock()
	shared.ImeiConnections[data.Imei] = models.ConnectionInfo{
		RemoteAddr: data.RemoteAddr,
		Protocol:   data.Protocol,
	}
	shared.ConnMutex.Unlock()
}
