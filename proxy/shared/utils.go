package shared

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"log"
)

var (
	verbose       bool
	remoteAddress string
)

// InitializeUtils sets up package-level variables from main.
func InitializeUtils(v bool, r string) {
	verbose = v
	remoteAddress = r
}

// VPrint prints logs only when the verbose flag is enabled.
func VPrint(format string, v ...interface{}) {
	if verbose {
		log.Printf(format, v...)
	}
}

// RemoteAddress returns the configured remote address.
func RemoteAddress() string {
	return remoteAddress
}

// GenerateSerial creates a random 16-bit serial number.
func GenerateSerial() uint16 {
	b := make([]byte, 2)
	rand.Read(b)
	return binary.BigEndian.Uint16(b)
}

// GenerateCallID creates a random hex-encoded string for call IDs.
func GenerateCallID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Min returns the smaller of two integers.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
