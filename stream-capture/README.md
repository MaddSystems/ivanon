# JT/T 1078 Protocol Technical Specification & Decoding Manual

## Overview

This document provides a comprehensive technical specification of the JT/T 1078 protocol structure, frame decoding methodology, and stream reconstruction algorithms. This protocol is the Chinese national standard for vehicle video transmission systems.

## JT/T 1078 Protocol Architecture

### Transport Layer
- **Protocol**: TCP-based streaming
- **Default Port**: 7800
- **Connection**: Persistent TCP connection with real-time data flow
- **Byte Order**: Big-endian (network byte order)

### Frame Structure Foundation

Every JT1078 frame follows this fundamental structure:

```
┌─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│   Fixed Header  │  Timestamp      │  Video Header   │    Payload      │
│    (16 bytes)   │   (8 bytes)     │   (4 bytes)     │   (Variable)    │
└─────────────────┴─────────────────┴─────────────────┴─────────────────┘
```

**Note**: Header sections are conditional based on data type - this is the key to proper parsing.

## JT/T 1078 Frame Header Specification

### Fixed Header Structure (16 bytes)

```
Offset | Size | Field Name        | Description                    | Example Value
-------|------|-------------------|--------------------------------|---------------
0x00   | 4    | Header Signature  | Fixed: 30 31 63 64             | 0x30316364
0x04   | 1    | V+P+X+CC          | Version + Padding + Ext + CC   | 0x80
0x05   | 1    | M+PT              | Marker + Payload Type          | 0x60
0x06   | 2    | Sequence Number   | Incremental packet sequence    | 0x0001-0xFFFF
0x08   | 6    | SIM Card Number   | Device SIM identifier (BCD)    | Variable
0x0E   | 1    | Logic Channel     | Camera channel number          | 0x01
0x0F   | 1    | Label3 (Critical) | Data Type + SubPackage Type    | See below
```

### Label3 Field Decoding (Byte 0x0F) - THE KEY

This single byte determines the entire frame structure:

```
Bit Position: 7 6 5 4 | 3 2 1 0
             --------- ---------
             Data Type | SubPkg Type
```

#### Data Type (Upper 4 bits):
```
0x0 (0000) = I-Frame (Intra-frame, keyframe)
0x1 (0001) = P-Frame (Predicted frame)  
0x2 (0010) = B-Frame (Bidirectional frame)
0x3 (0011) = Audio Frame (G.711A/G.726/etc)
0x4 (0100) = Transparent Data (passthrough)
```

#### SubPackage Type (Lower 4 bits):
```
0x0 (0000) = Atomic (complete frame in single packet)
0x1 (0001) = First (first fragment of multi-packet frame)
0x2 (0010) = Last (final fragment of multi-packet frame)
0x3 (0011) = Middle (intermediate fragment)
```

**Examples**:
- `0x00` = I-Frame, Atomic (complete I-frame in one packet)
- `0x01` = I-Frame, First fragment
- `0x13` = P-Frame, Middle fragment  
- `0x32` = Audio, Last fragment

## Dynamic Header Structure Based on Data Type

### Rule 1: Timestamp Section (8 bytes)
```
IF (DataType != 0x4) {  // Not transparent data
    Timestamp Present: YES (8 bytes at offset 0x10)
} ELSE {
    Timestamp Present: NO (skip to payload length)
}
```

### Rule 2: Video Header Section (4 bytes)
```
IF (DataType != 0x4 AND DataType != 0x3) {  // Not transparent AND not audio
    Video Header Present: YES (4 bytes after timestamp)
} ELSE {
    Video Header Present: NO (skip to payload length)
}
```

### Complete Frame Layouts

#### Video Frame (I/P/B-Frame) Structure:
```
┌──────────────────┬──────────────────┬──────────────────┬──────────────────┬──────────────────┐
│  Fixed Header    │    Timestamp     │   Video Header   │  Payload Length  │     Payload      │
│   (16 bytes)     │    (8 bytes)     │    (4 bytes)     │    (2 bytes)     │   (Variable)     │
│   0x00-0x0F      │   0x10-0x17      │   0x18-0x1B      │   0x1C-0x1D      │   0x1E onwards   │
└──────────────────┴──────────────────┴──────────────────┴──────────────────┴──────────────────┘
Total Header Size: 30 bytes
```

