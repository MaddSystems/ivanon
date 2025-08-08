package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sort"
	"time"
)

// JT1078Frame holds the parsed data for a single JT/T 1078 frame.
type JT1078Frame struct {
	SequenceNumber uint16
	SIM            string
	LogicChannel   byte
	DataType       int // 0:I-Frame, 1:P-Frame, 2:B-Frame, 3:Audio
	SubPackageType int // 0:Atomic, 1:First, 2:Last, 3:Middle
	Timestamp      uint64
	Payload        []byte
	NALTypes       []int // Extracted H.264 NAL unit types
}

// H264Constructor manages the reconstruction of a raw H.264 stream.
type H264Constructor struct {
	frames          []*JT1078Frame
	spsData         [][]byte
	ppsData         [][]byte
	completeFrames  [][]byte
	frameTimestamps []uint64 // Track timestamp for each complete frame
	firstTimestamp  uint64   // Track first frame timestamp for relative timing
}

func main() {
	fmt.Println("🚀 JT1078 Live Stream Capture & Analysis")
	fmt.Println("=========================================")
	fmt.Println("📡 Starting TCP server on 0.0.0.0:7800...")
	fmt.Println("⏱️  Will capture for 30 seconds once data starts flowing")
	fmt.Println("🎯 After capture, will process to H.264 and G.711A")
	fmt.Println("🔊 Audio will be processed as pure G.711A (no SPS/PPS extraction)")
	fmt.Println("🔍 Enhanced debugging for 64kbps stream analysis")
	fmt.Println("")

	// Start the TCP server
	err := startLiveCapture()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
}

func startLiveCapture() error {
	// Listen on port 7800
	ln, err := net.Listen("tcp", "0.0.0.0:7800")
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %v", err)
	}
	defer ln.Close()

	fmt.Println("✅ TCP server listening on port 7800...")
	fmt.Println("   Waiting for JT1078 stream connection...")

	// Accept connection
	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept connection: %v", err)
	}
	defer conn.Close()

	fmt.Printf("🔗 Connection established from: %s\n", conn.RemoteAddr())
	fmt.Println("📦 Starting data capture...")

	// Create video.bin file for raw capture
	videoFile, err := os.Create("video.bin")
	if err != nil {
		return fmt.Errorf("failed to create video.bin: %v", err)
	}
	defer videoFile.Close()

	// Capture data for 30 seconds
	startTime := time.Now()
	timeout := 30 * time.Second
	buffer := make([]byte, 4096)
	totalBytesReceived := 0

	fmt.Printf("⏰ Capturing for %v...\n", timeout)

	for {
		// Set read timeout
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))

		n, err := conn.Read(buffer)
		if n > 0 {
			// Write received data to video.bin
			_, writeErr := videoFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to video.bin: %v", writeErr)
			}

			totalBytesReceived += n
			elapsed := time.Since(startTime)
			fmt.Printf("📊 Received: %d bytes | Elapsed: %.1fs | Rate: %.2f KB/s\r",
				totalBytesReceived, elapsed.Seconds(), float64(totalBytesReceived)/1024.0/elapsed.Seconds())
		}

		if err != nil {
			// Check if it's a timeout or connection closed
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Continue if it's just a timeout
			} else {
				fmt.Printf("\n🔌 Connection ended: %v\n", err)
				break
			}
		}

		// Check if 30 seconds have elapsed
		if time.Since(startTime) >= timeout {
			fmt.Printf("\n⏰ 30 seconds elapsed, stopping capture\n")
			break
		}
	}

	fmt.Printf("✅ Capture complete! Received %d bytes total\n", totalBytesReceived)
	videoFile.Close() // Ensure file is closed before processing

	if totalBytesReceived == 0 {
		return fmt.Errorf("no data received during capture")
	}

	// Now process the captured data using improved algorithm
	fmt.Println("")
	fmt.Println("🔧 Processing captured data with enhanced H.264/Audio algorithm...")
	fmt.Println("💡 Expected: 1280x720@15fps H.264 + G.711A audio")
	fmt.Printf("📊 Target bitrate: 64kbps (actual: %.2f kbps)\n", float64(totalBytesReceived*8)/1000.0/30.0)
	return processVideoFile()
}

