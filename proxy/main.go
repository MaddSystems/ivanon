// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"proxy/api"
	"proxy/services"
	"proxy/shared"

	_ "proxy/docs" //  <-- ADD THIS BLANK IMPORT FOR SWAGGER
)

func main() {
	// Define and parse command-line flags
	localAddress := flag.String("l", "0.0.0.0:1024", "Local address")
	remoteAddress := flag.String("r", os.Getenv("PLATFORM_HOST"), "Remote address")
	verbose := flag.Bool("v", false, "Enable verbose logging")
	flag.Parse()

	// Initialize shared utilities from the correct package
	shared.InitializeUtils(*verbose, *remoteAddress)

	fmt.Printf("Listening: %v\nProxying %v\n", *localAddress, *remoteAddress)

	// Initialize the MQTT client
	services.InitializeMQTT()

	// Start the background cleanup routine for old snapshots
	go services.SnapshotCleanupRoutine()

	// Start the Gin HTTP server
	go func() {
		router := api.SetupRouter()
		if err := router.Run(":8080"); err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
	}()

	// Resolve the TCP address for the listener
	addr, err := net.ResolveTCPAddr("tcp", *localAddress)
	if err != nil {
		panic(err)
	}

	// Create the TCP listener
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Accept and handle incoming TCP connections
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go services.ProxyConnection(conn)
	}
}