#### Audio Frame Structure:
```
┌──────────────────┬──────────────────┬──────────────────┬──────────────────┐
│  Fixed Header    │    Timestamp     │  Payload Length  │     Payload      │
│   (16 bytes)     │    (8 bytes)     │    (2 bytes)     │   (Variable)     │
│   0x00-0x0F      │   0x10-0x17      │   0x18-0x19      │   0x1A onwards   │
└──────────────────┴──────────────────┴──────────────────┴──────────────────┘
Total Header Size: 26 bytes
```

#### Transparent Data Structure:
```
┌──────────────────┬──────────────────┬──────────────────┐
│  Fixed Header    │  Payload Length  │     Payload      │
│   (16 bytes)     │    (2 bytes)     │   (Variable)     │
│   0x00-0x0F      │   0x10-0x11      │   0x12 onwards   │
└──────────────────┴──────────────────┴──────────────────┘
Total Header Size: 18 bytes
```

## Offset Calculation Algorithm

The critical algorithm for parsing JT1078 frames:

```go
func calculatePayloadOffset(dataType int) int {
    offset := 16  // Start after fixed header
    
    // Step 1: Add timestamp if not transparent data
    if dataType != 4 {
        offset += 8  // Timestamp: 8 bytes
    }
    
    // Step 2: Add video header if video frame
    if dataType != 4 && dataType != 3 {  // Not transparent AND not audio
        offset += 4  // Video header: 4 bytes  
    }
    
    return offset  // Payload length starts here
}
```

**Offset Results**:
- Video frames (I/P/B): `16 + 8 + 4 = 28 bytes` → Payload length at 0x1C
- Audio frames: `16 + 8 = 24 bytes` → Payload length at 0x18  
- Transparent data: `16 bytes` → Payload length at 0x10

## Timestamp Structure (8 bytes)

When present, timestamp follows this format:

```
┌────────────────────────────────────────────────────────────────┐
│                    64-bit Timestamp (Big-endian)               │
│                        Units: Milliseconds                     │
│                    Base: Unix epoch or device boot             │
└────────────────────────────────────────────────────────────────┘
```

**Usage**: Critical for chronological frame ordering and synchronization.

## Video Header Structure (4 bytes)

Present only for video frames (I/P/B):

```
Offset | Size | Field          | Description
-------|------|----------------|----------------------------------
0x00   | 2    | Width          | Video width in pixels (1280)
0x02   | 2    | Height         | Video height in pixels (720)
```

## Frame Fragmentation System

### Fragmentation Logic

Large frames (especially I-frames) are split across multiple packets:

1. **MTU Consideration**: Each packet typically ≤ 1024-1500 bytes
2. **Fragment Sequence**: `[First] → [Middle...] → [Last]`
3. **Atomic Frames**: Small frames sent in single packet

### Fragment Identification

Each fragment shares:
- **Same Timestamp**: All fragments of one frame have identical timestamp
- **Sequential Numbers**: Continuous sequence numbers within timestamp group
- **SubPackage Progression**: `1 → 3 → 3 → ... → 3 → 2`

### Typical Fragmentation Patterns

```
I-Frame (Large, ~30KB):  [1, 3, 3, 3, ..., 3, 2]  (30+ fragments)
P-Frame (Medium, ~3KB):  [1, 3, 2]                 (3 fragments)
B-Frame (Small, ~2KB):   [1, 2]                    (2 fragments)
Audio (Small, ~256B):    [0]                       (atomic)
```

## Stream Reconstruction Algorithm

### Phase 1: Frame Detection & Parsing

```go
1. Scan for header signature: 0x30316364
2. Extract Label3 byte → decode DataType + SubPackageType  
3. Calculate dynamic offset based on DataType
4. Extract payload length (2 bytes, big-endian)
5. Validate payload size (0 < size ≤ 8192)
6. Extract complete frame data
```

### Phase 2: Fragment Grouping