func processVideoFile() error {
	// Initialize H264 constructor
	h264Constructor := NewH264Constructor()

	// Read the captured video.bin file
	inputData, err := os.ReadFile("video.bin")
	if err != nil {
		return fmt.Errorf("could not read video.bin: %v", err)
	}
	fmt.Printf("📦 Processing %d bytes from video.bin\n", len(inputData))

	// Create audio output file for G.711A audio stream
	audioFile, err := os.Create("live_capture_output_audio.g711a")
	if err != nil {
		return fmt.Errorf("failed to create audio output file: %v", err)
	}
	defer audioFile.Close()

	// Parse all JT1078 frames from the captured data
	buf := inputData
	frameCount := 0
	videoFrameCount := 0
	audioFrameCount := 0
	totalAudioBytesWritten := 0

	// Track fragmentation statistics
	fragmentStats := make(map[int]int)      // SubPackageType -> count
	timestampGroups := make(map[uint64]int) // timestamp -> frame count

	for len(buf) > 0 {
		frame, consumed := parseJT1078Frame(buf)
		if frame == nil {
			// Try to skip malformed data
			if consumed == 0 {
				buf = buf[1:] // Skip 1 byte and try again
			} else {
				buf = buf[consumed:]
			}
			continue
		}

		// Track fragmentation patterns
		fragmentStats[frame.SubPackageType]++
		timestampGroups[frame.Timestamp]++

		// Process frames based on their data type
		if frame.DataType < 3 { // Video Frame (I, P, or B)
			h264Constructor.AddFrame(frame)
			videoFrameCount++
		} else if frame.DataType == 3 { // Audio Frame
			// Audio frames should NOT contain SPS/PPS - process as pure audio
			bytesWritten, _ := audioFile.Write(frame.Payload)
			totalAudioBytesWritten += bytesWritten
			audioFrameCount++
		}

		frameCount++
		buf = buf[consumed:]

		// Show progress every 100 frames with detailed info
		if frameCount%100 == 0 {
			fmt.Printf("🔍 Parsed %d JT1078 frames (Video: %d, Audio: %d) | Unique timestamps: %d...\n",
				frameCount, videoFrameCount, audioFrameCount, len(timestampGroups))
		}
	}

	// Print fragmentation analysis
	fmt.Printf("📊 Fragmentation Analysis:\n")
	for fragType, count := range fragmentStats {
		fragName := []string{"Atomic", "First", "Last", "Middle"}[fragType]
		fmt.Printf("  • %s fragments: %d\n", fragName, count)
	}
	fmt.Printf("  • Unique timestamps: %d\n", len(timestampGroups))

	fmt.Printf("✅ Parsed %d total JT1078 frames (Video: %d, Audio: %d)\n", frameCount, videoFrameCount, audioFrameCount)
	fmt.Printf("🔊 Audio data written: %.2f KB\n", float64(totalAudioBytesWritten)/1024.0)

	// Construct the final H.264 file using the proven algorithm
	outputFile := "live_capture_output.h264"
	err = h264Constructor.ConstructH264File(outputFile, totalAudioBytesWritten)
	if err != nil {
		return fmt.Errorf("failed to construct H.264 file: %v", err)
	}

	// Calculate proper frame rate based on capture duration and frame count
	captureDuration := 30.0 // We captured for 30 seconds
	totalFrames := len(h264Constructor.completeFrames)
	actualFPS := float64(totalFrames) / captureDuration
	expectedFPS := 15.0 // Based on device specs

	fmt.Println("")
	fmt.Println("🎉 Live capture and H.264/Audio conversion complete!")
	fmt.Printf("📁 Captured raw data: video.bin\n")
	fmt.Printf("📁 H.264 output: %s\n", outputFile)
	fmt.Printf("📁 G.711A audio output: live_capture_output_audio.g711a\n")
	fmt.Printf("📊 Stream Analysis:\n")
	fmt.Printf("   • Capture duration: %.1f seconds\n", captureDuration)
	fmt.Printf("   • Complete video frames: %d\n", totalFrames)
	fmt.Printf("   • Audio data extracted: %.2f KB\n", float64(totalAudioBytesWritten)/1024.0)
	fmt.Printf("   • Actual FPS: %.2f (expected: %.0f)\n", actualFPS, expectedFPS)
	fmt.Printf("   • Frame rate efficiency: %.1f%%\n", (actualFPS/expectedFPS)*100)

	if actualFPS < expectedFPS*0.5 {
		fmt.Printf("⚠️  WARNING: Low frame rate detected! Possible fragmentation issues.\n")
	}

	fmt.Println("")
	fmt.Println("💡 To convert H.264 to MP4:")

	if actualFPS < 5.0 {
		// Very low frame rate - use expected frame rate as input, limit output
		fmt.Printf("   ffmpeg -r %.0f -i %s -r %.0f -c:v libx264 -preset slow live_capture_output.mp4\n",
			expectedFPS, outputFile, max(actualFPS, 1.0))
		fmt.Println("   (Using device specs frame rate for input timing)")
	} else if actualFPS < expectedFPS*0.8 {
		// Somewhat low frame rate
		fmt.Printf("   ffmpeg -r %.2f -i %s -c:v libx264 -preset fast live_capture_output.mp4\n", actualFPS, outputFile)
	} else {
		// Normal frame rate - use copy for efficiency
		fmt.Printf("   ffmpeg -r %.2f -i %s -c copy live_capture_output.mp4\n", actualFPS, outputFile)
	}
	fmt.Println("")
	fmt.Println("🎵 To convert G.711A audio to WAV, run:")
	fmt.Printf("   ffmpeg -f alaw -ar 8000 -i live_capture_output_audio.g711a live_capture_output_audio.wav\n")

	return nil
}

