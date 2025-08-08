# JT1078 Header Layout

The JT1078 protocol, used for vehicle-mounted video surveillance systems in China, defines a packet header structure for real-time audio and video transmission. The following table outlines the header fields, their sizes, and descriptions, based on documentation from the SmallChi/JT1078 GitHub repository.

| Field Name                  | Size (bytes) | Description                                   |
|-----------------------------|--------------|-----------------------------------------------|
| FH_Flag                    | 4            | Frame header identifier (e.g., "30 31 63 64") |
| Label1                     | 1            | RTP protocol version (V), padding (P), extension (X), CSRC counter (CC) |
| Label2                     | 1            | Marker bit (M), payload type (PT)             |
| SN (Packet Sequence)       | 2            | Packet sequence number                        |
| SIM (SIM Card Number)      | 6            | SIM card number                               |
| LogicChannelNumber         | 1            | Logical channel number                        |
| Label3                     | 1            | Data type (4 bits), subpackage type (4 bits)  |
| Timestamp                  | 8            | Timestamp                                     |
| LastIFrameInterval         | 2            | Last I frame interval                         |
| LastFrameInterval          | 2            | Last frame interval                           |
| DataBodyLength             | 2            | Length of data body                           |
| Bodies (Data Body)         | Variable     | Data body (e.g., H.264 encoded data)          |

## Explanation of Fields
- **FH_Flag**: A 4-byte fixed identifier, typically "30 31 63 64" in hexadecimal, marking the start of a JT1078 packet.
- **Label1**: A 1-byte field containing RTP header information: version (V, 2 bits, typically 2), padding (P, 1 bit, typically 0), extension (X, 1 bit, typically 0), and CSRC count (CC, 4 bits, typically 1).
- **Label2**: A 1-byte field with the marker bit (M, 1 bit) and payload type (PT, 7 bits, e.g., 114 for H.264 video).
- **SN (Packet Sequence)**: A 2-byte field for packet sequencing, crucial for handling fragmented data.
- **SIM (SIM Card Number)**: A 6-byte field for the vehicle's SIM card number, used for device identification.
- **LogicChannelNumber**: A 1-byte field indicating the logical channel (e.g., video or audio).
- **Label3**: A 1-byte field with data type (4 bits, e.g., video or audio) and subpackage type (4 bits, indicating fragmentation: first, middle, last).
- **Timestamp**: An 8-byte field for timing synchronization, typically in milliseconds since the epoch.
- **LastIFrameInterval**: A 2-byte field indicating the interval of the last I-frame, used in video encoding.
- **LastFrameInterval**: A 2-byte field for the interval of the last frame, also used in video encoding.
- **DataBodyLength**: A 2-byte field specifying the length of the payload (Bodies).
- **Bodies (Data Body)**: The variable-length payload, containing the actual data, such as H.264 video or PCMA audio.

## Notes
- The total header size, excluding the payload, is 29 bytes.
- This layout is based on the JT/T 1078-2016 standard and incorporates RTP protocol elements for real-time audio/video transmission.
- The header is sufficient for parsing JT1078 packets, enabling the splitting and reassembly of video and audio