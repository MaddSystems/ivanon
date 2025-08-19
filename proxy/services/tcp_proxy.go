// services/tcp_proxy.go
package services

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"proxy/models"
	"proxy/shared"
)

// ProxyConnection manages the bi-directional data flow for a single TCP connection.
func ProxyConnection(conn *net.TCPConn) {
	remoteAddr := conn.RemoteAddr().String()
	defer conn.Close()
	defer DeregisterClient(remoteAddr)

	shared.VPrint("New connection from: %s", remoteAddr)

	shared.ConnMutex.Lock()
	shared.ActiveConnections[remoteAddr] = conn
	shared.ConnMutex.Unlock()

	rAddr, err := net.ResolveTCPAddr("tcp", shared.RemoteAddress())
	if err != nil {
		shared.VPrint("Failed to resolve remote address: %v", err)
		return
	}
	rConn, err := net.DialTCP("tcp", nil, rAddr)
	if err != nil {
		shared.VPrint("Failed to connect to remote server: %v", err)
		return
	}
	defer rConn.Close()

	done := make(chan struct{})
	go forwarder(conn, rConn, processClientData, conn, remoteAddr, &done)
	go forwarder(rConn, conn, processPlatformData, conn, remoteAddr, &done)

	<-done
	<-done
	shared.VPrint("Connection closed for: %s", remoteAddr)
}

// forwarder reads from a source, writes to a destination, and processes the data.
func forwarder(src, dest net.Conn, processFunc func(net.Conn, []byte, string), conn net.Conn, remoteAddr string, done *chan struct{}) {
	defer func() { (*done) <- struct{}{} }()
	buffer := make([]byte, 0, 4096)
	readBuf := make([]byte, 2048)
	for {
		n, err := src.Read(readBuf)
		if err != nil {
			if err != io.EOF {
				shared.VPrint("Error reading from %s: %v", src.RemoteAddr(), err)
			}
			break
		}
		if n > 0 {
			if _, err := dest.Write(readBuf[:n]); err != nil {
				shared.VPrint("Error writing to %s: %v", dest.RemoteAddr(), err)
				break
			}
			buffer = append(buffer, readBuf[:n]...)
			for {
				startIdx := bytes.IndexByte(buffer, 0x7e)
				if startIdx == -1 {
					buffer = buffer[:0]
					break
				}
				if startIdx > 0 {
					buffer = buffer[startIdx:]
				}
				endIdx := bytes.IndexByte(buffer[1:], 0x7e)
				if endIdx == -1 {
					break
				}
				endIdx += 1
				msg := make([]byte, endIdx+1)
				copy(msg, buffer[:endIdx+1])
				go processFunc(conn, msg, remoteAddr)
				buffer = buffer[endIdx+1:]
			}
		}
	}
}

// processClientData handles data coming from the device.
func processClientData(conn net.Conn, data []byte, remoteAddr string) {
	publishToMQTT(data, remoteAddr)
	shared.VPrint("From tracker to platform:\n%s", hex.Dump(data))
	// This now calls the function within the same 'services' package
	HandleJT808Message(conn, data, remoteAddr)
}

// processPlatformData handles data coming from the remote platform.
func processPlatformData(conn net.Conn, data []byte, remoteAddr string) {
	shared.VPrint("From platform to tracker:\n%s", hex.Dump(data[:shared.Min(32, len(data))]))
	TrackDeviceFromPlatform(data)
}

// publishToMQTT sends data to the MQTT broker.
func publishToMQTT(data []byte, remoteAddr string) {
	trackerData := models.TrackerData{
		Payload:    hex.EncodeToString(data),
		RemoteAddr: remoteAddr,
	}
	payload, err := json.Marshal(trackerData)
	if err != nil {
		log.Printf("Error creating MQTT JSON: %v", err)
		return
	}

	if shared.MQTTClient != nil && shared.MQTTClient.IsConnected() {
		token := shared.MQTTClient.Publish("tracker/from-tcp", 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			shared.VPrint("Error publishing to MQTT: %v", token.Error())
		}
	}
}