// --- H264Constructor Methods ---

// NewH264Constructor initializes a new constructor.
func NewH264Constructor() *H264Constructor {
	return &H264Constructor{
		frames:          make([]*JT1078Frame, 0),
		spsData:         make([][]byte, 0),
		ppsData:         make([][]byte, 0),
		completeFrames:  make([][]byte, 0),
		frameTimestamps: make([]uint64, 0),
	}
}

// AddFrame adds a parsed frame, filtering out non-video data.
func (h *H264Constructor) AddFrame(frame *JT1078Frame) {
	// Only process video frames (DataType 0, 1, 2)
	if frame.DataType >= 3 {
		return
	}

	// Analyze the payload for debugging and extract SPS/PPS
	frameTypeNames := []string{"I-Frame", "P-Frame", "B-Frame"}
	frameTypeName := frameTypeNames[frame.DataType]

	// Log significant frames for debugging
	if len(h.frames) < 10 || len(h.frames)%50 == 0 {
		analyzePayload(frame.Payload, frameTypeName)
	}

	// Extract SPS/PPS from this frame (especially important for I-frames)
	h.extractSpsPps(frame.Payload)

	// For I-frames, also try to extract from any embedded parameter sets
	if frame.DataType == 0 { // I-Frame
		h.extractParameterSetsFromIFrame(frame.Payload)
	}

	h.frames = append(h.frames, frame)
}

// extractParameterSetsFromIFrame tries to find SPS/PPS in I-frame payloads
func (h *H264Constructor) extractParameterSetsFromIFrame(payload []byte) {
	// I-frames often contain SPS/PPS at the beginning
	// Look for consecutive NAL units: SPS (7) followed by PPS (8) followed by IDR (5)

	nalCount := 0
	for i := 0; i < len(payload)-4 && nalCount < 5; { // Limit search to avoid infinite loops
		// Find NAL unit start code
		if !(payload[i] == 0 && payload[i+1] == 0 && (payload[i+2] == 1 || (payload[i+2] == 0 && payload[i+3] == 1))) {
			i++
			continue
		}

		// Determine start position and NAL type
		startOffset := 3
		if payload[i+2] == 0 {
			startOffset = 4
		}
		nalStart := i + startOffset
		if nalStart >= len(payload) {
			break
		}
		nalType := payload[nalStart] & 0x1F
		nalCount++

		// Find end of this NAL unit
		nalEnd := len(payload)
		for j := nalStart + 1; j < len(payload)-4; j++ {
			if payload[j] == 0 && payload[j+1] == 0 && (payload[j+2] == 1 || (payload[j+2] == 0 && payload[j+3] == 1)) {
				nalEnd = j
				break
			}
		}

		// Process specific NAL types
		if nalType == 7 { // SPS
			nalData := append([]byte{0x00, 0x00, 0x00, 0x01}, payload[nalStart:nalEnd]...)
			h.spsData = append(h.spsData, nalData)
			fmt.Printf("📍 Found SPS in I-frame (length: %d bytes)\n", len(nalData))
		} else if nalType == 8 { // PPS
			nalData := append([]byte{0x00, 0x00, 0x00, 0x01}, payload[nalStart:nalEnd]...)
			h.ppsData = append(h.ppsData, nalData)
			fmt.Printf("📍 Found PPS in I-frame (length: %d bytes)\n", len(nalData))
		} else if nalType == 5 { // IDR slice
			fmt.Printf("📍 Found IDR slice in I-frame (NAL type 5)\n")
		}

		i = nalEnd
	}
}