```go
1. Group fragments by timestamp (identical timestamps = same frame)
2. Sort fragments within group by sequence number
3. Validate fragment sequence: detect gaps in sequence numbers
4. Check SubPackage progression: 1 → 3* → 2 (or 0 for atomic)
```

### Phase 3: Chronological Ordering

**CRITICAL**: Process timestamp groups in chronological order:

```go
1. Extract all unique timestamps from fragment groups
2. Sort timestamps numerically (ascending)
3. Process each timestamp group in chronological sequence
4. Reconstruct frames in temporal order
```

This ensures proper video playback without frame sequence corruption.

### Phase 4: Frame Assembly

```go
1. For each timestamp group (in chronological order):
   a. Concatenate fragment payloads in sequence order
   b. Prepend Access Unit Delimiter (AUD) for H.264 compatibility
   c. Add to final stream in temporal sequence
```

## H.264 Payload Structure

### NAL Unit Detection

H.264 payload contains NAL units with start codes:

```
Start Code Pattern: 0x00 0x00 0x00 0x01 (4-byte)
                or: 0x00 0x00 0x01     (3-byte)
```

### NAL Unit Types (Byte after start code & 0x1F):

```
Type | Description                | Importance
-----|----------------------------|---------------------------
1    | Coded slice (P-frame)      | Video data
2    | Coded slice (B-frame)      | Video data  
5    | IDR slice (I-frame)        | Keyframe data
7    | SPS (Sequence Parameter)   | CRITICAL for decoding
8    | PPS (Picture Parameter)    | CRITICAL for decoding
9    | Access Unit Delimiter      | Frame boundary marker
```

### Parameter Set Extraction

SPS/PPS are found in:
1. **First fragments** of I-frames (most common)
2. **Standalone parameter packets** (less common)
3. **Generated synthetically** if missing (fallback)

## G.711A Audio Structure

Audio frames contain raw G.711A (A-law) encoded data:

```
┌─────────────────────────────────────────────────────────────┐
│                    Raw G.711A Audio Data                    │
│                                                             │
│   • Sample Rate: 8000 Hz                                    │
│   • Encoding: A-law companding                              │
│   • Frame Size: ~160-320 bytes typical                      │
│   • No additional headers or structure                      │
└─────────────────────────────────────────────────────────────┘
```

**Processing**: Audio payloads are concatenated directly to output file.

## Practical Implementation Notes

### Error Handling
1. **Invalid signatures**: Skip byte-by-byte until valid header found
2. **Oversized payloads**: Reject frames with payload > 8192 bytes
3. **Missing fragments**: Assemble partial frames when possible
4. **Sequence gaps**: Report but continue processing

### Performance Optimizations
1. **Signature scanning**: Use Boyer-Moore or similar for fast header detection
2. **Memory management**: Stream processing without full buffering
3. **Fragment caching**: Group fragments efficiently by timestamp

### Validation Checks
1. **Payload length bounds**: 0 ≤ length ≤ 8192
2. **Header signature**: Must be exactly 0x30316364
3. **Sequence continuity**: Gap detection within timestamp groups
4. **Timestamp monotonicity**: Generally increasing (with tolerance for reordering)

## Protocol Constants & Limits

```go
const (
    JT1078_SIGNATURE     = 0x30316364
    FIXED_HEADER_SIZE    = 16
    TIMESTAMP_SIZE       = 8  
    VIDEO_HEADER_SIZE    = 4
    PAYLOAD_LENGTH_SIZE  = 2
    MAX_PAYLOAD_SIZE     = 8192
    
    DATA_TYPE_I_FRAME    = 0
    DATA_TYPE_P_FRAME    = 1  
    DATA_TYPE_B_FRAME    = 2
    DATA_TYPE_AUDIO      = 3
    DATA_TYPE_TRANSPARENT = 4
    
    SUBPKG_ATOMIC        = 0
    SUBPKG_FIRST         = 1
    SUBPKG_LAST          = 2  
    SUBPKG_MIDDLE        = 3
)
```

This specification provides the complete key to decode JT/T 1078 protocol streams with perfect accuracy and efficiency.