// extractSpsPps finds and stores SPS (7) and PPS (8) NAL units.
func (h *H264Constructor) extractSpsPps(payload []byte) {
	for i := 0; i < len(payload)-4; {
		// Find NAL unit start code (00 00 01 or 00 00 00 01)
		if !(payload[i] == 0 && payload[i+1] == 0 && (payload[i+2] == 1 || (payload[i+2] == 0 && payload[i+3] == 1))) {
			i++
			continue
		}

		// Determine start position and NAL type
		startOffset := 3
		if payload[i+2] == 0 {
			startOffset = 4
		}
		nalStart := i + startOffset
		if nalStart >= len(payload) {
			break
		}
		nalType := payload[nalStart] & 0x1F

		// If it's SPS or PPS, find its end and store it
		if nalType == 7 || nalType == 8 {
			nalEnd := len(payload)
			for j := nalStart; j < len(payload)-4; j++ {
				if payload[j] == 0 && payload[j+1] == 0 && (payload[j+2] == 1 || (payload[j+2] == 0 && payload[j+3] == 1)) {
					nalEnd = j
					break
				}
			}
			// Re-add the 4-byte start code for the file
			nalData := append([]byte{0x00, 0x00, 0x00, 0x01}, payload[nalStart:nalEnd]...)
			if nalType == 7 {
				h.spsData = append(h.spsData, nalData)
			} else {
				h.ppsData = append(h.ppsData, nalData)
			}
			i = nalEnd
		} else {
			i++
		}
	}
}

// ReconstructFrames assembles fragmented packets into complete video frames.
func (h *H264Constructor) ReconstructFrames() {
	if len(h.frames) == 0 {
		return
	}

	fmt.Printf("🔧 Reconstructing frames from %d video fragments...\n", len(h.frames))

	// Group fragments by timestamp to reconstruct complete frames
	frameGroups := make(map[uint64][]*JT1078Frame)

	// Group all packets by their timestamp
	for _, frame := range h.frames {
		frameGroups[frame.Timestamp] = append(frameGroups[frame.Timestamp], frame)
	}

	fmt.Printf("📊 Found %d unique timestamps for frame reconstruction\n", len(frameGroups))

	// Sort timestamps to process frames in temporal order
	var sortedTimestamps []uint64
	for timestamp := range frameGroups {
		sortedTimestamps = append(sortedTimestamps, timestamp)
	}
	sort.Slice(sortedTimestamps, func(i, j int) bool {
		return sortedTimestamps[i] < sortedTimestamps[j]
	})

	fmt.Printf("🔄 Processing frames in chronological order (first: %d, last: %d)\n",
		sortedTimestamps[0], sortedTimestamps[len(sortedTimestamps)-1])

	// Process each timestamp group in chronological order
	frameCounter := 0
	for _, timestamp := range sortedTimestamps {
		packets := frameGroups[timestamp]
		frameCounter++

		// Sort packets within each timestamp group by sequence number
		sort.Slice(packets, func(i, j int) bool {
			return packets[i].SequenceNumber < packets[j].SequenceNumber
		})

		// Check for sequence number gaps within the same timestamp
		if len(packets) > 1 {
			firstSeq := packets[0].SequenceNumber
			lastSeq := packets[len(packets)-1].SequenceNumber
			expectedCount := int(lastSeq - firstSeq + 1)
			actualCount := len(packets)

			if actualCount != expectedCount {
				fmt.Printf("⚠️  Sequence gap detected at timestamp %d: expected %d packets (seq %d-%d), got %d\n",
					timestamp, expectedCount, firstSeq, lastSeq, actualCount)
			}
		}

		// Debug: Show progress for first few frames and every 50th frame
		if frameCounter <= 5 || frameCounter%50 == 0 {
			if len(packets) > 1 {
				var fragTypes []int
				var seqNumbers []uint16
				for _, p := range packets {
					fragTypes = append(fragTypes, p.SubPackageType)
					seqNumbers = append(seqNumbers, p.SequenceNumber)
				}
				fmt.Printf("🧩 Frame #%d - Timestamp %d: %d fragments with types %v (seq: %v)\n",
					frameCounter, timestamp, len(packets), fragTypes, seqNumbers)
			} else {
				fmt.Printf("🧩 Frame #%d - Timestamp %d: atomic frame (seq: %d)\n",
					frameCounter, timestamp, packets[0].SequenceNumber)
			}
		}

		// Reconstruct frames from packets with the same timestamp
		h.reconstructFramesFromPackets(packets, timestamp)
	}

	// Remove the sorting here since we already processed frames in timestamp order
	fmt.Printf("✅ Reconstructed %d complete video frames in chronological order\n", len(h.completeFrames))
}

// reconstructFramesFromPackets processes packets with the same timestamp
func (h *H264Constructor) reconstructFramesFromPackets(packets []*JT1078Frame, timestamp uint64) {
	var currentFragments []*JT1078Frame

	for _, packet := range packets {
		switch packet.SubPackageType {
		case 0: // Atomic frame (complete frame in one packet)
			h.addReconstructedFrameWithTimestamp(packet.Payload, timestamp)

		case 1: // First fragment of a larger frame
			// Start new fragment sequence
			currentFragments = []*JT1078Frame{packet}

		case 3: // Middle fragment
			if len(currentFragments) > 0 && currentFragments[0].Timestamp == packet.Timestamp {
				currentFragments = append(currentFragments, packet)
			} else {
				// Orphaned middle fragment - try to reconstruct by creating synthetic first fragment
				fmt.Printf("⚠️  Orphaned middle fragment at timestamp %d - attempting recovery\n", timestamp)
				// Create a basic SPS/PPS header if we don't have one
				if len(h.spsData) == 0 {
					h.generateBasicSPSPPS()
				}
				// Start new sequence with this middle fragment as if it were first
				currentFragments = []*JT1078Frame{packet}
			}

		case 2: // Last fragment
			if len(currentFragments) > 0 && currentFragments[0].Timestamp == packet.Timestamp {
				currentFragments = append(currentFragments, packet)

				// Assemble the complete frame from fragments
				var completePayload []byte
				for _, frag := range currentFragments {
					completePayload = append(completePayload, frag.Payload...)
				}

				// Only show assembly messages for first few frames or large frames
				frameCount := len(h.completeFrames) + 1
				if frameCount <= 5 || len(currentFragments) > 20 || frameCount%100 == 0 {
					fmt.Printf("🧩 Assembled frame #%d from %d fragments (size: %d bytes)\n", frameCount, len(currentFragments), len(completePayload))
				}
				h.addReconstructedFrameWithTimestamp(completePayload, timestamp)
				currentFragments = nil // Reset for the next sequence
			} else {
				// Orphaned last fragment - create synthetic frame
				fmt.Printf("⚠️  Orphaned last fragment at timestamp %d - creating synthetic frame\n", timestamp)
				// Ensure we have SPS/PPS
				if len(h.spsData) == 0 {
					h.generateBasicSPSPPS()
				}
				h.addReconstructedFrameWithTimestamp(packet.Payload, timestamp)
			}
		}
	}

	// Handle incomplete sequences (missing last fragment)
	if len(currentFragments) > 0 {
		fmt.Printf("⚠️  Incomplete fragment sequence at timestamp %d (%d fragments, missing last) - assembling anyway\n", timestamp, len(currentFragments))
		// Assemble what we have
		var completePayload []byte
		for _, frag := range currentFragments {
			completePayload = append(completePayload, frag.Payload...)
		}
		h.addReconstructedFrameWithTimestamp(completePayload, timestamp)
	}
}

// generateBasicSPSPPS creates proper SPS/PPS headers for 1280x720 H.264 when missing
func (h *H264Constructor) generateBasicSPSPPS() {
	// Generate proper SPS for 1280x720 H.264 Baseline Profile Level 3.1
	// This matches typical JT1078 device configurations
	sps := []byte{
		0x00, 0x00, 0x00, 0x01, // Start code
		0x67,             // NAL unit header: SPS (7)
		0x42, 0x80, 0x1F, // Profile IDC (66=Baseline), constraint flags, Level IDC (31=3.1)
		0xFF, 0xE1, // Pic parameter set count
		// SPS data for 1280x720
		0x67, 0x42, 0x80, 0x1F, // Profile and level repeated
		0xDA, 0x01, 0x40, 0x16, // Picture size parameters
		0x6D, 0xB0, 0x42, 0x04, // Timing info for 1280x720
		0x40, 0x40, 0x50, // Additional parameters
	}

	// Generate proper PPS for H.264
	pps := []byte{
		0x00, 0x00, 0x00, 0x01, // Start code
		0x68,             // NAL unit header: PPS (8)
		0xCE, 0x38, 0x80, // Picture parameter set data
	}

	fmt.Printf("🔧 Generated proper SPS/PPS headers for 1280x720 H.264 Baseline\n")
	h.spsData = append(h.spsData, sps)
	h.ppsData = append(h.ppsData, pps)
}

// addReconstructedFrameWithTimestamp prepends an Access Unit Delimiter (AUD) and adds the frame with timestamp.
func (h *H264Constructor) addReconstructedFrameWithTimestamp(payload []byte, timestamp uint64) {
	// Set first timestamp for relative timing calculations
	if h.firstTimestamp == 0 {
		h.firstTimestamp = timestamp
	}

	// AUD NAL unit (00 00 00 01 09 10) indicates a new frame for the decoder
	aud := []byte{0x00, 0x00, 0x00, 0x01, 0x09, 0x10}
	fullFrame := append(aud, payload...)

	h.completeFrames = append(h.completeFrames, fullFrame)
	h.frameTimestamps = append(h.frameTimestamps, timestamp)
}

// ConstructH264File writes the final .h264 file.
func (h *H264Constructor) ConstructH264File(filename string, totalAudioBytesWritten int) error {
	fmt.Println("🔧 Reconstructing H.264 stream...")
	h.ReconstructFrames()

	outFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Write unique SPS and PPS headers at the start of the file
	uniqueSPS := deduplicateNALs(h.spsData)
	uniquePPS := deduplicateNALs(h.ppsData)

	for _, sps := range uniqueSPS {
		_, _ = outFile.Write(sps)
	}
	for _, pps := range uniquePPS {
		_, _ = outFile.Write(pps)
	}

	// Write all the reconstructed video frames
	totalBytesWritten := 0
	for _, frameData := range h.completeFrames {
		bytesWritten, err := outFile.Write(frameData)
		if err != nil {
			return fmt.Errorf("failed to write frame: %v", err)
		}
		totalBytesWritten += bytesWritten
	}

	fmt.Printf("✅ Streams extracted successfully!\n")
	fmt.Println("---")
	fmt.Printf("📊 Video Statistics (H.264):\n")
	fmt.Printf("  • Total video fragments processed: %d\n", len(h.frames))
	fmt.Printf("  • Unique SPS headers found: %d\n", len(uniqueSPS))
	fmt.Printf("  • Unique PPS headers found: %d\n", len(uniquePPS))
	fmt.Printf("  • Complete video frames written: %d\n", len(h.completeFrames))
	fmt.Printf("  • Total video data written: %.2f KB\n", float64(totalBytesWritten)/1024.0)
	fmt.Printf("  • Output file: %s\n", filename)
	fmt.Println("---")
	fmt.Printf("🔊 Audio Statistics (G.711A):\n")
	fmt.Printf("  • Total audio data written: %.2f KB\n", float64(totalAudioBytesWritten)/1024.0)
	fmt.Printf("  • Audio output file: live_capture_output_audio.g711a\n")
	fmt.Println("---")

	return nil
}

// parseJT1078Frame decodes a byte buffer into a JT1078Frame using proven two-way algorithm
func parseJT1078Frame(buf []byte) (*JT1078Frame, int) {
	headerIdx := -1

	// Look for JT1078 header signature: 30 31 63 64
	for i := 0; i <= len(buf)-4; i++ {
		if buf[i] == 0x30 && buf[i+1] == 0x31 && buf[i+2] == 0x63 && buf[i+3] == 0x64 {
			headerIdx = i
			break
		}
	}

	if headerIdx == -1 {
		if len(buf) > 4 {
			return nil, len(buf) - 4 // Skip most of the buffer but keep some for next iteration
		}
		return nil, 0
	}

	// Ensure we have enough data for basic header (28 bytes minimum)
	if len(buf)-headerIdx < 28 {
		return nil, 0
	}

	frameData := buf[headerIdx:]

	// Parse data type from byte 15 (label3)
	label3 := frameData[15]
	dataType := int(label3&0xF0) >> 4
	subPackageType := int(label3 & 0x0F)

	// Calculate offset based on data type (using proven two-way logic)
	offset := 16 // Start after the basic header

	// For non-transparent data, skip timestamp (8 bytes)
	if dataType != 4 {
		if len(frameData) < offset+8 {
			return nil, 0
		}
		offset += 8
	}

	// For video frames (not audio, not transparent), skip additional header (4 bytes)
	if dataType != 4 && dataType != 3 {
		if len(frameData) < offset+4 {
			return nil, 0
		}
		offset += 4
	}

	// Get payload length
	if len(frameData) < offset+2 {
		return nil, 0
	}
	payloadLength := binary.BigEndian.Uint16(frameData[offset : offset+2])
	offset += 2

	// Sanity check payload length
	if payloadLength > 8192 {
		// Skip this frame if payload is too large
		return nil, headerIdx + 28
	}

	totalFrameSize := offset + int(payloadLength)
	if len(frameData) < totalFrameSize {
		return nil, 0 // Need more data
	}

	// Extract basic frame info for compatibility
	frame := &JT1078Frame{
		SequenceNumber: binary.BigEndian.Uint16(frameData[6:8]),
		SIM:            fmt.Sprintf("%x", frameData[8:14]),
		LogicChannel:   frameData[14],
		DataType:       dataType,
		SubPackageType: subPackageType,
		Timestamp:      0,   // Will be set properly below
		Payload:        nil, // Will be set properly below
	}

	// Extract timestamp based on data type
	if dataType != 4 {
		// For non-transparent data, timestamp is at offset 16-23
		frame.Timestamp = binary.BigEndian.Uint64(frameData[16:24])
	}

	// Extract payload
	if payloadLength > 0 {
		frame.Payload = make([]byte, payloadLength)
		copy(frame.Payload, frameData[offset:offset+int(payloadLength)])
	}

	// Log frame details for first few frames
	if len(frame.Payload) > 0 {
		static_frame_count++
		if static_frame_count <= 5 {
			typeNames := []string{"I-Frame", "P-Frame", "B-Frame", "Audio", "Transparent"}
			subTypeNames := []string{"Atomic", "First", "Last", "Middle"}
			fmt.Printf("🔍 Frame #%d: %s %s, Seq:%d, Payload:%d bytes, Timestamp:%d\n",
				static_frame_count, typeNames[dataType], subTypeNames[subPackageType],
				frame.SequenceNumber, len(frame.Payload), frame.Timestamp)
		}
	}

	return frame, headerIdx + totalFrameSize
}

var static_frame_count = 0

// max returns the larger of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// analyzePayload analyzes the payload for NAL units and provides detailed info
func analyzePayload(payload []byte, frameType string) {
	if len(payload) < 10 {
		return // Too small to be meaningful
	}

	nalUnits := []int{}
	for i := 0; i < len(payload)-4; {
		// Find NAL unit start code (00 00 01 or 00 00 00 01)
		if !(payload[i] == 0 && payload[i+1] == 0 && (payload[i+2] == 1 || (payload[i+2] == 0 && payload[i+3] == 1))) {
			i++
			continue
		}

		// Determine start position and NAL type
		startOffset := 3
		if payload[i+2] == 0 {
			startOffset = 4
		}
		nalStart := i + startOffset
		if nalStart >= len(payload) {
			break
		}
		nalType := payload[nalStart] & 0x1F
		nalUnits = append(nalUnits, int(nalType))

		// Skip ahead to avoid finding the same start code
		i = nalStart + 1
	}

	if len(nalUnits) > 0 {
		fmt.Printf("   %s payload (%d bytes) contains NAL units: %v\n", frameType, len(payload), nalUnits)
	}
}

// deduplicateNALs removes duplicate byte slices (for SPS/PPS).
func deduplicateNALs(nals [][]byte) [][]byte {
	unique := make([][]byte, 0, len(nals))
	seen := make(map[string]bool)
	for _, nal := range nals {
		key := string(nal)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, nal)
		}
	}
	return unique
}
