GNSS system for operational vehicles\-\-\-\--General

specifications for vehicle terminal communication protocol and data
format

1.  Range

This standard specifies the communication protocol and data format
between GNSS system vehicle-mounted terminal (hereinafter referred to as
terminal) and supervision/monitoring platform (hereinafter referred to
as platform), including protocol basis, communication connection,
message processing, protocol classification and description,as well as
data format.

This protocol applies to the communication between the vehicle terminal
device and the platform of GNSS system.

2.  Terms, Definitions and Abbreviations

    2.1 Terms and Definitions

The below terms and definitions apply to this document.

2.1.1 abnormal data communication link

The wireless communication link is disconnected, or temporarily
suspended (such as during a call)

2.1.2 register

The terminal sends a message to the platform that it is installed on a
certain vehicle.

2.1.3 deregister

The terminal sends a message to the platform that it is removed from the
vehicle.

2.1.4 authentication

When the terminal is connected to the platform, it sends a message to
the platform, so that the platform can verify the identity.

2.1.5 location reporting strategy

Timed, distanced reporting or a combination of both.

2.1.6 location reporting program

Rules for determining the interval for periodic reporting based on
relevant conditions.

2.1.7 additional points report while turning

The terminal sends a position information report message when judging
that the vehicle turns.

2.2 abbreviation

The following abbreviations apply to this document

APN Access Point Name

SMS Short Message Service

TCP Transmission Control Protocol

TTS Text To Speech

UDP User Datagram Protocol

VSS Vehicle Speed Sensor

3.  Protocol Basics

    1.  way of communication

This communication protocol adopts TCP, the platform acts as the server
side, and the terminal acts as the client side.

2.  Data Types

The data types used in the protocol messages are shown in Table 1:

Table1 Data Types

  ---------------- ------------------------------------------------------
  Data Types                      description&requirements

  BYTE                  unsigned single-byte integer (byte, 8 bits)

  WORD                  unsigned double-byte integer (word, 16 bits)

  DWORD              Unsigned four-byte integer (double word, 32 bits)

  BYTE\[n\]                                N byte

  BCD\[n\]                            8421code，n byte

  STRING                  GBK encoding, empty if there is no data
  ---------------- ------------------------------------------------------

3.  Transmission Rules

The protocol uses big-endian network byte order to pass words and double
words.

The agreement is as follows:

\-\-\-\-\-\-\-\-\--Byte (BYTE) transmission convention: according to the
way of byte stream transmission;

\-\-\-\-\-\-\-\-\--Word (WORD) transmission convention: first transmit
the upper eight bits, and then transmit the lower eight bits

\-\-\-\-\-\-\-\-\--Double-byte (DWORD)transmission convention: first
transmit the high 24 bits, then transmit the high 16 bits, then transmit
the high eight bits, and finally transmit the low eight bits.

4.  composition of the message

3.4.1 message structure

Each message consists of a header, a message header, a message body and
a check code. The message structure is shown in Figure 1:

  ---------------- --------------- ----------------- ------------ ----------------
   identification  message header    message body     check code   identification
        bit                                                             bit

  ---------------- --------------- ----------------- ------------ ----------------

Figure 1 message structure

3.4.2identification bit

It is represented by 0x7e. If 0x7e appears in the check code, message
header and message body, it must be transferred meaning. The transferred
meaning rules are defined as follows:

0x7e ←→0x7d followed by a 0x02;

0x7d ←→0x7d followed by a 0x01

The transferred meaning process is as follows:

When sending a message: message encapsulation → computer and fill in
check code →transferred meaning;

When receiving the message: transfer recovery → verify the check code →
parse the message.

For example：

Send a packet with the content of 0x30 0x7e 0x08 0x7d 0x55, then
encapsulate as follows: 0x7e 0x30 0x7d 0x02 0x08 0x7d 0x01 0x55 0x7e.

3.4.3 message header

The content of the message header is shown in Table 2.

Table 2 Contents of message headers

+:-------:+:-------------:+:--------:+:--------------------------------:+
| START   | FIELD         | DATA     | Description                      |
| BYTE    |               |          |                                  |
|         |               | TYPE     |                                  |
+---------+---------------+----------+----------------------------------+
| 0       | Message ID    | WORD     |                                  |
+---------+---------------+----------+----------------------------------+
| 2       | message body  | WORD     | The message body attribute       |
|         | attribute     |          | format structure is shown in     |
|         |               |          | Figure 2                         |
+---------+---------------+----------+----------------------------------+
| 4       | Terminal      | BCD\[6\] | It is converted according to the |
|         | phone number  |          | mobile phone number of the       |
|         |               |          | installation terminal itself. If |
|         |               |          | the mobile phone number is less  |
|         |               |          | than 12 digits, the digits       |
|         |               |          | should be added in front, the    |
|         |               |          | mainland mobile phone number     |
|         |               |          | should be added with the digits  |
|         |               |          | 0, and the digits of Hong Kong,  |
|         |               |          | Macao and Taiwan should be added |
|         |               |          | according to their area codes.   |
+---------+---------------+----------+----------------------------------+
| 10      | message       | WORD     | Circular accumulation starting   |
|         | serial number |          | from 0 according to the sending  |
|         |               |          | order                            |
+---------+---------------+----------+----------------------------------+
| 12      | Message       |          | If the relevant identification   |
|         | Packet        |          | bit in the message body          |
|         | Encapsulation |          | attribute determines the packet  |
|         | Item          |          | processing of the message, the   |
|         |               |          | item has content, otherwise      |
|         |               |          | there is no item                 |
+---------+---------------+----------+----------------------------------+

The message body attribute format structure is shown in Figure 2:

+:----:+:----:+:-----------:+:------:+:------:+:------:+:---------:+:---------:+:---------:+:---------:+:---------:+:---------:+:---------:+:---------:+:---------:+:---------:+
| 15   | 14   | 13          | 12     | 11     | 10     | 9         | 8         | 7         | 6         | 5         | 4         | 3         | 2         | 1         | 0         |
+------+------+-------------+--------+--------+--------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
| reserve     | subcontract | reserve                  | message body length                                                                                                   |
+-------------+-------------+--------------------------+-----------------------------------------------------------------------------------------------------------------------+

Figure 2 Structure diagram of message body attribute format

subcontract：

When the 13th bit in the message body attribute is 1, it means that the
message body is a long message, and the sub-package sending process is
performed. The specific sub-package information is determined by the
message package encapsulation item; if the 13th bit is 0, then there is
no message packet encapsulation item field in the message header.

The contents of the message package encapsulation items are shown in
Table 3.

Table 3 Contents of message package encapsulation items

  ---------- -------------- ----------- ----------------------------------
  START BYTE     FIELD       DATA TYPE     description and requirements

      0          total         WORD      The total quantities of packets
             quantities of              after the message is sub-packaged
                message                 
                packets                 

      2      package serial    WORD                start from 1
                 number                 
  ---------- -------------- ----------- ----------------------------------

3.4.4 check code

Check code means starting from the message header, XOR with the next
byte until the byte before the check code, occupying 1 byte.

4.  Communication connection

    1.  Connection establishment

The daily data connection between the terminal and the platform can use
the TCP method. After the terminal is reset, it should establish a
connection with the platform as soon as possible, and immediately send a
terminal authentication message to the platform for authentication after
the connection is established.

2.  connection maintenance

After the connection is established and the terminal authentication is
successful, the terminal should periodically send a terminal heartbeat
message to the platform. After the platform receives it, it will send a
platform general response message to the terminal. The sending period is
specified by the terminal parameters.

3.  Disconnection

Both the platform and the terminal can actively disconnect according to
the TCP protocol, and both parties should actively determine whether the
TCP connection is disconnected.

The platform determines the method of TCP connection disconnection:

------------According to the TCP protocol, it is judged that the
terminal is actively disconnected;

------------The terminal with the same identity establishes a new
connection, indicating that the original connection has been
disconnected;

------------The message from the terminal is not received within a
certain period of time,such as the heartbeat of the terminal,

The method for the terminal to judge the disconnection of the TCP
connection:

------------According to the TCP protocol, it is judged that the
platform is actively disconnected;

------------the data communication link is disconnected;

------------The data communication link is normal, and no response is
received after the quantities of re-transmissions is reached.

5.  message processing

    5.1　TCP message processing

    5.1.1 Message from the main platform

All messages sent by the platform require the terminal to respond. The
responses are divided into general responses and special responses,
which are determined by each specific functional protocol. After the
sender waits for the response but time out, it should resend the
message. The response timeout time and the quantities of
re-transmissions are specified by platform parameters. The calculation
formula of the response timeout time after each re-transmission is shown
in formula (1).

***TN+1 = TN × ( N + 1 )***

1.  in the formula：

***TN+1 \-\-\-\-\-\-\-\-\-\--***Reply timeout after each
re-transmission；

***TN \-\-\-\-\-\--*** The previous reply timeout

***N*** \-\-\-\-\-\-- Quantities of re-transmissions.

5.1.2 The message sent by the terminal

**5.1.2.1 The data communication link is normal**

When the data communication link is normal, all messages sent by the
terminal require the platform to respond. The responses are divided into
general responses and special responses, which are determined by each
specific functional protocol. After the terminal waits for the response
timeout, it should re-send the message. The response timeout time and
the quantities of re-transmissions are specified by the terminal
parameters, and the response timeout time after each re-transmission is
calculated according to formula (1). For the key alarm message sent by
the terminal, if no response is received after the quantities of
re-transmissions, it should be saved. The saved critical alarm messages
are sent in the future before other messages are sent.

**5.1.2.2 The data communication link is abnormal**

When the data communication link is abnormal, the terminal should save
the location information report message to be sent. Saved messages are
sent as soon as the data communication link returns to normal

6.  Protocol Classification and Description

    1.  Overview

The protocols are described below by functional classification. Unless
otherwise specified, the TCP communication mode is adopted by default.
See Appendix A for the message comparison table of message name and
message ID in the protocol.

2.  Terminal management protocol

    Terminal registration/deregistration

When the terminal is not registered, it must first be registered. After
the registration is successful, the terminal will obtain the
authentication code and save it. The authentication code is used when
the terminal logs in. Before the vehicle needs to remove or replace the
terminal, the terminal should perform a logout operation to cancel the
corresponding relationship between the terminal and the vehicle.

Terminal authentication

After the terminal is registered, it must be authenticated immediately
after establishing a connection with the platform. The terminal shall
not send other messages until the authentication is successful.

The terminal performs authentication by sending a terminal
authentication message, and the platform replies with a platform general
response message.

Set/query terminal parameters

The platform sets terminal parameters by sending a set terminal
parameter message, and the terminal replies with a terminal general
response message. The platform queries terminal parameters by sending a
query terminal parameter message, and the terminal replies with a query
terminal parameter response message. Terminals under different network
standards should support some unique parameters of their respective
networks.

terminal control

The platform controls the terminal by sending terminal control messages,
and the terminal replies to the terminal general response message

3.  Location and alarm protocols

    location information report

The terminal periodically sends the location information report message
according to the parameter setting

According to parameter control, the terminal can send a position
information report message when it judges that the vehicle is turning

Location information query

The platform queries the current location information of the designated
vehicle terminal by sending a location information query message, and
the terminal replies with a location information query response message

Terminal alarm

When the terminal judges that the alarm condition is met, it sends a
location information report message, and sets the corresponding alarm
sign in the location report message. The platform can perform alarm
processing by replying to the platform general response message.

For each alarm type, see the description in the message body of the
location information report. The alarm sign is maintained until the
alarm condition is released. After the alarm condition is released, a
position information report message should be sent immediately to clear
the corresponding alarm sign.

4.  Information protocol

    Text message delivery

The platform sends messages by sending text messages to notify drivers
in a specified way. The terminal replies with a terminal general
response message.

5.  Vehicle Control Protocols

By sending vehicle control messages, the platform requires the terminal
to control the vehicle according to the specified operation. The
terminal will reply the terminal general response message immediately
after receiving it. Afterwards, the terminal controls the vehicle, and
replies to the vehicle control response message according to the result.

6.  Multimedia protocol

    6.6.1 Multimedia event information upload

When the terminal takes the initiative to shoot due to a specific event,
it should actively upload a multimedia event message after the event
occurs, which requires the platform to reply with a general response
message.

6.6.2 Multimedia data upload

The terminal sends a multimedia data upload message to upload the
multimedia data. Each complete multimedia data needs to be attached with
the location information reporting message body during recording, which
is called location multimedia data. The platform determines the
receiving timeout time according to the total quantities of packets.
After receiving all the data packets or reaching the timeout time, the
platform sends a multimedia data upload response message to the
terminal, which confirms the receipt of all data packets or requests the
terminal to re-transmit the specified data packets.

6.6.3 The camera shoots immediately

The platform issues a shooting command to the terminal by sending the
camera immediate shooting command message, which requires the terminal
to reply to the terminal general response message. If real-time upload
is specified, the terminal will upload the camera image/video after
shooting, otherwise, the image/video will be stored.

7.  Generic data transfer classes

Messages that are not defined in the protocol but need to be transmitted
in actual use can use data uplink transparent transmission messages and
data downlink transparent transmission messages for uplink and downlink
data exchange

7.  Data format

    1.  The data format of the terminal general response message body is
        shown in Table 4

Table 4 Data format of terminal general response message body

  ------- ---------- -------- --------------------------------------------
   start    FIELD      DATA           description and requirements
   byte                TYPE   

     0     Response    WORD      The serial number of the corresponding
            serial                          platform message
            number            

     2     Response    WORD   The ID of the corresponding platform message
              ID              

     4      Result     BYTE     0: success/confirmation; 1: failure; 2:
                                    message error; 3: not supported
  ------- ---------- -------- --------------------------------------------

2.  Platform General Response

Message ID:0x8001.

The data format of the general response message body of the platform is
shown in Table 5.

Table 5 Platform general response message body data format

  ------- ---------- -------- -------------------------------------------
   Start    field      Data          description and requirements
   byte                type   

     0     Response    WORD     The serial number of the corresponding
            serial                         terminal message
            number            

     2     Response    WORD      The ID of the corresponding terminal
              ID                                message

     4      Result     BYTE     0: success/confirmation; 1: failure; 2:
                               message error; 3: not supported; 4: alarm
                                        processing confirmation
  ------- ---------- -------- -------------------------------------------

3.  Terminal heartbeat【0002】

Message ID:0x0002

4.  Terminal registration【0100】

Message ID:0x0100

The data format of the terminal registration message body is shown in
Table 6.

Table 6 Terminal registration message body data format

+:-------:+:-----------:+:---------:+:-----------------------------------:+
| Start   | field       | Data      | description and requirements        |
|         |             |           |                                     |
| byte    |             | type      |                                     |
+---------+-------------+-----------+-------------------------------------+
| 0       | ID 1        | WORD      | reserve                             |
+---------+-------------+-----------+-------------------------------------+
| 2       | ID 2        | WORD      | reserve                             |
+---------+-------------+-----------+-------------------------------------+
| 4       | Manufacture | BYTE\[5\] | Five bytes, terminal manufacturer   |
|         | ID          |           | number                              |
+---------+-------------+-----------+-------------------------------------+
| 9       | Terminal    | BYTE\[8\] | Eight bytes, the terminal model is  |
|         | model       |           | defined by the manufacturer, if the |
|         |             |           | number of digits is less than       |
|         |             |           | eight, fill in spaces               |
+---------+-------------+-----------+-------------------------------------+
| 17      | Terminal ID | BYTE\[7\] | Seven bytes, consisting of          |
|         |             |           | uppercase letters and numbers, this |
|         |             |           | terminal ID is defined by the       |
|         |             |           | manufacturer                        |
+---------+-------------+-----------+-------------------------------------+
| 24      | License     | BYTE      | License Plate Color                 |
|         | Plate Color |           |                                     |
+---------+-------------+-----------+-------------------------------------+
| 25      | License     | STRING    | 12 letters or numbers, symbols, \*  |
|         | Plate       |           | is not supported                    |
+---------+-------------+-----------+-------------------------------------+

5.  Terminal registration response【8100】

Message ID:0x8100

The data format of the terminal registration response message body is
shown in Table 7.

Table 7 Terminal registration response message body data format

+:-----:+:--------------:+:------:+:--------------------------------------:+
| Start | field          | Data   | description and requirements           |
| byte  |                |        |                                        |
|       |                | type   |                                        |
+-------+----------------+--------+----------------------------------------+
| 0     | Response       | WORD   | The serial number of the corresponding |
|       | serial number  |        | terminal registration message          |
+-------+----------------+--------+----------------------------------------+
| 2     | Result         | BYTE   | 0: successful; 1: the vehicle has been |
|       |                |        | registered; 2: the vehicle is not in   |
|       |                |        | the database; 3: the terminal has been |
|       |                |        | registered; 4: the vehicle is not in   |
|       |                |        | the database                           |
+-------+----------------+--------+----------------------------------------+
| 3     | Authentication | STRING | This field is only available after     |
|       | code           |        | success                                |
+-------+----------------+--------+----------------------------------------+

7.6 Terminal authentication【0102】

MESSAGE ID:0x0102

The data format of the terminal authentication message body is shown in
Table 8.

Table 8 Data format of terminal authentication message body

+:------:+:--------------:+:-------:+:------------------------------------:+
| Start  | field          | Data    | description and requirements         |
| byte   |                |         |                                      |
|        |                | type    |                                      |
+--------+----------------+---------+--------------------------------------+
| 0      | Authentication | STRING  | The terminal reconnects and reports  |
|        | code           |         | the authentication code              |
+--------+----------------+---------+--------------------------------------+

7.7 Setting terminal parameters【8103】

Message ID:0x8103

Setting the terminal parameter message body data format see Table 9

Table 9 Terminal parameter message body data format

+:-----:+:----------:+:------:+:---------------------------------------:+
| Start | field      | Data   | description and requirements            |
| byte  |            |        |                                         |
|       |            | type   |                                         |
+-------+------------+--------+-----------------------------------------+
| 0     | Total      | BYTE   |                                         |
|       | quantities |        |                                         |
|       | of         |        |                                         |
|       | parameters |        |                                         |
+-------+------------+--------+-----------------------------------------+
| 1     | Quantities |        | The format of the parameter item is     |
|       | of package |        | shown in Table 10                       |
|       | parameters |        |                                         |
+-------+------------+--------+-----------------------------------------+

Table 10 Data format of terminal parameter item

+:---------:+:------:+:------------------------------------------------:+
| field     | Data   | description and requirements                     |
|           |        |                                                  |
|           | type   |                                                  |
+-----------+--------+--------------------------------------------------+
| parameter | DWORD  | Parameter ID definitions and descriptions are    |
| ID        |        | shown in Table 11                                |
+-----------+--------+--------------------------------------------------+
| parameter | BYTE   |                                                  |
| length    |        |                                                  |
+-----------+--------+--------------------------------------------------+
| parameter |        | If it is a multi-value parameter, multiple       |
| value     |        | parameter items with the same ID are used in the |
|           |        | message, such as the phone number of the         |
|           |        | dispatch center                                  |
+-----------+--------+--------------------------------------------------+

Table 11 Definition and description of each parameter item of terminal
parameter setting

+:--------------:+:------:+:-------------------------------------------------:+
| parameter ID   | Data   | description and requirements                      |
|                |        |                                                   |
|                | type   |                                                   |
+----------------+--------+---------------------------------------------------+
| 0x0001         | DWORD  | The heartbeat sending interval of the terminal,   |
|                |        | the unit is (s)                                   |
+----------------+--------+---------------------------------------------------+
| 0x0002～0x000F |        | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0010         | STRING | The main server APN, the wireless communication   |
|                |        | dial-up access point. If the network standard is  |
|                |        | CDMA, this place is the PPP dial-up number        |
+----------------+--------+---------------------------------------------------+
| 0x0011         | STRING | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0012         | STRING | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0013         | STRING | Main server address, IP or domain name            |
+----------------+--------+---------------------------------------------------+
| 0x0014         | STRING | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0015         | STRING | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0016         | STRING | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0017         | STRING | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0018         | DWORD  | Server TCP port                                   |
+----------------+--------+---------------------------------------------------+
| 0x0019         | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x001A～0x001F |        | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0020         | DWORD  | Location reporting strategy, 0: regular           |
|                |        | reporting; 1: fixed-distance reporting; 2:        |
|                |        | regular and fixed-distance reporting              |
+----------------+--------+---------------------------------------------------+
| 0x0021         | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0022         | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0023～0x0026 | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0027         | DWORD  | Report time interval when sleeping, the unit is   |
|                |        | second (s), \> 0                                  |
+----------------+--------+---------------------------------------------------+
| 0x0028         | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0029         | DWORD  | Default time reporting interval, in seconds (s),  |
|                |        | \>0                                               |
+----------------+--------+---------------------------------------------------+
| 0x002A～0x002B | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x002C         | DWORD  | Default distance reporting interval, in meters    |
|                |        | (m), \>0                                          |
+----------------+--------+---------------------------------------------------+
| 0x002D         | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x002E         | DWORD  | Report distance interval when sleeping, the unit  |
|                |        | is meters (m), \> 0                               |
+----------------+--------+---------------------------------------------------+
| 0x002F～0x004F |        | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0050         | DWORD  | Alarm shielded word, corresponding to the alarm   |
|                |        | sign in the location information report message.  |
|                |        | if the corresponding bit is 1, the corresponding  |
|                |        | alarm is shielded.                                |
+----------------+--------+---------------------------------------------------+
| 0x0051         | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0052         | DWORD  | Alarm shooting switch, corresponding to the alarm |
|                |        | sign in the location information report message,  |
|                |        | if the corresponding bit is 1, the camera will    |
|                |        | shoot when the corresponding alarm occurs. time   |
|                |        | out\]                                             |
+----------------+--------+---------------------------------------------------+
| 0x0053         | DWORD  | The alarm shooting storage sign corresponds to    |
|                |        | the alarm sign in the location information report |
|                |        | message. If the corresponding bit is 1, the       |
|                |        | photos taken during the corresponding alarm will  |
|                |        | be stored, otherwise they will be uploaded in     |
|                |        | real time.                                        |
+----------------+--------+---------------------------------------------------+
|                | DWORD  | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0055         | DWORD  | Maximum speed in kilometers per hour (km/h)       |
+----------------+--------+---------------------------------------------------+
| 0x0056         | DWORD  | Overspeed duration, in seconds (s)                |
+----------------+--------+---------------------------------------------------+
| 0x0057～0x0074 |        | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0075         | \-     | Audio and video parameter settings, see Table 12  |
|                |        | for description                                   |
+----------------+--------+---------------------------------------------------+
| 0x0076         | \-     | Audio and video channel list settings, see Table  |
|                |        | 13 for description                                |
+----------------+--------+---------------------------------------------------+
| 0x0077         | \-     | Individual video channel parameter settings, see  |
|                |        | Table 15 for description                          |
+----------------+--------+---------------------------------------------------+
| 0x0079～0x007F |        | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0x0080         | DWORD  | Vehicle odometer reading, 1/10km                  |
+----------------+--------+---------------------------------------------------+
| 0x0081         | DWORD  | ID1                                               |
+----------------+--------+---------------------------------------------------+
| 0x0082         | DWORD  | ID2                                               |
+----------------+--------+---------------------------------------------------+
| 0x0083         | STRING | motor vehicle plate number                        |
+----------------+--------+---------------------------------------------------+
| 0x0084         | BYTE   | license plate color                               |
+----------------+--------+---------------------------------------------------+
| 0xF364         | \-     | ADAS system parameters, see Table 17              |
+----------------+--------+---------------------------------------------------+
| 0xF365         | \-     | DMS system parameters, see Table 18               |
+----------------+--------+---------------------------------------------------+
| 0xF366         | \-     | Reserve                                           |
+----------------+--------+---------------------------------------------------+
| 0xF367         | \-     | BSD system parameters, see Table 19               |
+----------------+--------+---------------------------------------------------+

Table 12 Definition and description of audio and video parameters

+:--------:+:-------------------:+:--------:+:-----------------------:+
| Start    | field               | Data     | description and         |
|          |                     |          | requirements            |
| byte     |                     | type     |                         |
+----------+---------------------+----------+-------------------------+
| 0        | real-time Streaming | BYTE     | O：CBR （Fixed bit      |
|          | Encoding Mode       |          | rate）；                |
|          |                     |          |                         |
|          |                     |          | 1：VBR（variable bit    |
|          |                     |          | rate）；                |
|          |                     |          |                         |
|          |                     |          | 2：ABR（Average bit     |
|          |                     |          | rate）；                |
|          |                     |          |                         |
|          |                     |          | 100-127：customize      |
+----------+---------------------+----------+-------------------------+
| 1        | real-time Streaming | BYTE     | O\~2：reserve；         |
|          | Resolution          |          |                         |
|          |                     |          | 3：D1；                 |
|          |                     |          |                         |
|          |                     |          | 4：WD1；                |
|          |                     |          |                         |
|          |                     |          | 5：720P；               |
|          |                     |          |                         |
|          |                     |          | 6：1080P；              |
|          |                     |          |                         |
|          |                     |          | 100\~127：customize     |
+----------+---------------------+----------+-------------------------+
| 2        | real-time Streaming | WORD     | range(1\~1000)          |
|          | Keyframe Interval   |          |                         |
+----------+---------------------+----------+-------------------------+
| 4        | real-time Streaming | BYTE     | range（l\~120）frame/s  |
|          | Target Frame Rate   |          |                         |
+----------+---------------------+----------+-------------------------+
| 5        | real-time Streaming | DWORD    | in kilobits per         |
|          | Target Bit Rate     |          | second（kbps）          |
+----------+---------------------+----------+-------------------------+
| 9        | Store stream        | BYTE     | O：CBR （Fixed bit      |
|          | encoding mode       |          | rate）；                |
|          |                     |          |                         |
|          |                     |          | 1：VBR（variable bit    |
|          |                     |          | rate）；                |
|          |                     |          |                         |
|          |                     |          | 2：ABR（Average bit     |
|          |                     |          | rate）；                |
|          |                     |          |                         |
|          |                     |          | 100-127：customize      |
+----------+---------------------+----------+-------------------------+
| 10       | Store stream        | BYTE     | > 0\~2：reserve；       |
|          | resolution          |          | >                       |
|          |                     |          | > 3：D1；               |
|          |                     |          | >                       |
|          |                     |          | > 4：WD1；              |
|          |                     |          | >                       |
|          |                     |          | > 5：720P；             |
|          |                     |          | >                       |
|          |                     |          | > 6：1080P；            |
|          |                     |          | >                       |
|          |                     |          | > 100 \~ 127：customize |
+----------+---------------------+----------+-------------------------+
| 11       | Store stream        | WORD     | range(l-1000)           |
|          | keyframe interval   |          |                         |
+----------+---------------------+----------+-------------------------+
| 13       | Store stream target | BYTE     | range(l\~120)           |
|          | frame rate          |          |                         |
+----------+---------------------+----------+-------------------------+
| 14       | Store stream target | DWORD    | in kilobits per         |
|          | bit rate            |          | second（kbps）          |
+----------+---------------------+----------+-------------------------+
| 18       | OSD text overlay    | WORD     | > Bitwise setting: 0    |
|          | settings            |          | > means no overlay, 1   |
|          |                     |          | > means overlay         |
|          |                     |          | >                       |
|          |                     |          | > bit0：Date and time   |
|          |                     |          | >                       |
|          |                     |          | > bitl：License plate   |
|          |                     |          | > number                |
|          |                     |          | >                       |
|          |                     |          | > bit2：logical channel |
|          |                     |          | > number                |
|          |                     |          |                         |
|          |                     |          | bit3：latitude and      |
|          |                     |          | longitude               |
|          |                     |          |                         |
|          |                     |          | bit4:driving record     |
|          |                     |          | speed                   |
|          |                     |          |                         |
|          |                     |          | bit5：satellite         |
|          |                     |          | positioning speed；     |
|          |                     |          |                         |
|          |                     |          | > bit6：Continuous      |
|          |                     |          | > driving time          |
|          |                     |          | >                       |
|          |                     |          | > bit7\~bitl0：Reserve  |
|          |                     |          | >                       |
|          |                     |          | > bitll                 |
|          |                     |          | > \~bitl5:customize     |
+----------+---------------------+----------+-------------------------+
| 20       | Whether to enable   | BYTE     | 0: not enabled; 1:      |
|          | audio output        |          | enabled                 |
+----------+---------------------+----------+-------------------------+

+:---------:+:---------:+:-----------:+:------------------------------:+
| **Channel | **Channel | **Channel   | **Monitoring area**            |
| number**  | name**    | type**      |                                |
+-----------+-----------+-------------+--------------------------------+
| 1         | Channel 1 | Audio/Video | > driver                       |
+-----------+-----------+-------------+--------------------------------+
| 2         | Channel 2 | Audio/Video | > directly in front of the     |
|           |           |             | > vehicle                      |
+-----------+-----------+-------------+--------------------------------+
| 3         | Channel 3 | Audio/Video | > other areas                  |
+-----------+-----------+-------------+--------------------------------+
| 4         | Channel 4 | Audio/Video | > other areas                  |
+-----------+-----------+-------------+--------------------------------+
| 5\~35     | Channel   | Reserve     | > reserve                      |
|           | 5\~35     |             |                                |
|           |           | reserve     |                                |
|           | Channel   |             |                                |
|           | 5\~35     |             |                                |
+-----------+-----------+-------------+--------------------------------+
| 36        | Channel   | Audio       | > Two-way talk/monitor         |
|           | 36        |             |                                |
+-----------+-----------+-------------+--------------------------------+

: Table 12-1 Definition of audio and video channels of on-board video
terminals of commercial vehicles

Table 13 List of audio and video channels

+:--------:+:-------------------:+:--------:+:-----------------------:+
| START    | FIELD               | DATA     | description and         |
|          |                     |          | requirements            |
| BYTE     |                     | TYPE     |                         |
+----------+---------------------+----------+-------------------------+
| 0        | Total quantities of | BYTE     | represented by 1        |
|          | audio and video     |          |                         |
|          | channels            |          |                         |
+----------+---------------------+----------+-------------------------+
| 1        | Total quantities of | BYTE     | represented by m        |
|          | audio channels      |          |                         |
+----------+---------------------+----------+-------------------------+
| 2        | Total quantities of | BYTE     | represented by n        |
|          | video channels      |          |                         |
+----------+---------------------+----------+-------------------------+
| 3        | Audio and video     | BYTE     | See Table 14            |
|          | channel comparison  | \[4x(l + |                         |
|          | table               | m4-n)\]  |                         |
+----------+---------------------+----------+-------------------------+

Table 14 Audio and video channel comparison table

+:--------:+:-------------------:+:--------:+:-----------------------:+
| START    | FIELD               | DATA     | description and         |
|          |                     |          | requirements            |
| BYTE     |                     | TYPE     |                         |
+----------+---------------------+----------+-------------------------+
| 0        | physical channel    | BYTE     | start from 1            |
|          | number              |          |                         |
+----------+---------------------+----------+-------------------------+
| 1        | logical channel     | BYTE     | Follow Table 12-1 in    |
|          | number              |          | the documentation       |
+----------+---------------------+----------+-------------------------+
| 2        | channel type        | BYTE     | > 0: audio and video;   |
|          |                     |          | > 1: audio; 2: video    |
+----------+---------------------+----------+-------------------------+
| 3        | Whether to connect  | BYTE     | > This field is valid   |
|          | the PTZ(            |          | > when the channel type |
|          | Pan/Tilt/Zoom)      |          | > is 0 and 2; 0: not    |
|          |                     |          | > connected; 1:         |
|          |                     |          | > connected             |
+----------+---------------------+----------+-------------------------+

Table 15 Individual video channel parameter settings

+:--------:+:-------------------:+:--------:+:-----------------------:+
| START    | FIELD               | DATA     | description and         |
|          |                     |          | requirements            |
| BYTE     |                     | TYPE     |                         |
+----------+---------------------+----------+-------------------------+
| 0        | Quantities of       | BYTE     | represented by n        |
|          | channels for        |          |                         |
|          | setting video       |          |                         |
|          | parameters          |          |                         |
|          | individually        |          |                         |
|          |                     |          |                         |
|          | Set video           |          |                         |
|          | parameters          |          |                         |
|          | individually        |          |                         |
|          |                     |          |                         |
|          | Number of channels  |          |                         |
+----------+---------------------+----------+-------------------------+
| 1        | List of individual  | BYTE\[21 | See Table16             |
|          | channel video       | xn\]     |                         |
|          | parameter settings  |          |                         |
+----------+---------------------+----------+-------------------------+

Table 16 Individual channel video parameter settings

+:--------:+:-------------------:+:--------:+:------------------------:+
| START    | FIELD               | DATA     | description and          |
|          |                     |          | requirements             |
| BYTE     |                     | TYPE     |                          |
+----------+---------------------+----------+--------------------------+
| 0        | logical channel     | BYTE     | Follow Table 12-1 in the |
|          | number              |          | documentation            |
+----------+---------------------+----------+--------------------------+
| 1        | real-time Streaming | BYTE     | > 0：CBR （Fixed bit     |
|          | Encoding Mode       |          | > rate）；               |
|          |                     |          | >                        |
|          |                     |          | > 1：VBR（variable bit   |
|          |                     |          | > rate）；               |
|          |                     |          | >                        |
|          |                     |          | > 2：ABR（Average bit    |
|          |                     |          | > rate）；               |
|          |                     |          | >                        |
|          |                     |          | > 100 \~ 127：customize  |
+----------+---------------------+----------+--------------------------+
| 2        | real-time Streaming | BYTE     | > 0：reserve；           |
|          | Resolution          |          | >                        |
|          |                     |          | > 1：reserve；           |
|          |                     |          | >                        |
|          |                     |          | > 2：reserve；           |
|          |                     |          | >                        |
|          |                     |          | > 3：D1；                |
|          |                     |          | >                        |
|          |                     |          | > 4：WD1；               |
|          |                     |          | >                        |
|          |                     |          | > 5：720P；              |
|          |                     |          | >                        |
|          |                     |          | > 6：1080P；             |
|          |                     |          | >                        |
|          |                     |          | > 100 \~ 127：customize  |
+----------+---------------------+----------+--------------------------+
| 3        | real-time Streaming | WORD     | range(1\~1000)           |
|          | Key Frame Interval  |          |                          |
+----------+---------------------+----------+--------------------------+
| 5        | real-time Streaming | BYTE     | range(l\~120)            |
|          | Target Frame Rate   |          |                          |
+----------+---------------------+----------+--------------------------+
| 6        | real-time Streaming | DWORD    | in kilobits per          |
|          | Target Bit Rate     |          | second（kbps）           |
+----------+---------------------+----------+--------------------------+
| 10       | Store stream        | BYTE     | > 0：CBR （Fixed bit     |
|          | encoding mode       |          | > rate）；               |
|          |                     |          | >                        |
|          |                     |          | > 1：VBR（variable bit   |
|          |                     |          | > rate）；               |
|          |                     |          | >                        |
|          |                     |          | > 2：ABR（Average bit    |
|          |                     |          | > rate）；               |
|          |                     |          | >                        |
|          |                     |          | > 100 \~ 127：customize  |
+----------+---------------------+----------+--------------------------+
| 11       | save stream         | BYTE     | > 0：reserve；           |
|          | resolution          |          | >                        |
|          |                     |          | > 1：reserve；           |
|          |                     |          | >                        |
|          |                     |          | > 2：reserve；           |
|          |                     |          | >                        |
|          |                     |          | > 3：D1；                |
|          |                     |          | >                        |
|          |                     |          | > 4：WD1；               |
|          |                     |          | >                        |
|          |                     |          | > 5：720P；              |
|          |                     |          | >                        |
|          |                     |          | > 6：1080P；             |
|          |                     |          | >                        |
|          |                     |          | > 100 \~ 127：customize  |
+----------+---------------------+----------+--------------------------+
| 12       | Store stream        | WORD     | range(1\~1000)           |
|          | keyframe interval   |          |                          |
+----------+---------------------+----------+--------------------------+
| 14       | Store stream target | BYTE     | range(l\~120)            |
|          | rate                |          |                          |
+----------+---------------------+----------+--------------------------+
| 15       | Store stream target | DWORD    | in kilobits per          |
|          | bit rate            |          | second（kbps）           |
+----------+---------------------+----------+--------------------------+
| 19       | OSD Overlay         | WORD     | Bitwise setting: 0 means |
|          | settings            |          | no overlay, 1 means      |
|          |                     |          | overlay                  |
|          |                     |          |                          |
|          |                     |          | bit0：Date and time      |
|          |                     |          |                          |
|          |                     |          | bitl：License plate      |
|          |                     |          | number                   |
|          |                     |          |                          |
|          |                     |          | bit2：logical channel    |
|          |                     |          | number                   |
|          |                     |          |                          |
|          |                     |          | bit3：latitude and       |
|          |                     |          | longitude                |
|          |                     |          |                          |
|          |                     |          | bit4:driving record      |
|          |                     |          | speed                    |
|          |                     |          |                          |
|          |                     |          | bit5：satellite          |
|          |                     |          | positioning speed；      |
|          |                     |          |                          |
|          |                     |          | > bit6\~bitl0：reserve； |
|          |                     |          | >                        |
|          |                     |          | > bitll                  |
|          |                     |          | > \~bitl5:customize      |
+----------+---------------------+----------+--------------------------+

+:-------------:+:--------------:+:---------:+:-----------------------------:+
| START         | FIELD          | DATA      | description and requirements  |
|               |                |           |                               |
| BYTE          |                | TYPE      |                               |
+---------------+----------------+-----------+-------------------------------+
| 0             | Alarm judgment | BYTE      | The unit is km/h, the value   |
|               | speed          |           | range is 0\~60, and the       |
|               | threshold      |           | default value is 30. It is    |
|               |                |           | only used for road departure  |
|               |                |           | alarm, forward collision      |
|               |                |           | alarm, vehicle distance alarm |
|               |                |           | and frequent lane change      |
|               |                |           | alarm,indicating that the     |
|               |                |           | alarm function can only be    |
|               |                |           | enabled when the vehicle      |
|               |                |           | speed is higher than this     |
|               |                |           | threshold                     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify this |
|               |                |           | parameter.                    |
+---------------+----------------+-----------+-------------------------------+
| 1             | Alarm prompt   | BYTE      | 0\~8, 8 is max, 0 is mute,    |
|               | volume         |           | default is 6                  |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 2             | Active photo   | BYTE      | 0x00：do not turn on          |
|               | strategy       |           |                               |
|               |                |           | 0x01：Take pictures regularly |
|               |                |           |                               |
|               |                |           | 0x02：Take pictures at a      |
|               |                |           | fixed distance                |
|               |                |           |                               |
|               |                |           | 0x03：reserve                 |
|               |                |           |                               |
|               |                |           | Defaults 0x00，               |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 3             | Active timed   | WORD      | The unit is second, the value |
|               | photo time     |           | range is 0\~3600, the default |
|               | interval       |           | value is 60,                  |
|               |                |           |                               |
|               |                |           | 0 means no snapshot, 0xFFFF   |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
|               |                |           |                               |
|               |                |           | Valid when the active camera  |
|               |                |           | policy is 0x01                |
+---------------+----------------+-----------+-------------------------------+
| 5             | Active timed   | WORD      | The unit is meter, the value  |
|               | photo time     |           | range is 0\~60000, the        |
|               | interval       |           | default value is 200,         |
|               |                |           |                               |
|               |                |           | 0 means no snapshot, 0xFFFF   |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
|               |                |           |                               |
|               |                |           | Valid when the active camera  |
|               |                |           | policy is 0x02                |
+---------------+----------------+-----------+-------------------------------+
| 7             | Quantities of  | BYTE      | The value range is 1-10, the  |
|               | photos taken   |           | default is 3                  |
|               | at one time    |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 8             | Active timed   | BYTE      | The unit is 100ms, the value  |
|               | photo time     |           | range is 1\~5, the default    |
|               |                |           | value is 2,                   |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 9             | Photo          | BYTE      | 0x01：352×288                 |
|               | resolution     |           |                               |
|               |                |           | 0x02：704×288                 |
|               |                |           |                               |
|               |                |           | 0x03：704×576                 |
|               |                |           |                               |
|               |                |           | 0x04：640×480                 |
|               |                |           |                               |
|               |                |           | 0x05：1280×720                |
|               |                |           |                               |
|               |                |           | 0x06：1920×1080               |
|               |                |           |                               |
|               |                |           | default value: 0x01，         |
|               |                |           |                               |
|               |                |           | 0xFF Indicates that the       |
|               |                |           | parameter is not modified，   |
|               |                |           |                               |
|               |                |           | This parameter also applies   |
|               |                |           | to the alarm trigger camera   |
|               |                |           | resolution.                   |
+---------------+----------------+-----------+-------------------------------+
| 10            | Video          | BYTE      | 0x01：reserve                 |
|               | recording      |           |                               |
|               | resolution     |           | 0x02：reserve                 |
|               |                |           |                               |
|               |                |           | 0x03：D1                      |
|               |                |           |                               |
|               |                |           | 0x04：WD1                     |
|               |                |           |                               |
|               |                |           | 0x05：VGA                     |
|               |                |           |                               |
|               |                |           | 0x06：720P                    |
|               |                |           |                               |
|               |                |           | 0x07：1080P                   |
|               |                |           |                               |
|               |                |           | default value: 0x01           |
|               |                |           |                               |
|               |                |           | 0xFF Indicates that the       |
|               |                |           | parameter is not modified，   |
|               |                |           |                               |
|               |                |           | This parameter also applies   |
|               |                |           | to the alarm trigger camera   |
|               |                |           | resolution.                   |
+---------------+----------------+-----------+-------------------------------+
| 11            | Alarm enable   | DWORD     | Alarm enable bit 0: off 1: on |
|               |                |           |                               |
|               |                |           | bit0:Obstacle detection       |
|               |                |           | first-level alarm             |
|               |                |           |                               |
|               |                |           | bit1:Obstacle detection       |
|               |                |           | second-level alarm            |
|               |                |           |                               |
|               |                |           | bit2:First-level alarm for    |
|               |                |           | frequent lane changes         |
|               |                |           |                               |
|               |                |           | bit3:Second-level alarm for   |
|               |                |           | frequent lane changes         |
|               |                |           |                               |
|               |                |           | bit4:Lane Departure first     |
|               |                |           | level Warning                 |
|               |                |           |                               |
|               |                |           | bit5:Lane Departure second    |
|               |                |           | level Warning                 |
|               |                |           |                               |
|               |                |           | bit6:Forward Collision first  |
|               |                |           | level Warning                 |
|               |                |           |                               |
|               |                |           | bit7:Forward Collision second |
|               |                |           | level Warning                 |
|               |                |           |                               |
|               |                |           | bit8:Pedestrian collision     |
|               |                |           | first level warning           |
|               |                |           |                               |
|               |                |           | bit9:Pedestrian collision     |
|               |                |           | second level warning          |
|               |                |           |                               |
|               |                |           | bit10:vehicle too close first |
|               |                |           | level alarm                   |
|               |                |           |                               |
|               |                |           | bit11:vehicle too close       |
|               |                |           | second level alarm            |
|               |                |           |                               |
|               |                |           | bit12\~bit15：Customize       |
|               |                |           |                               |
|               |                |           | bit16:Road sign overrun alarm |
|               |                |           |                               |
|               |                |           | bit17\~bit29：Customize       |
|               |                |           |                               |
|               |                |           | bit30\~bit31:reserve          |
|               |                |           |                               |
|               |                |           | default: 0x00010FFF           |
|               |                |           |                               |
|               |                |           | 0xFFFFFFFF Indicates that the |
|               |                |           | parameter is not modified     |
+---------------+----------------+-----------+-------------------------------+
| 15            | event enable   | DWORD     | Event Enable Bit 0: Off 1: On |
|               |                |           |                               |
|               |                |           | bit0:road sign recognition    |
|               |                |           |                               |
|               |                |           | bit1:Take the initiative to   |
|               |                |           | take pictures                 |
|               |                |           |                               |
|               |                |           | bit2\~bit29：Customize        |
|               |                |           |                               |
|               |                |           | bit30\~bit31:reserve          |
|               |                |           |                               |
|               |                |           | default: 0x00000003           |
|               |                |           |                               |
|               |                |           | 0xFFFFFFFF Indicates that the |
|               |                |           | parameter is not modified     |
+---------------+----------------+-----------+-------------------------------+
| 19            | reserved field | BYTE      | reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 20            | reserved field | BYTE      | reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 21            | reserved field | BYTE      | reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 22            | reserved field | BYTE      | reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 23            | reserved field | BYTE      | reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 24            | reserved field | BYTE      | reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 25            | Frequent lane  | BYTE      | The unit is second, the value |
|               | change alarm   |           | range is 30\~120, the default |
|               | judgment time  |           | value is 60,                  |
|               | period         |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 26            | Frequent lane  | BYTE      | The number of lane changes is |
|               | change alarm   |           | 3\~10, the default is 5,0xFF  |
|               | judgment times |           | means do not modify the       |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 27            | Frequent lane  | BYTE      | The unit is km/h, the value   |
|               | change alarm   |           | range is 0\~220, and the      |
|               | classification |           | default value is 50, which    |
|               | speed          |           | means that when the alarm is  |
|               | threshold      |           | triggered, the vehicle speed  |
|               |                |           | is higher than the threshold, |
|               |                |           | which is a second-level       |
|               |                |           | alarm, otherwise it is a      |
|               |                |           | first-level alarm.            |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters.                   |
+---------------+----------------+-----------+-------------------------------+
| 28            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5,                   |
|               | after frequent |           |                               |
|               | lane change    |           | 0 means no recording, 0xFF    |
|               | alarm          |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 29            | Quantities of  | BYTE      | The value range is 0-10, the  |
|               | pictures taken |           | default value is 3,           |
|               | for frequent   |           |                               |
|               | lane change    |           | 0 means no snapshot, 0xFF     |
|               | alarms         |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 30            | Frequent lane  | BYTE      | The unit is 100ms, the value  |
|               | change alarm   |           | range is 1\~10, the default   |
|               | camera         |           | is 2,                         |
|               | interval       |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 31            | Lane Departure | BYTE      | The unit is km/h, the value   |
|               | Alert          |           | range is 0\~220, and the      |
|               | Classification |           | default value is 50.          |
|               | Speed          |           | Indicates that the vehicle    |
|               | Threshold      |           | speed is higher than the      |
|               |                |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm.    |
|               |                |           |                               |
|               |                |           | 0 means no recording, 0xFF    |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 32            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5,                   |
|               | after lane     |           |                               |
|               | departure      |           | 0 means no recording, 0xFF    |
|               | warning        |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 33            | Quantities of  | BYTE      | The value range is 0-10, the  |
|               | photos taken   |           | default value is 3,           |
|               | for lane       |           |                               |
|               | departure      |           | 0 means no snapshot, 0xFF     |
|               | warning        |           | means no modification         |
+---------------+----------------+-----------+-------------------------------+
| 34            | Lane departure | BYTE      | The unit is 100ms, the value  |
|               | warning photo  |           | range is 1\~10, the default   |
|               | interval       |           | value is 2                    |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 35            | Forward        | BYTE      | The unit is 100ms, and the    |
|               | Collision      |           | value range is 10\~50.        |
|               | Warning Time   |           | Currently, the national       |
|               | Threshold      |           | standard value of 27 is used, |
|               |                |           | and the interface for         |
|               |                |           | modification is reserved.     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 36            | Forward        | BYTE      | The unit is km/h, the value   |
|               | Collision      |           | range is 0\~220, and the      |
|               | Alert          |           | default value is 50.          |
|               | Classification |           | Indicates that the vehicle    |
|               | Speed          |           | speed is higher than the      |
|               | Threshold      |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 37            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5,                   |
|               | after forward  |           |                               |
|               | collision      |           | 0 means no recording, 0xFF    |
|               | warning        |           | means no parameter            |
|               |                |           | modification.                 |
+---------------+----------------+-----------+-------------------------------+
| 38            | Quantities of  | BYTE      | The value range is 0-10, the  |
|               | photos taken   |           | default value is 3,           |
|               | for forward    |           |                               |
|               | collision      |           | 0 means no snapshot, 0xFF     |
|               | warning        |           | means no modification.        |
+---------------+----------------+-----------+-------------------------------+
| 39            | Forward        | BYTE      | The unit is 100ms, the value  |
|               | Collision      |           | range is 1\~10, the default   |
|               | Alarm Photo    |           | value is 2,                   |
|               | Interval       |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 40            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 41            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 42            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 43            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 44            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 45            | Vehicle        | BYTE      | The unit is 100ms, the value  |
|               | distance       |           | range is 10-50, the default   |
|               | monitoring     |           | value is 10,                  |
|               | alarm distance |           |                               |
|               | threshold      |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 46            | Vehicle        | BYTE      | The unit is km/h, the value   |
|               | distance       |           | range is 0\~220, and the      |
|               | monitoring     |           | default value is 50.          |
|               | alarm          |           | Indicates that the vehicle    |
|               | classification |           | speed is higher than the      |
|               | speed          |           | threshold when the alarm is   |
|               | threshold      |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 47            | Video          | BYTE      | Video recording time before   |
|               | recording time |           | and after the vehicle         |
|               | before and     |           | distance alarm                |
|               | after the      |           |                               |
|               | vehicle        |           |                               |
|               | distance alarm |           |                               |
+---------------+----------------+-----------+-------------------------------+
| 48            | The quantities | BYTE      | The value range is 0-10, the  |
|               | of photos      |           | default value is 3,           |
|               | taken for the  |           |                               |
|               | alarm when the |           | 0 means no snapshot, 0xFF     |
|               | vehicle is too |           | means no parameter            |
|               | close          |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 49            | Vehicle        | BYTE      | The unit is 100ms, the value  |
|               | distance is    |           | range is 1\~10, the default   |
|               | too close      |           | value is 2,                   |
|               | alarm photo    |           |                               |
|               | interval       |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 50            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 51            | Reserved field | BYTE      | Reserve                       |
+---------------+----------------+-----------+-------------------------------+
| 52            | Reserved field | BYTE\[4\] |                               |
+---------------+----------------+-----------+-------------------------------+

: Table 17 Advanced driver assistance system parameters

+:-------------:+:--------------:+:---------:+:-----------------------------:+
| START         | FIELD          | DATA TYPE | description and requirements  |
|               |                |           |                               |
| BYTE          |                |           |                               |
+---------------+----------------+-----------+-------------------------------+
| 0             | Alarm judgment | BYTE      | The unit is km/h, the value   |
|               | speed          |           | range is 0\~60, and the       |
|               | threshold      |           | default value is 30.          |
|               |                |           | Indicates that the alarm      |
|               |                |           | function can only be enabled  |
|               |                |           | when the vehicle speed is     |
|               |                |           | higher than this threshold    |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify this |
|               |                |           | parameter                     |
+---------------+----------------+-----------+-------------------------------+
| 1             | Alarm volume   | BYTE      | 0\~8, 8 is max, 0 is mute,    |
|               |                |           | default is 6                  |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 2             | Active photo   | BYTE      | 0x00：don't turn on           |
|               | strategy       |           |                               |
|               |                |           | 0x01：Take pictures regularly |
|               |                |           |                               |
|               |                |           | 0x02：Take pictures at a      |
|               |                |           | fixed distance                |
|               |                |           |                               |
|               |                |           | 0x03：card trigger            |
|               |                |           |                               |
|               |                |           | 0x04：reserve                 |
|               |                |           |                               |
|               |                |           | Defaults:0x00，               |
|               |                |           |                               |
|               |                |           | 0xFF indicates that the       |
|               |                |           | parameter is not modified     |
+---------------+----------------+-----------+-------------------------------+
| 3             | Active timed   | WORD      | The unit is second, the value |
|               | photo time     |           | range is 60\~60000, the       |
|               | interval       |           | default value is 3600         |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 5             | Active fixed   | WORD      | The unit is meter, the value  |
|               | distance       |           | range is 0\~60000, the        |
|               | camera         |           | default value is 200          |
|               | distance       |           |                               |
|               | interval       |           | 0 means no snapshot, 0xFFFF   |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
|               |                |           |                               |
|               |                |           | It is valid when the active   |
|               |                |           | photo strategy is 02.         |
+---------------+----------------+-----------+-------------------------------+
| 7             | Quantities of  | BYTE      | The value range is 1-10.      |
|               | photos taken   |           | default value 3,              |
|               | at one time    |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 8             | Time interval  | BYTE      | The unit is 100ms, the value  |
|               | for a single   |           | range is 1\~5, the default    |
|               | active photo   |           | value is 2,                   |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 9             | Photo          | BYTE      | 0x01：352×288                 |
|               | resolution     |           |                               |
|               |                |           | 0x02：704×288                 |
|               |                |           |                               |
|               |                |           | 0x03：704×576                 |
|               |                |           |                               |
|               |                |           | 0x04：640×480                 |
|               |                |           |                               |
|               |                |           | 0x05：1280×720                |
|               |                |           |                               |
|               |                |           | 0x06：1920×1080               |
|               |                |           |                               |
|               |                |           | Default value 0x01,           |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters,                   |
|               |                |           |                               |
|               |                |           | This parameter is also        |
|               |                |           | applicable to the resolution  |
|               |                |           | of the alarm-triggered photo. |
+---------------+----------------+-----------+-------------------------------+
| 10            | Video          | BYTE      | 0x01：reserve                 |
|               | recording      |           |                               |
|               | resolution     |           | 0x02：reserve                 |
|               |                |           |                               |
|               |                |           | 0x03：D1                      |
|               |                |           |                               |
|               |                |           | 0x04：WD1                     |
|               |                |           |                               |
|               |                |           | 0x05：VGA                     |
|               |                |           |                               |
|               |                |           | 0x06：720P                    |
|               |                |           |                               |
|               |                |           | 0x07：1080P                   |
|               |                |           |                               |
|               |                |           | Default value 0x01,           |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
|               |                |           |                               |
|               |                |           | This parameter also applies   |
|               |                |           | to alarm trigger video        |
|               |                |           | resolution.                   |
+---------------+----------------+-----------+-------------------------------+
| 11            | Alarm enable   | DWORD     | Alarm enable bit 0: off 1: on |
|               |                |           |                               |
|               |                |           | bit0：Fatigue driving level 1 |
|               |                |           | alarm                         |
|               |                |           |                               |
|               |                |           | bit1: Fatigue driving level 2 |
|               |                |           | alarm                         |
|               |                |           |                               |
|               |                |           | bit2: First-level alarm for   |
|               |                |           | incoming and outgoing calls   |
|               |                |           |                               |
|               |                |           | Bit3: Second-level alarm for  |
|               |                |           | incoming and outgoing calls   |
|               |                |           |                               |
|               |                |           | bit4：Smoking first-level     |
|               |                |           | alarm                         |
|               |                |           |                               |
|               |                |           | bit5：Smoking second-level    |
|               |                |           | alarm                         |
|               |                |           |                               |
|               |                |           | bit6：Distracted driving      |
|               |                |           | level 1 alarm                 |
|               |                |           |                               |
|               |                |           | bit7：Distracted Driving      |
|               |                |           | Level 2 Alarm                 |
|               |                |           |                               |
|               |                |           | bit8：Level 1 alarm for       |
|               |                |           | driver abnormality            |
|               |                |           |                               |
|               |                |           | bit9: Level 2 alarm for       |
|               |                |           | driver abnormality            |
|               |                |           |                               |
|               |                |           | bit10\~bit29：customize       |
|               |                |           |                               |
|               |                |           | bit30\~bit31：reserve         |
|               |                |           |                               |
|               |                |           | Default value:0x000001FF      |
|               |                |           |                               |
|               |                |           | 0xFFFFFFFF means do not       |
|               |                |           | modify the parameters         |
+---------------+----------------+-----------+-------------------------------+
| 15            | event enable   | DWORD     | Event Enable Bit 0: Off 1: On |
|               |                |           |                               |
|               |                |           | bit0：driver change event     |
|               |                |           |                               |
|               |                |           | bit1：Active photo event      |
|               |                |           |                               |
|               |                |           | bit2\~bit29：customize        |
|               |                |           |                               |
|               |                |           | bit30\~bit31：reserve         |
|               |                |           |                               |
|               |                |           | Default value0x00000003       |
|               |                |           |                               |
|               |                |           | 0xFFFFFFFF means do not       |
|               |                |           | modify the parameters         |
+---------------+----------------+-----------+-------------------------------+
| 19            | Smoking alarm  | WORD      | The unit is second, the value |
|               | judgment time  |           | range is 0\~3600. The default |
|               | interval       |           | value is 180. Indicates that  |
|               |                |           | only one smoking alarm is     |
|               |                |           | triggered during this         |
|               |                |           | interval.                     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify this |
|               |                |           | parameter                     |
+---------------+----------------+-----------+-------------------------------+
| 21            | Time interval  | WORD      | The unit is second, the value |
|               | for receiving  |           | range is 0\~3600. The default |
|               | and calling    |           | value is 120. Indicates that  |
|               | alarm judgment |           | the incoming and outgoing     |
|               |                |           | call alarm is triggered only  |
|               |                |           | once within this time         |
|               |                |           | interval.                     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify this |
|               |                |           | parameter                     |
+---------------+----------------+-----------+-------------------------------+
| 23            | reserve field  | BYTE\[3\] | reserve field                 |
+---------------+----------------+-----------+-------------------------------+
| 26            | Fatigue        | BYTE      | The unit is km/h, the value   |
|               | driving        |           | range is 0\~220, and the      |
|               | warning        |           | default value is 50.          |
|               | classification |           | Indicates that the vehicle    |
|               | speed          |           | speed is higher than the      |
|               | threshold      |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 27            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5                    |
|               | after fatigue  |           |                               |
|               | driving alarm  |           | 0 means no recording, 0xFF    |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 28            | Quantities of  | BYTE      | The value range is 0-10, the  |
|               | photos taken   |           | default value is 3            |
|               | for fatigue    |           |                               |
|               | driving alarm  |           | 0 means no snapshot, 0xFF     |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 29            | Fatigue        | BYTE      | The unit is 100ms, the value  |
|               | driving alarm  |           | range is 1\~5, the default is |
|               | photographing  |           | 2,                            |
|               | interval       |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 30            | Alarm          | BYTE      | The unit is km/h, the value   |
|               | classification |           | range is 0\~220, and the      |
|               | speed          |           | default value is 50.          |
|               | threshold for  |           | Indicates that the vehicle    |
|               | incoming and   |           | speed is higher than the      |
|               | outgoing calls |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 31            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5,                   |
|               | after the call |           |                               |
|               | and alarm      |           | 0 means no recording, 0xFF    |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 32            | Quantities of  | BYTE      | The value range is 1-10, the  |
|               | photos of the  |           | default value is 3            |
|               | driver\'s      |           |                               |
|               | facial         |           | 0 means no snapshot, 0xFF     |
|               | features taken |           | means no parameter            |
|               | when answering |           | modification                  |
|               | the phone and  |           |                               |
|               | calling the    |           |                               |
|               | police         |           |                               |
+---------------+----------------+-----------+-------------------------------+
| 33            | Interval time  | BYTE      | The unit is 100ms, the value  |
|               | between        |           | range is 1\~5, the default    |
|               | answering and  |           | value is 2                    |
|               | calling the    |           |                               |
|               | police and     |           | 0xFF means do not modify the  |
|               | taking photos  |           | parameters                    |
|               | of the         |           |                               |
|               | driver\'s      |           |                               |
|               | facial         |           |                               |
|               | features       |           |                               |
+---------------+----------------+-----------+-------------------------------+
| 34            | Smoking alarm  | BYTE      | The unit is km/h, the value   |
|               | classification |           | range is 0\~220, and the      |
|               | speed          |           | default value is 50.          |
|               | threshold      |           | Indicates that the vehicle    |
|               |                |           | speed is higher than the      |
|               |                |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 35            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5                    |
|               | after smoking  |           |                               |
|               | alarm          |           | 0 means no recording, 0xFF    |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 36            | Quantities of  | BYTE      | The value range is 1-10, the  |
|               | photos of the  |           | default value is 3            |
|               | driver\'s      |           |                               |
|               | facial         |           | 0 means no snapshot, 0xFF     |
|               | features taken |           | means no parameter            |
|               | by the smoking |           | modification                  |
|               | alarm          |           |                               |
+---------------+----------------+-----------+-------------------------------+
| 37            | Interval time  | BYTE      | The unit is 100ms, the value  |
|               | between        |           | range is 1\~5, the default is |
|               | smoking alarm  |           | 2                             |
|               | and taking     |           |                               |
|               | photos of      |           | 0xFF means do not modify the  |
|               | driver\'s      |           | parameters                    |
|               | facial         |           |                               |
|               | features       |           |                               |
+---------------+----------------+-----------+-------------------------------+
| 38            | Distracted     | BYTE      | The unit is km/h, the value   |
|               | Driving        |           | range is 0\~220, and the      |
|               | Warning Graded |           | default value is 50.          |
|               | Speed          |           | Indicates that the vehicle    |
|               | Threshold      |           | speed is higher than the      |
|               |                |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 39            | Video          | BYTE      | The unit is second, the value |
|               | recording time |           | range is 0-60, the default    |
|               | before and     |           | value is 5                    |
|               | after          |           |                               |
|               | distracted     |           | 0 means no recording, 0xFF    |
|               | driving alarm  |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 40            | Quantities of  | BYTE      | The value range is 1-10, the  |
|               | photos taken   |           | default value is 3            |
|               | for distracted |           |                               |
|               | driving alarm  |           | 0 means no snapshot, 0xFF     |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 41            | Distracted     | BYTE      | The unit is 100ms, the value  |
|               | driving alarm  |           | range is 1\~5, the default is |
|               | photo interval |           | 2                             |
|               | time           |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 42            | Speed          | BYTE      | The unit is km/h, the value   |
|               | threshold for  |           | range is 0\~220, and the      |
|               | abnormal       |           | default value is 50.          |
|               | driving        |           | Indicates that the vehicle    |
|               | behavior       |           | speed is higher than the      |
|               | classification |           | threshold when the alarm is   |
|               |                |           | triggered, it is a            |
|               |                |           | second-level alarm, otherwise |
|               |                |           | it is a first-level alarm     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 43            | Abnormal       | BYTE      | The unit is second, the value |
|               | driving        |           | range is 0-60, the default    |
|               | behavior video |           | value is 5                    |
|               | recording time |           |                               |
|               |                |           | 0 means no recording, 0xFF    |
|               |                |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 44            | Quantities of  | BYTE      | The value range is 1-10, the  |
|               | snapshots of   |           | default value is 3            |
|               | abnormal       |           |                               |
|               | driving        |           | 0 means no snapshot, 0xFF     |
|               | behavior       |           | means no parameter            |
|               |                |           | modification                  |
+---------------+----------------+-----------+-------------------------------+
| 45            | Abnormal       | BYTE      | The unit is 100ms, the value  |
|               | driving        |           | range is 1\~5, the default is |
|               | behavior       |           | 2                             |
|               | photographing  |           |                               |
|               | interval       |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 46            | reserve        | BYTE      | 0x00：reserve                 |
|               |                |           |                               |
|               |                |           | 0x01：reserve                 |
|               |                |           |                               |
|               |                |           | 0x02：reserve                 |
|               |                |           |                               |
|               |                |           | 0x03：reserve                 |
|               |                |           |                               |
|               |                |           | 0x04：reserve                 |
|               |                |           |                               |
|               |                |           | The default value is 0x01     |
|               |                |           |                               |
|               |                |           | 0xFF means do not modify the  |
|               |                |           | parameters                    |
+---------------+----------------+-----------+-------------------------------+
| 47            | reserve field  | BYTE\[2\] |                               |
+---------------+----------------+-----------+-------------------------------+

: Table 18 Parameters of driver status monitoring system

+:-------------:+:-----------:+:--------:+:-----------------------------:+
| START         | FIELD       | DATA     | description and requirements  |
|               |             |          |                               |
| BYTE          |             | TYPE     |                               |
+---------------+-------------+----------+-------------------------------+
| 0             | Rear        | BYTE     | The unit is second, the value |
|               | approaching |          | range is 1\~10                |
|               | alarm time  |          |                               |
|               | threshold   |          | 0xFF means do not modify the  |
|               |             |          | parameters                    |
+---------------+-------------+----------+-------------------------------+
| 1             | Side rear   | BYTE     | The unit is second, the value |
|               | approach    |          | range is 1\~10                |
|               | alarm time  |          |                               |
|               | threshold   |          | 0xFF means do not modify the  |
|               |             |          | parameters                    |
+---------------+-------------+----------+-------------------------------+

: Table 19 Blind spot monitoring system parameters

7.8 Query terminal parameters【8104】

message ID:0x8104

Query terminal parameter message body is empty

7.9 Inquire terminal parameter response【0104】

message ID:0x0104

Query terminal parameter response message body data format see Table 20

Table 20 Query terminal parameter response message body data format

+:-------:+:-----------:+:---------:+:-------------------------------:+
| START   | FIELD       | DATA      | description and requirements    |
|         |             |           |                                 |
| BYTE    |             | TYPE      |                                 |
+---------+-------------+-----------+---------------------------------+
| 0       | Reply       | WORD      | The serial number of the        |
|         | serial      |           | corresponding terminal          |
|         | number      |           | parameter query message         |
+---------+-------------+-----------+---------------------------------+
| 2       | quantities  | BYTE      |                                 |
|         | of response |           |                                 |
|         | parameters  |           |                                 |
+---------+-------------+-----------+---------------------------------+
| 3       | parameter   |           | The format and definition of    |
|         | list        |           | parameter items are shown in    |
|         |             |           | Table 10                        |
+---------+-------------+-----------+---------------------------------+

7.10 terminal control【8105】 （Don't support GH03T）

message ID:0x8105

The data format of the terminal control message body is shown in Table
21

Table 21 Data format of terminal control message body

+:-----:+:----------:+:------:+:-------------------------------------:+
| START | FIELD      | DATA   | description and requirements          |
| BYTE  |            |        |                                       |
|       |            | TYPE   |                                       |
+-------+------------+--------+---------------------------------------+
| 0     | Command    | BYTE   | The terminal control command          |
|       | word       |        | description is shown in Table 22      |
+-------+------------+--------+---------------------------------------+
| 1     | Command    | STRING | The command parameter format is       |
|       | parameters |        | described later. Each field is        |
|       |            |        | separated by a half-width \";\". Each |
|       |            |        | STRING field is first processed by    |
|       |            |        | GBK encoding and then formed into a   |
|       |            |        | message.                              |
+-------+------------+--------+---------------------------------------+

Table 22 Description of Terminal Control Commands

  --------- ------------ ---------------------------------------------------
   Command  Command                 description and requirements
    word    parameters   

      1     reserve                            reserve

      2     reserve                            reserve

      3     reserve                            reserve

      4     none                           terminal reset

      5     reserve                            reserve

      6     reserve                            reserve

      7     reserve                            reserve

    Ox64    none                               oil cut

    0x65    none                               oil on
  --------- ------------ ---------------------------------------------------

7.11 location information report【0200】

The location information report message body consists of a list of
location basic information and location additional information items,
and the message structure is shown in Figure 3.

  -------------------------------------- --------------------------------
        Basic location information       List of location extension items

  -------------------------------------- --------------------------------

Figure 3 Location report message structure diagram

The list of location additional information items is a combination of
each location additional information item, or may not, and is determined
according to the length field in the message header.

The data format of the basic location information is shown in Table 23.

Table 23 Location Basic Information Data Format

+:-----:+:---------:+:--------:+:----------------------------------------:+
| START | FIELD     | DATA     | description                              |
| BYTE  |           |          |                                          |
|       |           | TYPE     |                                          |
+-------+-----------+----------+------------------------------------------+
| 0     | alarm     | DWORD    | The definition of the alarm sign is      |
|       | sign      |          | shown in Table 24                        |
+-------+-----------+----------+------------------------------------------+
| 4     | status    | DWORD    | Status bits are defined in Table 25      |
+-------+-----------+----------+------------------------------------------+
| 8     | latitude  | DWORD    | Latitude value in degrees multiplied by  |
|       |           |          | 10 to the 6th power, accurate to one     |
|       |           |          | millionth of a degree                    |
+-------+-----------+----------+------------------------------------------+
| 12    | longitude | DWORD    | Longitude value in degrees multiplied by |
|       |           |          | 10 to the 6th power, accurate to one     |
|       |           |          | millionth of a degree                    |
+-------+-----------+----------+------------------------------------------+
| 16    | Elevation | WORD     | Altitude, in meters (m)                  |
+-------+-----------+----------+------------------------------------------+
| 18    | speed     | WORD     | 1/10km/h                                 |
+-------+-----------+----------+------------------------------------------+
| 20    | direction | WORD     | 0---359，Facing North is 0, clockwise    |
+-------+-----------+----------+------------------------------------------+
| 21    | 时间time  | BCD\[6\] | YY-MM-DD-hh-mm-ss（Default 0 time zone） |
+-------+-----------+----------+------------------------------------------+

Table 24 Definition of Alarm Standard Bits

+:-------:+:-----------------------:+:--------------------------------:+
| Bit     | definition              | Handling instructions            |
+---------+-------------------------+----------------------------------+
| 0       | 1：Emergency alarm,     | Cleared after receiving response |
|         | triggered after the     |                                  |
|         | alarm switch is         |                                  |
|         | triggered               |                                  |
+---------+-------------------------+----------------------------------+
| 1       | 1：Overspeed alarm      | The sign is maintained until the |
|         |                         | alarm condition is released      |
+---------+-------------------------+----------------------------------+
| 2       | 1：reserve              | reserve                          |
+---------+-------------------------+----------------------------------+
| 3       | 1：reserve              | reserve                          |
+---------+-------------------------+----------------------------------+
| 4       | 1：GNSS module failure  | The sign is maintained until the |
|         |                         | alarm condition is released      |
+---------+-------------------------+----------------------------------+
| 5       | 1：GNSS antenna missing | The sign is maintained until the |
|         | or clipped              | alarm condition is released      |
+---------+-------------------------+----------------------------------+
| 6       | 1：reserve              | reserve                          |
+---------+-------------------------+----------------------------------+
| 7       | 1：Terminal main power  | The sign is maintained until the |
|         | supply undervoltage     | alarm condition is released      |
+---------+-------------------------+----------------------------------+
| 8       | 1：                     | The sign is maintained until the |
|         |                         | alarm condition is released      |
|         | The main power of the   |                                  |
|         | terminal is powered off |                                  |
+---------+-------------------------+----------------------------------+
| 9\~31   | 1：reserve              | reserve                          |
+---------+-------------------------+----------------------------------+

Table 25 Status Bit Definitions

+:--------------------:+:---------------------------------------------:+
| bit                  | status                                        |
+----------------------+-----------------------------------------------+
| 0                    | 0：ACC off 1：ACC on                          |
+----------------------+-----------------------------------------------+
| 1                    | 0：no position 1：position                    |
+----------------------+-----------------------------------------------+
| 2                    | 0：north latitude 1：south latitude           |
+----------------------+-----------------------------------------------+
| 3                    | 0：East longitude 1：west longitude           |
+----------------------+-----------------------------------------------+
| 4～9                 | reserve                                       |
+----------------------+-----------------------------------------------+
| 10                   | 0： Vehicle oil circuit is normal             |
|                      |                                               |
|                      | 1： Vehicle oil circuit disconnected          |
+----------------------+-----------------------------------------------+
| 11                   | 0: The vehicle circuit is normal 1: The       |
|                      | vehicle circuit is disconnected               |
+----------------------+-----------------------------------------------+
| 12                   | reserve                                       |
+----------------------+-----------------------------------------------+
| 13                   | 0：front door                                 |
|                      |                                               |
|                      | 1：Front door open (door magnetic wire)       |
+----------------------+-----------------------------------------------+
| 14\~31               | reserve                                       |
+----------------------+-----------------------------------------------+

The format of the location additional information item is shown in Table
26.

Table 26 Format of location additional information item

  ------------- -------- --------------------------------------------------
      field       data              description and requirements
                  type   

   Additional     BYTE                         1～255
   Information           
       ID                

   Additional     BYTE   
   information           
     length              

   Additional              Additional information is defined in Table 27
   information           
  ------------- -------- --------------------------------------------------

Table 27 Additional Information Definitions

  ------------- ------------- -----------------------------------------------
   Additional    Additional            description and requirements
   Information   information  
       ID          length     

      0x01            4        Mileage, DWORD, 1/10km, corresponding to the
                                          reading on the odometer

      0x02            2                           reserve

      0x03           ２                    speed，WORD,1/10km/h

   0x04～0x13                                     reserve

      0x14            4         Video related alarm, DWORD, set by bit, see
                                  Table 28 for the definition of sign bit

      0x15            4         Video sign loss alarm status, DWORD, set by
                              bit, bit0～bit31 respectively represent the 1st
                               to 32nd logical channel, if the corresponding
                               bit is 1, it means the video sign is lost in
                                           this logical channel.

      0x16            4       Video sign blocking alarm status, DWORD, set by
                              bit, bit0～bit31 respectively represent the 1st
                              to 32nd logical channel, the corresponding bit
                               is 1, it means the video sign blocking occurs
                                         in this logical channel.

      0x17            2       Memory failure alarm status, WORD, set by bit,
                                  bit0\~bit11 respectively represent the
                                    1st\~12th main memory, bit12\~bit15
                               respectively represent the 1st\~4th disaster
                              recovery storage device, the corresponding bit
                                   is 1, it means the memory is faulty.

      0x18            2          Detailed description of abnormal driving
                                  behavior alarm, WORD, see Table 29 for
                                                definition

      0x64           \-        ADDAS alarm information, see Table 30 for the
                                                definition

      0x65           \-         DMS alarm information, see Table 31 for the
                                                definition

      0x66           \-                           Reserve

      0x67           \-         BSD alarm information, see Table 32 for the
                                                definition

      0xEB           \-                           Reserve

      0xEF           \-                           Reserve
  ------------- ------------- -----------------------------------------------

Table 28 Definition of video alarm signs

  ---------- ----------------- ------------------------------------------
     bit        definition              Processing Instructions

      0       Video sign loss    The sign is maintained until the alarm
                   alarm                 condition is released

      1         Video sign       The sign is maintained until the alarm
              blocking alarm             condition is released

      2        Storage unit      The sign is maintained until the alarm
               failure alarm             condition is released

    3\~31         reserve      
  ---------- ----------------- ------------------------------------------

Table 29 Definition of abnormal driving behavior flag bit

+:-----:+:-------------:+:------:+:----------------------------------:+
| START | field         | data   | description and requirements       |
|       |               |        |                                    |
| BYTE  |               | type   |                                    |
+-------+---------------+--------+------------------------------------+
| 0     | Types of      | WORD   | Bitwise setting: 0 means no, 1     |
|       | Abnormal      |        | means yes                          |
|       | Driving       |        |                                    |
|       | Behaviors     |        | bitO：fatigue；                    |
|       |               |        |                                    |
|       |               |        | bit1：phone calling；              |
|       |               |        |                                    |
|       |               |        | bit2：smoke；                      |
|       |               |        |                                    |
|       |               |        | bit3\~bitlO：Reserve；             |
|       |               |        |                                    |
|       |               |        | bitll \~bit15:customize            |
+-------+---------------+--------+------------------------------------+
| 2     | fatigue level | BYTE   | The fatigue level is represented   |
|       |               |        | by 0 \~ 100, the larger the value, |
|       |               |        | the more serious the fatigue level |
+-------+---------------+--------+------------------------------------+

+-----------+----------------+------------+---------------------------------+
| **Start   | **field**      | **Data     | **Descriptions & requirements** |
| byte**    |                | length**   |                                 |
+-----------+----------------+------------+---------------------------------+
| 0         | Alarm ID       | DWORD      | According to the alarm          |
|           |                |            | sequence, it starts to          |
|           |                |            | accumulate from 0, and does not |
|           |                |            | distinguish the alarm type.     |
+-----------+----------------+------------+---------------------------------+
| 4         | sign Status    | BYTE       | > 0x00：unavailable             |
|           |                |            | >                               |
|           |                |            | > 0x01：start sign              |
|           |                |            | >                               |
|           |                |            | > 0x02：end sign                |
|           |                |            | >                               |
|           |                |            | > This field is only applicable |
|           |                |            | > to alarms or events with      |
|           |                |            | > start and end flag types. If  |
|           |                |            | > the alarm type or event type  |
|           |                |            | > does not have start and end   |
|           |                |            | > flags, this bit is not        |
|           |                |            | > available, just fill in 0x00. |
+-----------+----------------+------------+---------------------------------+
| 5         | Alarm/Event    | BYTE       | > 0x01：forward collision       |
|           | Type           |            | > warning                       |
|           |                |            | >                               |
|           |                |            | > 0x02：Lane Departure Warning  |
|           |                |            | >                               |
|           |                |            | > 0x03：car too close alarm     |
|           |                |            | >                               |
|           |                |            | > 0x04：Reserve                 |
|           |                |            | >                               |
|           |                |            | > 0x05：Frequent lane change    |
|           |                |            | > alarm                         |
|           |                |            | >                               |
|           |                |            | > 0x06：Reserve                 |
|           |                |            | >                               |
|           |                |            | > 0x07：Reserve                 |
|           |                |            | >                               |
|           |                |            | > 0x08\~0x0F：customize         |
|           |                |            | >                               |
|           |                |            | > 0x10：Reserve                 |
|           |                |            | >                               |
|           |                |            | > 0x11：Actively capture events |
|           |                |            | >                               |
|           |                |            | > 0x12\~0x1F：customize         |
+-----------+----------------+------------+---------------------------------+
| 6         | Alarm level    | BYTE       | > 0x01：First class alarm       |
|           |                |            | >                               |
|           |                |            | > 0x02：Second class alarm      |
+-----------+----------------+------------+---------------------------------+
| 7         | front vehicle  | BYTE       | > The unit is Km/h. The range   |
|           | speed          |            | > is 0\~250, only valid when    |
|           |                |            | > the alarm type is 0x01 and    |
|           |                |            | > 0x02.                         |
+-----------+----------------+------------+---------------------------------+
| 8         | front car      | BYTE       | > The unit is 100ms, the range  |
|           |                |            | > is 0\~100, and it is only     |
|           |                |            | > valid when the alarm type is  |
|           |                |            | > 0x01, 0x02 and 0x04.          |
+-----------+----------------+------------+---------------------------------+
| 9         | type of        | BYTE       | > 0x01：left deviation          |
|           | deviation      |            | >                               |
|           |                |            | > 0x02：right deviation         |
|           |                |            | >                               |
|           |                |            | > Only valid when the alarm     |
|           |                |            | > type is 0x02                  |
+-----------+----------------+------------+---------------------------------+
| 10        | Reserve        | BYTE       | > 0x01：Reserve                 |
|           |                |            | >                               |
|           |                |            | > 0x02：Reserve                 |
|           |                |            | >                               |
|           |                |            | > 0x03：Reserve                 |
|           |                |            | >                               |
|           |                |            | > Only valid when the alarm     |
|           |                |            | > type is 0x06 and 0x10         |
+-----------+----------------+------------+---------------------------------+
| 11        | Reserve        | BYTE       | > Reserve                       |
+-----------+----------------+------------+---------------------------------+
| 12        | speed          | BYTE       | > The unit is Km/h. Range       |
|           |                |            | > 0\~250                        |
+-----------+----------------+------------+---------------------------------+
| 13        | Elevation      | WORD       | > Altitude, in meters (m)       |
+-----------+----------------+------------+---------------------------------+
| 15        | latitude       | DWORD      | > Latitude value in degrees     |
|           |                |            | > multiplied by 10 to the 6th   |
|           |                |            | > power, accurate to one        |
|           |                |            | > millionth of a degree         |
+-----------+----------------+------------+---------------------------------+
| 19        | longitude      | DWORD      | > Longitude value in degrees    |
|           |                |            | > multiplied by 10 to the 6th   |
|           |                |            | > power, accurate to one        |
|           |                |            | > millionth of a degree         |
+-----------+----------------+------------+---------------------------------+
| 23        | date time      | BCD\[6\]   | > YY-MM-DD-hh-mm-ss （GMT+8     |
|           |                |            | > time）                        |
+-----------+----------------+------------+---------------------------------+
| 29        | vehicle status | WORD       | > See table 25                  |
+-----------+----------------+------------+---------------------------------+
| 31        | Alarm          | BYTE\[16\] | > The definition of alarm       |
|           | identification |            | > identification number is      |
|           | number         |            | > shown in Table 33             |
+-----------+----------------+------------+---------------------------------+

: Table 30 ADAS warning information data format

  ------------ ------------- ------------ ----------------------------------
    **start      **field**      **Data             **description**
     byte**                    length**   

       0        Terminal ID   BYTE\[7\]                Reserve

       7           Time        BCD\[6\]           YY-MM-DD-hh-mm-ss

       13      serial number     BYTE     The serial number of the alarm at
                                           the same time point, cyclically
                                                  accumulated from 0

       14      Quantities of     BYTE          Indicates the number of
                attachments                attachments corresponding to the
                                                        alarm

       15         reserve        BYTE     
  ------------ ------------- ------------ ----------------------------------

  : Table 33 Alarm identification number format

+-----------+----------------+------------+---------------------------------+
| **START   | **FIELD**      | **DATA     | **Descriptions & requirements** |
| BYTE**    |                | LENGTH**   |                                 |
+-----------+----------------+------------+---------------------------------+
| 0         | Alarm ID       | DWORD      | > According to the alarm        |
|           |                |            | > sequence, it starts from 0    |
|           |                |            | > and accumulates cyclically,   |
|           |                |            | > regardless of the alarm type. |
+-----------+----------------+------------+---------------------------------+
| 4         | sign Status    | BYTE       | > 0x00：unavailable             |
|           |                |            | >                               |
|           |                |            | > 0x01：start sign              |
|           |                |            | >                               |
|           |                |            | > 0x02：end sign                |
|           |                |            | >                               |
|           |                |            | > This field is only applicable |
|           |                |            | > to alarms or events with      |
|           |                |            | > start and end flag types. If  |
|           |                |            | > the alarm type or event type  |
|           |                |            | > does not have start and end   |
|           |                |            | > flags, this bit is not        |
|           |                |            | > available, just fill in 0x00. |
+-----------+----------------+------------+---------------------------------+
| 5         | Alarm/Event    | BYTE       | > 0x01:Fatigue driving alarm    |
|           | Type           |            | >                               |
|           |                |            | > 0x02:phone calls alarm        |
|           |                |            | >                               |
|           |                |            | > 0x03:smoke alarm              |
|           |                |            | >                               |
|           |                |            | > 0x04:Distracted Driving Alarm |
|           |                |            | >                               |
|           |                |            | > 0x05:Driver abnormal alarm    |
|           |                |            | >                               |
|           |                |            | > 0x06\~0x0F：customize         |
|           |                |            | >                               |
|           |                |            | > 0x10：Automatically capture   |
|           |                |            | > events                        |
|           |                |            | >                               |
|           |                |            | > 0x11：driver change event     |
|           |                |            | >                               |
|           |                |            | > 0x12\~0x1F：customize         |
+-----------+----------------+------------+---------------------------------+
| 6         | Alarm level    | BYTE       | > 0x01：First level alarm       |
|           |                |            | >                               |
|           |                |            | > 0x02：second level alarm      |
+-----------+----------------+------------+---------------------------------+
| 7         | fatigue level  | BYTE       | > The range is 1\~10. The       |
|           |                |            | > larger the value, the more    |
|           |                |            | > severe the fatigue, which is  |
|           |                |            | > only valid when the alarm     |
|           |                |            | > type is 0x01                  |
+-----------+----------------+------------+---------------------------------+
| 8         | reserve        | BYTE\[4\]  | > reserve                       |
+-----------+----------------+------------+---------------------------------+
| 12        | speed          | BYTE       | > The unit is Km/h. Range       |
|           |                |            | > 0\~250                        |
+-----------+----------------+------------+---------------------------------+
| 13        | Elevation      | WORD       | > Altitude, in meters (m)       |
+-----------+----------------+------------+---------------------------------+
| 15        | latitude       | DWORD      | > Latitude value in degrees     |
|           |                |            | > multiplied by 10 to the 6th   |
|           |                |            | > power, accurate to one        |
|           |                |            | > millionth of a degree         |
+-----------+----------------+------------+---------------------------------+
| 19        | longitude      | DWORD      | > Longitude value in degrees    |
|           |                |            | > multiplied by 10 to the 6th   |
|           |                |            | > power, accurate to one        |
|           |                |            | > millionth of a degree         |
+-----------+----------------+------------+---------------------------------+
| 23        | date time      | BCD\[6\]   | > YY-MM-DD-hh-mm-ss （GMT+8     |
|           |                |            | > time）                        |
+-----------+----------------+------------+---------------------------------+
| 29        | vehicle status | WORD       | > See table 5‑9                 |
+-----------+----------------+------------+---------------------------------+
| 31        | Alarm          | BYTE\[16\] | > The definition of alarm       |
|           | identification |            | > identification number is      |
|           | number         |            | > shown in Table 33             |
+-----------+----------------+------------+---------------------------------+

: Table 31 Data format of alarm information of DMS

+-----------+----------------+------------+---------------------------------+
| **START   | **FIELD**      | **DATA     | **Descriptions & requirements** |
| BYTE**    |                | LENGTH**   |                                 |
+-----------+----------------+------------+---------------------------------+
| 0         | Alarm ID       | DWORD      | > According to the alarm        |
|           |                |            | > sequence, it starts from 0    |
|           |                |            | > and accumulates cyclically,   |
|           |                |            | > regardless of the alarm type. |
+-----------+----------------+------------+---------------------------------+
| 4         | sign Status    | BYTE       | > 0x00：unavailable             |
|           |                |            | >                               |
|           |                |            | > 0x01：start sign              |
|           |                |            | >                               |
|           |                |            | > 0x02：end sign                |
|           |                |            | >                               |
|           |                |            | > This field is only applicable |
|           |                |            | > to alarms or events with      |
|           |                |            | > start and end flag types. If  |
|           |                |            | > the alarm type or event type  |
|           |                |            | > does not have start and end   |
|           |                |            | > flags, this bit is not        |
|           |                |            | > available, just fill in 0x00. |
+-----------+----------------+------------+---------------------------------+
| 5         | Alarm/Event    | BYTE       | > 0x01：rear approach alarm     |
|           | Type           |            | >                               |
|           |                |            | > 0x02：Left rear approach      |
|           |                |            | > alarm                         |
|           |                |            | >                               |
|           |                |            | > 0x03：Right rearapproach      |
|           |                |            | > alarm                         |
+-----------+----------------+------------+---------------------------------+
| 6         | speed          | BYTE       | > The unit is Km/h. Range       |
|           |                |            | > 0\~250                        |
+-----------+----------------+------------+---------------------------------+
| 7         | Elevation      | WORD       | > Altitude, in meters (m)       |
+-----------+----------------+------------+---------------------------------+
| 9         | latitude       | DWORD      | > Latitude value in degrees     |
|           |                |            | > multiplied by 10 to the 6th   |
|           |                |            | > power, accurate to one        |
|           |                |            | > millionth of a degree         |
+-----------+----------------+------------+---------------------------------+
| 13        | longitude      | DWORD      | > Longitude value in degrees    |
|           |                |            | > multiplied by 10 to the 6th   |
|           |                |            | > power, accurate to one        |
|           |                |            | > millionth of a degree         |
+-----------+----------------+------------+---------------------------------+
| 17        | date time      | BCD\[6\]   | > YY-MM-DD-hh-mm-ss             |
+-----------+----------------+------------+---------------------------------+
| 23        | vehicle status | WORD       | > See table 5‑9                 |
+-----------+----------------+------------+---------------------------------+
| 25        | Alarm          | BYTE\[16\] | > The definition of alarm       |
|           | identification |            | > identification number is      |
|           | number         |            | > shown in Table 33             |
+-----------+----------------+------------+---------------------------------+

: Table 32 Blind spot monitoring system alarm definition data format

7.12 Location information query【8201】

Message ID:0x8201.

The location information query message body is empty.

7.13 Location information query response【0201】

Message ID:0x0201.

See Table 34 for the data format of the location information query
response message body

Table 34 Location information query response message body data format

  ---------- ------------------ -------------- ---------------------------
  **START    **FIELD**          **DATA TYPE**  **Descriptions &
  BYTE**                                       requirements**

  0          Reply serial       WORD           The serial number of the
             number                            corresponding location
                                               information query message

  2          location                          For location information
             information report                reporting, see 8.12
  ---------- ------------------ -------------- ---------------------------

7.14 Text message delivery【8300】

Message ID:0x8300.

See Table 35 for the data format of the text information delivery
message body.

Table 35 Message body data format for text information delivery

  ------------ ----------- ------------ ---------------------------------
  **START      **FIELD**   **DATA       **Descriptions & requirements**
  BYTE**                   TYPE**       

  0            sign        BYTE         The meaning of the text
                                        information flag is shown in
                                        Table 36

  1            Text        STRING       Up to 1024 bytes
               message                  
  ------------ ----------- ------------ ---------------------------------

Table 36 Meaning of text information flag bits

  --------------- -------------------------------------------------------
       byte                                sign

         0                             1：emergency

       1\~2                               Reserve

         3                  1：Terminal TTS broadcast and read

       4～7                               Reserve
  --------------- -------------------------------------------------------

7.15Multimedia event information upload【0800】

message ID: 0x0800.

See Table 37 for the data format of the multimedia event message upload
message body.

Table 37 Multimedia event message upload message body data format

  ---------- -------------- ----------- ----------------------------------
  START BYTE FIELD          DATA TYPE   Descriptions & requirements

  0          Command word   DOWORD      \>0

  4          The total      BYTE        0:image;1:audio;2:video;
             length of the              
             data block                 

  5          packet data    BYTE        0：JPEG;1：TIF;other reservations
             length                     

  6          data block     BYTE        0:The platform issues an
                                        instruction; 1: timing action; 2:
                                        robbery alarm triggered; other
                                        reservations

  7          channel ID     BYTE        
  ---------- -------------- ----------- ----------------------------------

7.16 Multimedia data upload【0801】

message ID：0x0801.

The data format of the multimedia data upload message body is shown in
Table 38.

Table 38 Multimedia data message upload message body data format

+---------+-------------+----------------+----------------------------+
| START   | FIELD       | DATA TYPE      | Descriptions &             |
| BYTE    |             |                | requirements               |
+---------+-------------+----------------+----------------------------+
| 0       | multimedia  | DWORD          | > ＞0                      |
|         | ID          |                |                            |
+---------+-------------+----------------+----------------------------+
| 4       | Multimedia  | BYTE           | 0:Image；1:Audio;2:video;  |
|         | Type        |                |                            |
+---------+-------------+----------------+----------------------------+
| 5       | multimedia  | BYTE           | 0：JPEG;1：TIF;other       |
|         | format      |                | reservations               |
|         | encoding    |                |                            |
+---------+-------------+----------------+----------------------------+
| 6       | event item  | BYTE           | 0：The platform issues     |
|         | code        |                | instructions;1：timing     |
|         |             |                | action；2：robbery alarm   |
|         |             |                | triggered； other          |
|         |             |                | reservations               |
+---------+-------------+----------------+----------------------------+
| 7       | channel ID  | BYTE           |                            |
+---------+-------------+----------------+----------------------------+
| 8       | multimedia  |                |                            |
|         | packet      |                |                            |
+---------+-------------+----------------+----------------------------+

7.17 Multimedia data upload response【8800】

message ID: 0x8800

The data format of the multimedia data upload response message body is
shown in Table 39

Table 39 Multimedia data upload response message body data format

  ---------- ----------------- ------------------ ----------------------------
  START BYTE FIELD             DATA TYPE          Descriptions & requirements

  0          multimedia ID     DWORD              ＞0

  4          Total quantities  BYTE               
             of re-transmit                       
             packets                              

  5          List of                              No more than 125 items,
             re-transmission                      without this field, it means
             packet IDs                           that all data packets have
                                                  been received, each item is
                                                  2 bytes
  ---------- ----------------- ------------------ ----------------------------

7.18 The camera shoots the command immediately【8801】

message ID: 0x8801

The data format of the camera immediate shooting command message body is
shown in Table 40

Table 40 Camera Immediate Shooting Command Message Body Data Format

+--------------------+--------------------+--------------------+----------------------------------+
| START BYTE         | FIELD              | DATA TYPE          | Descriptions & requirements      |
+--------------------+--------------------+--------------------+----------------------------------+
| 0                  | channel ID         | BYTE               | ＞0                              |
+--------------------+--------------------+--------------------+----------------------------------+
| 1                  | shooting order     | WORD               | 0 means stop shooting; 0xFFFF    |
|                    |                    |                    | means video recording; other     |
|                    |                    |                    | means the number of pictures     |
|                    |                    |                    | taken                            |
+--------------------+--------------------+--------------------+----------------------------------+
| 3                  | Photo              | WORD               | The unit is seconds (s), 0 means |
|                    | interval/recording |                    | taking pictures at the minimum   |
|                    | time               |                    | interval or recording all the    |
|                    |                    |                    | time                             |
+--------------------+--------------------+--------------------+----------------------------------+
| 5                  | save sign          | BYTE               | 1：save；                        |
|                    |                    |                    |                                  |
|                    |                    |                    | 0：real-time upload              |
+--------------------+--------------------+--------------------+----------------------------------+
| 6                  | Resolution         | BYTE               | 0x01:320\*240;                   |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x02:640\*480                    |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x03:800\*600                    |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x04:1024\*768                   |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x05:176\*144;\[Qcif\]；         |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x06:352\*288\[cif\]；           |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x07:704\*288；\[HALF D1\]       |
|                    |                    |                    |                                  |
|                    |                    |                    | 0x08:704\*576；\[D1\]            |
+--------------------+--------------------+--------------------+----------------------------------+
| 7                  | Image/Video        | BYTE               | 1\~10,1 means the least quality  |
|                    | Quality            |                    | loss, 10 means the maximum       |
|                    |                    |                    | compression ratio                |
+--------------------+--------------------+--------------------+----------------------------------+
| 8                  | brightness         | BYTE               | 0\~\~255                         |
+--------------------+--------------------+--------------------+----------------------------------+
| 9                  | Contrast           | BYTE               | 0\~\~127                         |
+--------------------+--------------------+--------------------+----------------------------------+
| 10                 | saturation         | BYTE               | 0\~\~127                         |
+--------------------+--------------------+--------------------+----------------------------------+
| 11                 | Chroma             | BYTE               | 0\~\~255                         |
+--------------------+--------------------+--------------------+----------------------------------+
| If terminal A does not support the resolution required by the system, then will take the        |
| closest resolution to shoot and upload                                                          |
+-------------------------------------------------------------------------------------------------+

7.19 Stored multimedia data retrieval【8802】

message ID:0x8802.

The data format of the stored multimedia data retrieval message body is
shown in Table 41

Note: If the time range is not selected, the start time/end time are set
to 00-00-00-00-00-00.

Table 41 Stored multimedia data retrieval message body data format

  ---------- -------------- ----------- ----------------------------------
  START BYTE FIELD          DATA TYPE   Descriptions & requirements

  0          Multimedia     BYTE        0：image；1：audio；2：video
             Type                       

  1          channel ID     BYTE        0 means to retrieve all channels
                                        of this media type

  2          event item     BYTE        0: The platform issues an
             code                       instruction; 1: Timing action; 2:
                                        The robbery alarm is triggered;
                                        others reservations

  3          start time     BCD\[6\]    YY-MM-DD-hh-mm-ss

  9          End Time       BCD\[6\]    YY-MM-DD-hh-mm-ss
  ---------- -------------- ----------- ----------------------------------

7.20 Store Multimedia Data Retrieval Response【0802】

message ID:0x0802

The data format of the stored multimedia data retrieval response message
body is shown in Table 42.

Table 42 Storage multimedia data retrieval response message body data
format

  ------- ----------------- ------------------ ---------------------------
  START   FIELD             DATA TYPE          Descriptions & requirements
  BYTE                                         

  0       Reply serial      WORD               The serial number of the
          number                               corresponding multimedia
                                               data retrieval message

  2       Total number of   WORD               The total quantities of
          multimedia data                      multimedia data items that
          items                                meet the retrieval
                                               conditions

  3       search term                          See Table 43 for the data
                                               format of multimedia
                                               retrieval items
  ------- ----------------- ------------------ ---------------------------

Table 43 Data Format of Multimedia Search Items

  ---------------- ---------------- ------------ ------------------------------
  START BYTE       FIELD            DATA TYPE    Descriptions & requirements

  0                multimedia type  BYTE         0：image；1：Audio；2：Vedio

  1                channel ID       BYTE         

  2                event item code  BYTE         0: The platform issues an
                                                 instruction; 1: Timing action;
                                                 2: The robbery alarm is
                                                 triggered;others reservations

  3                Location                      Indicating the report
                   information                   information from the start of
                   report (0x0200)               the shooting or recording
                   message body                  
  ---------------- ---------------- ------------ ------------------------------

7.21 Store multimedia data upload command【8803】

message ID:0x8803.

The data format of the message body of the stored multimedia data upload
command is shown in Table 44.

Table 44 Store multimedia data upload command message body data format

  -------- -------------- ------------- ----------------------------------
  START    FIELD          DATA TYPE     Descriptions & requirements
  BYTE                                  

  0        Multimedia     BYTE          
           type                         

  1        channel ID     BYTE          

  2        event item     BYTE          0: The platform issues an
           code                         instruction; 1: Timing action; 2:
                                        The robbery alarm is
                                        triggered;others reservations

  3        start time     BCD\[6\]      YY-MM-DD-hh-mm-ss

  9        end time       BCD\[6\]      YY-MM-DD-hh-mm-ss

  15       delete sign    BYTE          0：Reserve；1：delete
  -------- -------------- ------------- ----------------------------------

7.22 Data downlink transparent transmission【8900】

message ID:0x8900.

See Table 45 for the data format of the downlink transparent
transmission message body.

Table 45 Data format of downlink transparent transmission message body

  ---------- -------------- ----------- ----------------------------------
  START BYTE FIELD          DATA TYPE   Descriptions & requirements

  0          Transparent    BYTE        customize
             Transmission               
             message type               

  1          Transparent                Different transparent transmission
             transmission               types have different meanings
             message                    
             content                    
  ---------- -------------- ----------- ----------------------------------

7.23 Data uplink transparent transmission【0900】

message ID:0x0900

See Table 46 for the data format of the data uplink transparent
transmission message body

Table 46 Data format of data uplink transparent transmission message
body

  ---------- -------------- ----------- ----------------------------------
  START BYTE FIELD          DATA TYPE   Descriptions & requirements

  0          Transparent    BYTE        
             Transmission               
             message type               

  1          Transparent                
             transmission               
             message                    
             content                    
  ---------- -------------- ----------- ----------------------------------

7.24 Query terminal audio and video attributes【9003】

message ID:0x9003

message body is empty.

7.25 Terminal upload audio and video attributes【1003】

message ID:0x1003

Packet Type: Signaling Data Packet

Use the terminal to upload the audio and video attribute command to
respond to the message for querying the terminal audio and video
attributes sent by the platform. The data format of the message body is
shown in Table 47.

Table 47 Format of audio and video attribute data uploaded by terminal

+---------+--------------------------+-------+:-----------------------+
| START   | FIELD                    | DATA  | Descriptions &         |
| BYTE    |                          | TYPE  | requirements           |
+---------+--------------------------+-------+------------------------+
| 0       | input audio codec        | BYTE  | See table 48           |
+---------+--------------------------+-------+------------------------+
| 1       | Input sound original     | BYTE  |                        |
|         | channel number           |       |                        |
+---------+--------------------------+-------+------------------------+
| 2       | Input audio sample rate  | BYTE  | > 0：8 kHz；           |
|         |                          |       | >                      |
|         |                          |       | > 1：22.05 kHz；       |
|         |                          |       | >                      |
|         |                          |       | > 2：44.1 kHz；        |
|         |                          |       | >                      |
|         |                          |       | > 3：48 kHz            |
+---------+--------------------------+-------+------------------------+
| 3       | Input audio sample bits  | BYTE  | > 0：8 位；            |
|         |                          |       | >                      |
|         |                          |       | > 1：16 位；           |
|         |                          |       | >                      |
|         |                          |       | > 2：32 位；           |
+---------+--------------------------+-------+------------------------+
| 4       | audio frame length       | WORD  | range 1\~4294967295    |
+---------+--------------------------+-------+------------------------+
| 6       | Whether to support audio | BYTE  | 0: not supported; 1:   |
|         | output                   |       | supported              |
+---------+--------------------------+-------+------------------------+
| 7       | Video encoding method    | BYTE  | see table 48           |
+---------+--------------------------+-------+------------------------+
| 8       | The maximum number of    | BYTE  |                        |
|         | audio physical channels  |       |                        |
|         | supported by the         |       |                        |
|         | terminal                 |       |                        |
+---------+--------------------------+-------+------------------------+
| 9       | The maximum number of    | BYTE  |                        |
|         | video physical channels  |       |                        |
|         | supported by the         |       |                        |
|         | terminal                 |       |                        |
+---------+--------------------------+-------+------------------------+

Table 48 Definition table of audio and video coding types

  ----------------- ------------------------- ---------------------------
        codes                 name                      remark

        0\~5                 Reserve          

          6                  G.711A                      audio

          7                  G.711U                      audio

          8                   G.726                      audio

        9\~90                Reserve          

         91         transparent transmission  

       92\~97                Reserve          

         98                   H.264                      video

       99\~101               Reserve          

      102\~110               Reserve          

      111\~127               Reserve                   customize
  ----------------- ------------------------- ---------------------------

7.26 Real-time video transmission request【9101】

message ID:0x9101

Packet type: Signaling data packet.

The platform requests real-time audio and video transmission from the
terminal device, including real-time video transmission, and actively
initiates a two-way call. The message body data format is shown in Table
49.After receiving this message, the terminal replies to the video
terminal general response, and then establishes a transmission link
through the corresponding server IP address and port number, and then
transmits the corresponding audio and video stream data according to the
audio and video stream transmission protocol.

Table 49 Real-time audio and video transmission request data format

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | Server IP   | BYTE     | LENGTH n                         |
|         | address     |          |                                  |
|         | length      |          |                                  |
+---------+-------------+----------+----------------------------------+
| 1       | Server IP   | STRING   | Real-time video server IP        |
|         | address     |          | address                          |
+---------+-------------+----------+----------------------------------+
| 1 +n    | (TCP)       | WORD     | Real-time video server TCP port  |
|         |             |          | number                           |
|         | Server      |          |                                  |
|         | video       |          |                                  |
|         | channel     |          |                                  |
|         | listening   |          |                                  |
|         | port number |          |                                  |
+---------+-------------+----------+----------------------------------+
| 3 +n    | Reserve     | WORD     | Reserve                          |
+---------+-------------+----------+----------------------------------+
| 5 +n    | logical     | BYTE     | Follow Table 12-1 in the         |
|         | channel     |          | documentation                    |
|         | number      |          |                                  |
+---------+-------------+----------+----------------------------------+
| 6       | data type   | BYTE     | 0：audio and                     |
|         |             |          | video,1：video,2：two-way talk   |
+---------+-------------+----------+----------------------------------+
| 7+n     | Stream type | BYTE     | 0: Main stream, 1: Sub stream    |
+---------+-------------+----------+----------------------------------+

7.27 Audio and video real-time transmission control【9102】

message ID:0x9102

Packet type: Signaling data packet.

The platform sends audio and video real-time transmission control
commands to switch code streams, pause code stream transmission, close
audio and video transmission channels, etc. The data format of the
message body is shown in Table 50.

Table 50 Audio and video real-time transmission control data format

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | logical     | BYTE     | Follow Table 12-1 in the         |
|         | channel     |          | documentation                    |
|         | number      |          |                                  |
+---------+-------------+----------+----------------------------------+
| 1       | Control     | BYTE     | The platform can control the     |
|         | instruction |          | real-time audio and video of the |
|         |             |          | device through this command:     |
|         |             |          |                                  |
|         |             |          | 0:Turn off the audio and video   |
|         |             |          | transmission instructions;       |
|         |             |          |                                  |
|         |             |          | 1: switch stream (add pause and  |
|         |             |          | resume);                         |
|         |             |          |                                  |
|         |             |          | 2: Pause the sending of all      |
|         |             |          | streams on this channel;         |
|         |             |          |                                  |
|         |             |          | 3: Resume the transmission of    |
|         |             |          | the stream before the            |
|         |             |          | suspension, which is the same as |
|         |             |          | the stream type before the       |
|         |             |          | suspension;                      |
|         |             |          |                                  |
|         |             |          | 4: Turn off two-way talk         |
+---------+-------------+----------+----------------------------------+
| 2       | Turn off    | BYTE     | 0: close the audio and video     |
|         | audio and   |          | data related to this channel;    |
|         | video types |          |                                  |
|         |             |          | 1: Only the audio related to     |
|         |             |          | this channel is turned off, and  |
|         |             |          | the video related to this        |
|         |             |          | channel is reserved;             |
|         |             |          |                                  |
|         |             |          | 2: Only turn off videos related  |
|         |             |          | to this channel, and retain      |
|         |             |          | audio related to this channel    |
+---------+-------------+----------+----------------------------------+
| 3       | Switch      | BYTE     | Switch the previously applied    |
|         | stream type |          | code stream to the newly applied |
|         |             |          | code stream, and the audio is    |
|         |             |          | the same as before the switch.   |
|         |             |          |                                  |
|         |             |          | The code stream of the new       |
|         |             |          | application is:                  |
|         |             |          |                                  |
|         |             |          | 0：main stream;                  |
|         |             |          |                                  |
|         |             |          | 1：substream                     |
+---------+-------------+----------+----------------------------------+

7.28

Real-time audio and video streaming and transparent data transmission
\[Video format\]

Message type: stream data message

The transmission of real-time audio and video stream data refers to the
RTP protocol, and uses the TCP bearer payload format to supplement the
message serial number, SIM card number, audio and video channel number
and other fields on the basis of the definition in IETF RFC 3550. The
definition of the payload format is shown in Table 51. The bits defined
in the table are filled in according to the big-endian mode (big-endian)

Table 51 Definition table of payload packet format of audio and video
stream and transparent data transmission protocol

+---------+-------------+-----------+----------------------------------+
| START   | FIELD       | DATA TYPE | Descriptions & requirements      |
| BYTE    |             |           |                                  |
+---------+-------------+-----------+----------------------------------+
| 0       | frame       | DWORD     | fixed as 0x30 0x31 0x63 0x64     |
|         | header      |           |                                  |
|         | identifier  |           |                                  |
+---------+-------------+-----------+----------------------------------+
| 4       | V           | 2 BITS    | fixed as2                        |
+---------+-------------+-----------+----------------------------------+
|         | P           | 1 BIT     | fixed as0                        |
+---------+-------------+-----------+----------------------------------+
|         | X           | 1 BIT     | Whether the RTP header requires  |
|         |             |           | extension bits. Fixed to 0       |
+---------+-------------+-----------+----------------------------------+
|         | CC          | 4 BITS    | fixed as1                        |
+---------+-------------+-----------+----------------------------------+
| 5       | M           | 1 BIT     | Sign bit to determine whether it |
|         |             |           | is the boundary of a complete    |
|         |             |           | data frame                       |
+---------+-------------+-----------+----------------------------------+
|         | PT          | 7 BITS    | Load type, see Table 19          |
+---------+-------------+-----------+----------------------------------+
| 6       | package     | WORD      | The initial value is 0. Each     |
|         | serial      |           | time an RTP packet is sent, the  |
|         | number      |           | sequence number is incremented   |
|         |             |           | by 1.                            |
+---------+-------------+-----------+----------------------------------+
| 8       | SIM card    | BCD\[6\]  | Terminal device SIM card number  |
|         | number      |           |                                  |
+---------+-------------+-----------+----------------------------------+
| 14      | logical     | BYTE      | Follow Table 12-1 in the         |
|         | channel     |           | documentation                    |
|         | number      |           |                                  |
+---------+-------------+-----------+----------------------------------+
| 15      | data type   | 4 BITS    | 0000：video I frame；            |
|         |             |           |                                  |
|         |             |           | 0001：video P frame；            |
|         |             |           |                                  |
|         |             |           | 0010：video B frame；            |
|         |             |           |                                  |
|         |             |           | 0011：audio frame；              |
|         |             |           |                                  |
|         |             |           | 0100：Transparent data           |
|         |             |           | transmission；                   |
+---------+-------------+-----------+----------------------------------+
|         | Subcontract | 4 BITS    | 0000：Atomic packets, cannot be  |
|         | processing  |           | split                            |
|         | sign        |           |                                  |
|         |             |           | 0001: The first packet when      |
|         |             |           | subcontracting is processed;     |
|         |             |           |                                  |
|         |             |           | 0010: The last packet during     |
|         |             |           | subcontracting processing;       |
|         |             |           |                                  |
|         |             |           | 0011: Intermediate package       |
|         |             |           | during subcontracting processing |
+---------+-------------+-----------+----------------------------------+
| 16      | timestamp   | BYTE\[8\] | Identifies the relative time of  |
|         |             |           | the current frame of this RTP    |
|         |             |           | packet, in milliseconds (ms).    |
|         |             |           | When the data type is 0100,      |
|         |             |           | there is no such field           |
+---------+-------------+-----------+----------------------------------+
| 24      | Last I      | WORD      | The time interval between this   |
|         | Frame       |           | frame and the last key frame, in |
|         | Interval    |           | milliseconds (ms), when the data |
|         |             |           | type is non-video frame, there   |
|         |             |           | is no such field                 |
+---------+-------------+-----------+----------------------------------+
| 26      | Last Frame  | WORD      | The time interval between this   |
|         | Interval    |           | frame and the previous one, in   |
|         |             |           | milliseconds (ms), when the data |
|         |             |           | type is non-video frame, there   |
|         |             |           | is no such field                 |
+---------+-------------+-----------+----------------------------------+
| 28      | data body   | WORD      | Subsequent data body length,     |
|         | length      |           | excluding this field             |
+---------+-------------+-----------+----------------------------------+
| 30      | data body   | BYTE\[n\] | Audio and video data or          |
|         |             |           | transparent data, the length     |
|         |             |           | does not exceed 950 bytes;       |
+---------+-------------+-----------+----------------------------------+

7.29 Real-time audio and video transmission status notification【9105】

message ID:0x9105

Packet type: Signaling data packet.

In the process of receiving the audio and video data uploaded by the
terminal, the platform sends notification packets to the terminal
according to the set time interval. The format of the message body data
is shown in Table 52.

Table 52 Real-time audio and video transmission status notification data
format

  ---------- -------------- ----------- ----------------------------------
  START BYTE FIELD          DATA TYPE   Descriptions & requirements

  0          logical        BYTE        Follow Table 12-1 in the
             channel number             documentation

  1          Packet loss    BYTE        The packet loss rate of the
             rate                       current transmission channel.
                                        Multiply the value by 100 and take
                                        the integer part
  ---------- -------------- ----------- ----------------------------------

7.30 Query resource list【9205】

message ID:0x9105

Packet type: Signaling data packet.

The platform queries the video file list from the terminal according to
the combined conditions of audio and video type, channel number, alarm
type and start and end time.

See Table 53 for the format of the message body data.

Table 53 Query Video File List Data Format

+---------+-------------+----------+-----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements       |
| BYTE    |             | TYPE     |                                   |
+---------+-------------+----------+-----------------------------------+
| 0       | logical     | BYTE     | According to table 12-1 in the    |
|         | channel     |          | documentation, 0 means all        |
|         | number      |          | channels                          |
+---------+-------------+----------+-----------------------------------+
| 1       | start time  | BCD\[6\] | YY-MM-DD-HH-MM-SS, all 0 means no |
|         |             |          | start time condition              |
+---------+-------------+----------+-----------------------------------+
| 7       | end time    | BCD\[6\] | YY-MM-DD-HH-MM-SS, all 0 means no |
|         |             |          | termination time condition        |
+---------+-------------+----------+-----------------------------------+
| 13      | Alarm sign  | 64BITS   | bit0\~ bit31 See Table 24         |
|         |             |          | Definition of Alarm Flag Bits；   |
|         |             |          |                                   |
|         |             |          | bit32\~bit63 see Table 13;        |
|         |             |          |                                   |
|         |             |          | All 0s indicate no alarm type     |
|         |             |          | condition                         |
+---------+-------------+----------+-----------------------------------+
| 21      | Audio and   | BYTE     | 0: audio and video, 1: audio, 2:  |
|         | video       |          | video, 3: video or audio and      |
|         | resource    |          | video                             |
|         | types       |          |                                   |
+---------+-------------+----------+-----------------------------------+
| 22      | Stream type | BYTE     | 0: All streams, 1: Main stream,   |
|         |             |          | 2: Sub streams                    |
+---------+-------------+----------+-----------------------------------+
| 23      | memory type | BYTE     | 0: all memory, 1: main memory, 2: |
|         |             |          | reserve                           |
+---------+-------------+----------+-----------------------------------+

7.31 List of audio and video resources uploaded by the terminal【1205】

message ID:0x1205

Packet Type: Signaling Data Packet

The terminal responds to the platform\'s instruction for querying the
audio and video resource list, and responds with the terminal uploading
the audio and video resource list message.If the list is too large and
needs to be subcontracted for transmission, the subcontracting mechanism
defined in 3.4.3 in the document is used for processing, and the
platform shall reply to the general response of the video platform for
each individual subcontract. The message body data format is shown in
Table 54.

Table 54 Data format of the list of audio and video resources uploaded
by the terminal

  ---------- -------------- ----------- ----------------------------------
  START BYTE FIELD          DATA TYPE   Descriptions & requirements

  0          serial number  WORD        The serial number corresponding to
                                        the command to query the audio and
                                        video resource list

  2          Total          DWORD       No audio and video resources that
             quantities of              meet the conditions, set to 0
             audio and                  
             video                      
             resources                  

  6          Audio and                  see table 55
             video resource             
             list                       
  ---------- -------------- ----------- ----------------------------------

Table 55 Format of the list of audio and video resources uploaded by the
terminal

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | logical     | BYTE     | Follow Table 12-1 in the         |
|         | channel     |          | documentation                    |
|         | number      |          |                                  |
+---------+-------------+----------+----------------------------------+
| 1       | start time  | BCD\[6\] | YY-MM-DD-HH-MM-SS                |
+---------+-------------+----------+----------------------------------+
| 7       | end time    | BCD\[6\] | YY-MM-DD-HH-MM-SS                |
+---------+-------------+----------+----------------------------------+
| 13      | Alrm sign   | 64BITS   | Bit0 \~ bit31 Define the alarm   |
|         |             |          | flag bits according to Table 24  |
|         |             |          | in the document;                 |
|         |             |          |                                  |
|         |             |          | bit32\~bit63 See table 13        |
+---------+-------------+----------+----------------------------------+
| 21      | Audio and   | BYTE     | 0: audio and video, 1: audio, 2: |
|         | video       |          | video, 3: video or audio and     |
|         | resource    |          | video                            |
|         | types       |          |                                  |
+---------+-------------+----------+----------------------------------+
| 22      | Stream type | BYTE     | 1: Main stream, 2: Sub stream    |
+---------+-------------+----------+----------------------------------+
| 23      | memory type | BYTE     | 1: main memory, 2: reserve       |
+---------+-------------+----------+----------------------------------+
| 24      | File size   | DWORD    | unit(BYTE)                       |
+---------+-------------+----------+----------------------------------+

7.32 The platform issues a remote video playback request【9201】

message ID:0x9201

Packet type: Signaling data packet.

When the platform requests audio and video video playback from the
terminal device, the terminal should respond with the 0x1205 (terminal
upload video file list) command, and then transmit the video data using
the packet format defined in Table 80 Real-time audio and video stream
data transmission RTP protocol payload format. The format of the message
body data is shown in Table 56.

Table 56 Data format of remote video playback request issued by the
platform

  ------------- ----------- ---------------------------------------------
  START BYTE    DATA TYPE   Descriptions & requirements

  0             BYTE        length n

  1             STRING      IP address of real-time audio and video
                            server

  15            WORD        Real-time audio and video server port number,
                            set to 0 when not using TCP transmission

  3 + n         WORD        Reserve

  5 + n         BYTE        Follow Table 12-1 in the documentation

  6 + n         BYTE        0: audio and video, 1: audio, 2: video, 3:
                            video or audio and video

  7 + n         BYTE        0: main stream or sub stream, 1: main stream,
                            2: sub stream; if this channel only transmits
                            audio, this field is set to 0

  8 + n         BYTE        0: Main memory, 1: Main memory, 2:Reserve

  9 + n         BYTE        0：normal playback

  10 + n        BYTE        0：invalid

  11 + n        BCD\[6\]    YY-MM-DD-HH-MM-SS,When the playback mode is
                            4, this field indicates the upload time of a
                            single frame

  17 + n        BCD\[6\]    YY-MM-DD-HH-MM-SS, if it is 0, it means
                            playback all the time. When the playback mode
                            is 4, this field is invalid.
  ------------- ----------- ---------------------------------------------

7.33 Remote video playback control issued by the platform【9202】

message ID:0x9202

Packet Type: Signaling Data Packet

During the playback of audio and video recordings by the terminal
device, the platform can issue playback control instructions to control
the playback process. The format of the message body data is shown in
Table 57.

Table 57 Remote video playback control data format issued by the
platform

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | logical     | BYTE     | Follow Table 12-1 in the         |
|         | channel     |          | documentation                    |
|         | number      |          |                                  |
+---------+-------------+----------+----------------------------------+
| 1       | playback    | BYTE     | 0：start playback；              |
|         | control     |          |                                  |
|         |             |          | 1：Pause playback；              |
|         |             |          |                                  |
|         |             |          | 2：end playback；                |
|         |             |          |                                  |
|         |             |          | 3\~4：Reserve；                  |
|         |             |          |                                  |
|         |             |          | 5：Drag to playback；            |
+---------+-------------+----------+----------------------------------+
| 2       | Fast        | BYTE     | 0：invalid；                     |
|         | forward or  |          |                                  |
|         | rewind      |          |                                  |
|         | multiples   |          |                                  |
+---------+-------------+----------+----------------------------------+
| 3       | Drag        | BCD\[6\] | YY-MM-DD-HH-MM-SS, when the      |
|         | playback    |          | playback control is 5, this      |
|         | position    |          | field is valid                   |
+---------+-------------+----------+----------------------------------+

7.34 file upload instruction【9206】

message ID:0x9206

Packet type: Signaling data packet.

The platform issues a file upload command to the terminal, and the
terminal replies with a general response and uploads the file to the
specified path of the target FTP server through FTP. The message body
data format is shown in Table 58.

Table 58 File upload instruction data format

+------------+--------------+-----------+-------------------------------+
| START BYTE | FIELD        | DATA TYPE | Descriptions & requirements   |
+------------+--------------+-----------+-------------------------------+
| 0          | server       | BYTE      | length k                      |
|            | address      |           |                               |
|            | length       |           |                               |
+------------+--------------+-----------+-------------------------------+
| 1          | server       | STRING    | FTP server address            |
|            | address      |           |                               |
+------------+--------------+-----------+-------------------------------+
| l+k        | port         | WORD      | FTP server port number        |
+------------+--------------+-----------+-------------------------------+
| 3+k        | Username     | BYTE      | 长度length l                  |
|            | length       |           |                               |
+------------+--------------+-----------+-------------------------------+
| 4+k        | Username     | STRING    | FTP Username                  |
+------------+--------------+-----------+-------------------------------+
| 4+k+l      | password     | BYTE      | length m                      |
|            | length       |           |                               |
+------------+--------------+-----------+-------------------------------+
| 5+k+l      | password     | STRING    | FTP password                  |
+------------+--------------+-----------+-------------------------------+
| 5+k+l+m    | File upload  | BYTE      | length n                      |
|            | path length  |           |                               |
+------------+--------------+-----------+-------------------------------+
| 6+k+l+m    | file upload  | STRING    | file upload path              |
|            | path         |           |                               |
+------------+--------------+-----------+-------------------------------+
| 6+k+l+m+n  | logical      | BYTE      | See Table 12-1 in the         |
|            | channel      |           | documentation                 |
|            | number       |           |                               |
+------------+--------------+-----------+-------------------------------+
| 7+k+l+m+n  | Starting     | BCD\[6\]  | YY-MM-DD-HH-MM-SS             |
|            | time         |           |                               |
+------------+--------------+-----------+-------------------------------+
| 13+k+l+m+n | End Time     | BCD\[6\]  | YY-MM-DD-HH-MM-SS             |
+------------+--------------+-----------+-------------------------------+
| 19+k+l+m+n | alarm sign   | 64BHS     | Bit0\~bit31 see Table 24 in   |
|            |              |           | the document for the          |
|            |              |           | definition of alarm flags;    |
|            |              |           |                               |
|            |              |           | bit32\~bit63 see Table 12;    |
|            |              |           |                               |
|            |              |           | All 0 means do not specify    |
|            |              |           | whether there is an alarm     |
+------------+--------------+-----------+-------------------------------+
| 27+k+l+m+n | Audio and    | BYTE      | 0: audio and video, 1: audio, |
|            | video        |           | 2: video, 3: video or audio   |
|            | resource     |           | and video                     |
|            | types        |           |                               |
+------------+--------------+-----------+-------------------------------+
| 28+k+l+m+n | Stream type  | BYTE      | 0: Main stream or sub stream, |
|            |              |           | 1: Main stream, 2: Sub stream |
+------------+--------------+-----------+-------------------------------+
| 29+k+l+m+n | storage      | BYTE      | 0: Main memory, 1: Main       |
|            | location     |           | memory, 2: Reserve            |
+------------+--------------+-----------+-------------------------------+
| 30+k+l+m+n | task         | BYTE      | bit: WIFI, when it is 1, it   |
|            | execution    |           | can be downloaded under WI-H; |
|            | conditions   |           |                               |
|            |              |           | bitl:LAN, when it is 1, it    |
|            |              |           | means that it can be          |
|            |              |           | downloaded when the LAN is    |
|            |              |           | connected;                    |
|            |              |           |                               |
|            |              |           | bit2: 3G/4G, when it is 1, it |
|            |              |           | means that it can be          |
|            |              |           | downloaded when connected to  |
|            |              |           | 3G/4G                         |
+------------+--------------+-----------+-------------------------------+

7.35 File upload completion notification【1206】

message ID:0x1206

Packet type: Signaling data packet.

When all files are uploaded through FTP, the terminal will report this
command to notify the platform. The message body data format is shown in
Table 59.

Table 59 File upload completion notification data format

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | Reply       | WORD     | The serial number corresponding  |
|         | serial      |          | to the platform file upload      |
|         | number      |          | message                          |
+---------+-------------+----------+----------------------------------+
| 2       | result      | BYTE     | 0：success；                     |
|         |             |          |                                  |
|         |             |          | 1：failure                       |
+---------+-------------+----------+----------------------------------+

7.36 File upload control【9207】

message ID:0x9207

Packet Type: Signaling Data Packet

The platform notifies the terminal to pause, resume or cancel all files
in transit. The message body data format is shown in Table 60.

Table 60 File upload control data format

+---------+----------------+----------+----------------------------------+
| START   | FIELD          | DATA     | Descriptions & requirements      |
| BYTE    |                | TYPE     |                                  |
+---------+----------------+----------+----------------------------------+
| 0       | Reply serial   | WORD     | > The serial number              |
|         | number         |          | > corresponding to the platform  |
|         |                |          | > file upload message            |
+---------+----------------+----------+----------------------------------+
| 2       | 上传控制upload | BYTE     | > 0：pause；                     |
|         | control        |          | >                                |
|         |                |          | > 1：continue；                  |
|         |                |          | >                                |
|         |                |          | > 2：cancel                      |
+---------+----------------+----------+----------------------------------+

7.37 Alarm attachment upload instruction【9208】

message ID:0x9208

Packet type: Signaling data packet.

61。After receiving the alarm/event information with attachments, the
platform sends an attachment uploading instruction to the terminal. The
data format of the instruction message body is shown in Table 61.

  ------------ ---------------- ------------ ----------------------------------
  START BYTE   FIELD            DATA TYPE    Descriptions & requirements

  0            Attachment       BYTE         length k
               server IP                     
               address length                

  1            Attachment       STRING       server IP address
               server IP                     
               address                       

  1+k          Attachment       WORD         Server port number when using TCP
               server                        transport
               port（TCP）                   

  3+k          Reserve          WORD         Reserve

  5+k          Alarm            BYTE\[16\]   The definition of the alarm
               identification                identification number is shown in
               number                        Table 4-16

  21+k         Alarm number     BYTE\[32\]   The unique number assigned by the
                                             platform to the alarm

  53+k         reserve          BYTE\[16\]   
  ------------ ---------------- ------------ ----------------------------------

  : Table 61 File upload instruction data format

After receiving the alarm attachment upload instruction issued by the
platform, the terminal sends a general response message to the platform.

7.38 Alarm attachment information message【1210】

message ID:0x1210

Packet type: Signaling data packet.

The terminal connects to the attachment server according to the
attachment upload instruction, and sends an alarm attachment information
message to the server. The data format of the message body is shown in
Table 62.

+-----------+----------------+------------+--------------------------------+
| START     | FIELD          | DATA       | Descriptions & requirements    |
| BYTE      |                | Length     |                                |
+-----------+----------------+------------+--------------------------------+
| 0         | Terminal ID    | BYTE\[7\]  | > 7 bytes, consisting of       |
|           |                |            | > uppercase letters and        |
|           |                |            | > numbers, the terminal ID is  |
|           |                |            | > defined by the manufacturer, |
|           |                |            | > if the number of digits is   |
|           |                |            | > insufficient, \"0x00\" will  |
|           |                |            | > be added after               |
+-----------+----------------+------------+--------------------------------+
| 7         | Alarm          | BYTE\[16\] | > The definition of alarm      |
|           | identification |            | > identification number is     |
|           | number         |            | > shown in Table 33            |
+-----------+----------------+------------+--------------------------------+
| 23        | Alarm number   | BYTE\[32\] | > The unique number assigned   |
|           |                |            | > by the platform to the alarm |
+-----------+----------------+------------+--------------------------------+
| 55        | type of        | BYTE       | 0x00: normal alarm file        |
|           | information    |            | information                    |
|           |                |            |                                |
|           |                |            | > 0x01: Supplementary          |
|           |                |            | > transmission of alarm file   |
|           |                |            | > information                  |
+-----------+----------------+------------+--------------------------------+
| 56        | Quantities of  | BYTE       | > Quantities of attachments    |
|           | attachments    |            | > associated with the alarm    |
+-----------+----------------+------------+--------------------------------+
| 57        | Attachment     |            | > See Table 63                 |
|           | information    |            |                                |
|           | list           |            |                                |
+-----------+----------------+------------+--------------------------------+

: Table 62 Alarm attachment information message data format

After receiving the alarm accessory information message uploaded by the
terminal, the accessory server sends a general response message to the
terminal. If the connection between the terminal and the attachment
server is abnormally disconnected in the process of uploading the alarm
attachment, the alarm attachment information message needs to be resent
when the link is restored. The attachment file in the message is the
attachment file that was not uploaded and completed before the
disconnection.

+-----------+-----------+-----------+--------------------------------+
| START     | FIELD     | DATA      | Descriptions & requirements    |
| BYTE      |           | Length    |                                |
+-----------+-----------+-----------+--------------------------------+
| 0         | file name | BYTE      | > length k                     |
|           | length    |           |                                |
+-----------+-----------+-----------+--------------------------------+
| 1         | file name | STRING    | > file name string             |
+-----------+-----------+-----------+--------------------------------+
| 1+k       | file size | DWORD     | > the size of the current file |
+-----------+-----------+-----------+--------------------------------+

: Table 63 Alarm Attachment Message Data Format

The file name naming convention is:

\<file type\>\_\<channel number\>\_\<alarm type\>\_\<serial
number\>\_\<alarm number\>.\<suffix name\>

The fields are defined as follows:

File type: 00 - picture; 01 - audio; 02 - video; 03 - text; 04 - other.

Channel number: 0\~37 indicates the video channel defined in Table 12 in
the document.

64 represents the ADAS module video channel.

65 represents the DSM module video channel.

If the attachment has nothing to do with the channel, fill in 0
directly.

Alarm type: a code consisting of the peripheral ID and the corresponding
module alarm type, for example, forward collision alarm is represented
as \"6401\".

Serial number: used to distinguish the file numbers of the same channel
and the same type.

Alarm Number: The unique number assigned by the platform to the alarm.

Suffix: jpg or png for image files, wav for audio files, h264 for video
files, and bin for text files.

After receiving the alarm accessory information instruction reported by
the terminal, the accessory server sends a general response message to
the terminal.

7.39 File information upload【1211】

message ID:0x1211

Packet type: Signaling data packet.

After the terminal sends an alarm attachment information instruction to
the attachment server and gets a response, it sends an attachment file
information message to the attachment server. The data format of the
message body is shown in Table 64.

+-----------+-----------+-----------+--------------------------------+
| START     | FIELD     | DATA      | Descriptions & requirements    |
| BYTE      |           | Length    |                                |
+-----------+-----------+-----------+--------------------------------+
| 0         | file name | BYTE      | > file name length is l        |
|           | length    |           |                                |
+-----------+-----------+-----------+--------------------------------+
| 1         | file name | STRING    | file name                      |
+-----------+-----------+-----------+--------------------------------+
| 1+l       | file type | BYTE      | > 0x00：photo                  |
|           |           |           | >                              |
|           |           |           | > 0x01：audio                  |
|           |           |           | >                              |
|           |           |           | > 0x02：video                  |
|           |           |           | >                              |
|           |           |           | > 0x03：text                   |
|           |           |           | >                              |
|           |           |           | > 0x04：others                 |
+-----------+-----------+-----------+--------------------------------+
| 2+l       | File size | DWORD     | > The size of the currently    |
|           |           |           | > uploaded file.               |
+-----------+-----------+-----------+--------------------------------+

: Table 64 Attachment file information message data format

After receiving the attachment file information instruction reported by
the terminal, the attachment server sends a general response message to
the terminal.

7.40 File data upload \[alarm attachment stream format\]

Packet type: Signaling data packet.

After the terminal sends the file information upload instruction to the
attachment server and gets a response, it sends the file data to the
attachment server. The format of the payload packet is defined in Table
65.

+-----------+------------+------------+--------------------------------+
| START     | FIELD      | DATA       | Descriptions & requirements    |
| BYTE      |            | Length     |                                |
+-----------+------------+------------+--------------------------------+
| 0         | frame      | DWORD      | > fixed as 0x30 0x31 0x63 0x64 |
|           | header     |            |                                |
|           | identifier |            |                                |
+-----------+------------+------------+--------------------------------+
| 4         | file name  | BYTE\[50\] | > file name                    |
+-----------+------------+------------+--------------------------------+
| 54        | data       | DWORD      | > The data offset of the       |
|           | offset     |            | > current transfer file        |
+-----------+------------+------------+--------------------------------+
| 58        | Data       | DWORD      | > length of payload data       |
|           | length     |            |                                |
+-----------+------------+------------+--------------------------------+
| 62        | data body  | BYTE\[n\]  | > The default length is 64K,   |
|           |            |            | > the actual length of the     |
|           |            |            | > file is less than 64K        |
+-----------+------------+------------+--------------------------------+

: Table 65 Definition of file stream payload packet format

When the attachment server receives the file stream reported by the
terminal, it does not need to reply.

7.41 File upload complete message【1212】

MESSAGE ID:0x1212

Packet type: Signaling data packet.

When the terminal completes sending a file data to the attachment
server, it sends a file sending completion message to the attachment
server. The data format of the message body is shown in Table 66.

+-----------+-----------+-----------+--------------------------------+
| START     | FIELD     | DATA      | Descriptions & requirements    |
| BYTE      |           | Length    |                                |
+-----------+-----------+-----------+--------------------------------+
| 0         | file name | BYTE      | > l                            |
|           | length    |           |                                |
+-----------+-----------+-----------+--------------------------------+
| 1         | file name | STRING    | > file name                    |
+-----------+-----------+-----------+--------------------------------+
| 1+l       | File type | BYTE      | > 0x00：photo                  |
|           |           |           | >                              |
|           |           |           | > 0x01：audio                  |
|           |           |           | >                              |
|           |           |           | > 0x02：video                  |
|           |           |           | >                              |
|           |           |           | > 0x03：text                   |
|           |           |           | >                              |
|           |           |           | > 0x04：others                 |
+-----------+-----------+-----------+--------------------------------+
| 2+l       | file size | DWORD     | > The size of the currently    |
|           |           |           | > uploaded file                |
+-----------+-----------+-----------+--------------------------------+

: Table 66 File sending complete message body data structure

7.42 File upload complete message response【9212】

message ID:0x9212

Packet type: Signaling data packet.

When the attachment server receives the file sending completion message
reported by the terminal, it sends a file uploading completion message
response to the terminal. The data structure of the response message is
shown in Table 67.

+-----------+---------------+-----------+--------------------------------+
| START     | FIELD         | DATA      | Descriptions & requirements    |
| BYTE      |               | Length    |                                |
+-----------+---------------+-----------+--------------------------------+
| 0         | file name     | BYTE      | > l                            |
|           | length        |           |                                |
+-----------+---------------+-----------+--------------------------------+
| 1         | file name     | STRING    | > file name                    |
+-----------+---------------+-----------+--------------------------------+
| 1+l       | file type     | BYTE      | > 0x00：photo                  |
|           |               |           | >                              |
|           |               |           | > 0x01：audio                  |
|           |               |           | >                              |
|           |               |           | > 0x02：video                  |
|           |               |           | >                              |
|           |               |           | > 0x03：text                   |
|           |               |           | >                              |
|           |               |           | > 0x04：others                 |
+-----------+---------------+-----------+--------------------------------+
| 2+l       | Upload        | BYTE      | > 0x00：finish                 |
|           | results       |           | >                              |
|           |               |           | > 0x01：need upload            |
|           |               |           | > supplementary                |
+-----------+---------------+-----------+--------------------------------+
| 3+l       | quantities of | BYTE      | > The quantities of data       |
|           | supplementary |           | > packets to be retransmitted, |
|           | packets       |           | > the value is 0 when there is |
|           |               |           | > no retransmission            |
+-----------+---------------+-----------+--------------------------------+
| 4+l       | List of       |           | > See table 68                 |
|           | supplementary |           |                                |
|           | packets       |           |                                |
+-----------+---------------+-----------+--------------------------------+

: Table 67 File upload complete message response data structure

+-----------+-----------+-----------+--------------------------------+
| START     | FIELD     | DATA      | Descriptions & requirements    |
| BYTE      |           | Length    |                                |
+-----------+-----------+-----------+--------------------------------+
| 0         | data      | DWORD     | > The offset of the data to be |
|           | offset    |           | > supplemented in the file     |
+-----------+-----------+-----------+--------------------------------+
| 1         | data      | DWORD     | > The length of the data to be |
|           | length    |           | > supplemented                 |
+-----------+-----------+-----------+--------------------------------+

: Table 68 Data structure of supplementary packet information

If there is data to be supplemented, the terminal should perform data
supplementary upload through file data upload, and then report the file
upload completion message after the supplementary upload is completed
until the file data is sent.

After all files are sent, the terminal actively disconnects from the
attachment server

7.43 set circular area【8600】

message ID:0x8600

See Table 69 for setting the format of the message body data in the
circular area.

Note: This message protocol supports the periodic time range. If it is
limited from 8:30 to 18:00 every day, the actual/end time is set to:
00-00-00-08-30-00/00-00-00-18:00 -00, and so on for others.

Table 69 Set circular area message body data format

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | set         | BYTE     | 0：update area；                 |
|         | Attributes  |          |                                  |
|         |             |          | 1 ：Append area；                |
|         |             |          |                                  |
|         |             |          | 2： Modify area；                |
+---------+-------------+----------+----------------------------------+
| 1       | area total  | BYTE     |                                  |
|         | quantities  |          |                                  |
+---------+-------------+----------+----------------------------------+
| 2       | area item   |          | The data format of the area item |
|         |             |          | content of the circular area is  |
|         |             |          | shown in Table 70                |
+---------+-------------+----------+----------------------------------+

Table 70 Regional Item Content Data Format for Circular Regions

  -------- ------------ ---------- ------------------------------------------
  START    FIELD        DATA TYPE  Descriptions & requirements
  BYTE                             

  0        AREA ID      DWORD      

  4        Regional     WORD       See Table 71 for the definition of area
           properties              attributes

  6        center point DWORD      Latitude value in degrees multiplied by 10
           latitude                to the 6th power, accurate to one
                                   millionth of a degree

  10       center point DWORD      Longitude value in degrees multiplied by
           longitude               10 to the 6th power, accurate to one
                                   millionth of a degree

  14       radius       DWORD      The unit is meters (m), and the road
                                   segment is from the inflection point to
                                   the next inflection point

  18       start time   BCD\[6\]   YY-MM-DD-hh-mm-ss, if the area attribute 0
                                   bit is 0, there is no such field

  24       end time     BCD\[6\]   YY-MM-DD-hh-mm-ss, if the area attribute 0
                                   bit is 0, there is no such field

  30       top speed    WORD       Km/h, if the area attribute 1 bit is 0,
                                   there is no such field

  32       Overspeed    BYTE       The unit is second (s) (similar
           Duration                expression, same as before), if the 1st
                                   bit of the area attribute is 0, there is
                                   no such field
  -------- ------------ ---------- ------------------------------------------

Table 71 Area attribute definitions for areas

  ------------------------ ----------------------------------------------
            Byte                                sign

             0                          1：according to time

             1                             1：speed limit

             2               1：Alert the driver when entering the area

             3               1：Alarm to the platform when entering the
                                                area

             4               1：Alert the driver when leaving the area

             5             1：Alarm to the platform when leaving the area

             6                  0：north latitude；1：south latitude

             7                  0：east longitude；1：west longitude

           8～15                              Reserve
  ------------------------ ----------------------------------------------

7.44 Delete circular area \[8601\] (GD02T does not support)

message ID:0x8601

Delete circular area message body data format see Table 72

Table 72 Delete circular area message body data format

  -------- ------------ --------- ----------------------------------------
  START    FIELD        DATA TYPE Descriptions & requirements
  BYTE                            

  0        area         BYTE      The area quantities included in this
           quantities             message is no more than 125, if more
                                  than 125,it is recommended to use
                                  multiple messages, 0 means to delete all
                                  circular areas

  1        area ID1     DWORD     

           ......       DWORD     

           area IDn     DWORD     
  -------- ------------ --------- ----------------------------------------

7.45 Set the rectangular area \[8602\] (GD02T does not support)

message ID:0x8602

See Table 73 for setting the format of the message body data in the
rectangular area

Table 73 Set the data format of the message body in the rectangular area

+---------+-------------+----------+----------------------------------+
| START   | FIELD       | DATA     | Descriptions & requirements      |
| BYTE    |             | TYPE     |                                  |
+---------+-------------+----------+----------------------------------+
| 0       | set up      | BYTE     | 0 ：update area；                |
|         |             |          |                                  |
|         | Attributes  |          | 1 ：Append area；                |
|         |             |          |                                  |
|         |             |          | 2 ：Modify area；                |
+---------+-------------+----------+----------------------------------+
| 1       | area        | BYTE     |                                  |
|         | quantities  |          |                                  |
+---------+-------------+----------+----------------------------------+
| 2       | Packet area |          | The area item data format of the |
|         | quantities  |          | rectangular area is shown in     |
|         |             |          | Table 50                         |
+---------+-------------+----------+----------------------------------+

7.46 Delete rectangular area \[8603\] (GD02T does not support)

message ID:0x8603

See Table 74 for the data format of the message body in the deleted
rectangular area

Table 74 Delete Rectangular Area Message Body Data Format

  -------- ------------ --------- ---------------------------------------
  START    FIELD        DATA TYPE Descriptions & requirements
  BYTE                            

  0        Area         BYTE      The area quantities included in this
           quantities             message is no more than 125, if more
                                  than 125,it is recommended to use
                                  multiple messages, 0 means to delete
                                  all rectangular areas

  1        area ID1     DWORD     

           ......       DWORD     

           area IDn     DWORD     
  -------- ------------ --------- ---------------------------------------

7.47 Set polygon area \[8604\] (GD02T does not support)

message ID:0x8604

See Table 75 for setting the data format of the polygon area message
body.

Table 75 Set polygon area message body data format

  -------- ------------ ---------- ---------------------------------------
  START    FIELD        DATA TYPE  Descriptions & requirements
  BYTE                             

  0        area ID      DWORD      

  4        area         WORD       See Table 47 for the definition of area
           attributes              attributes

  6        start time   BCD\[6\]   Same as the time range setting in the
                                   circular area

  12       end time     BCD\[6\]   Same as the time range setting in the
                                   circular area

  18       top speed    WORD       The unit is kilometers per hour (km/h),
                                   if the area attribute 1 bit is 0, there
                                   is no such field

  20       Overspeed    BYTE       The unit is seconds (s), seconds. If
           Duration                the 1st bit of the area attribute is 0,
                                   there is no such field.

  21       The total    WORD       
           quantities              
           of vertices             
           in the area             

  23       total                   The vertex item data format of the
           quantities              polygon area is shown in Table 76
           of vertices             
  -------- ------------ ---------- ---------------------------------------

Table 76 Vertex Item Data Format for Polygon Regions

  -------- ----------- --------- -----------------------------------------
  START    FIELD       DATA TYPE Descriptions & requirements
  BYTE                           

  0        Vertex      DWORD     Latitude value in degrees multiplied by
           Latitude              10 to the 6th power, accurate to one
                                 millionth of a degree

  4        Vertex      DWORD     Longitude value in degrees multiplied by
           Longitude             10 to the 6th power, accurate to one
                                 millionth of a degree
  -------- ----------- --------- -----------------------------------------

7.48 Delete polygon area \[8605\] (GD02T does not support)

message ID:0x8605

Delete polygon area message body data format see Table 77

Table 77 Delete polygon area message body data format

  -------- ------------ --------- ---------------------------------------
  START    FIELD        DATA TYPE Descriptions & requirements
  BYTE                            

  0        Area         BYTE      The area quantities included in this
           quantities             message is no more than 125, if more
                                  than 125,it is recommended to use
                                  multiple messages, 0 means to delete
                                  all rectangular areas

  1        area ID1     DWORD     

           ......       DWORD     

           area IDn     DWORD     
  -------- ------------ --------- ---------------------------------------

**Appendix A Message Comparison Table**

(Normative Appendix) Message Comparison Table

The message comparison table of the terminal communication protocol is
shown in Table A.1

Table A.1 Message comparison table

  ----- --------------------- --------- ----- ---------------------- ---------
  NO.   message body name     message   NO.   message body name      Message
                              ID                                     ID

  1     Terminal general      0x0001    24    Query terminal audio   0x9003
        Response                              and video attributes   

  2     Platform general      0x8001    25    Data uplink            0x1003
        Response                              transparent            
                                              transmission           

  3     Terminal heartbeat    0x0002    26    Real-time video        0x9101
                                              transmission request   

  4     Terminal registration 0x0100    27    Audio and video        0x9102
                                              real-time transmission 
                                              control                

  5     Terminal registration 0x8100    28    Real-time audio and    0x9105
        response                              video transmission     
                                              status notification    

  6     Terminal              0x0102    29    Query resource list    0x9205
        authentication                                               

  7     Set terminal          0x8103    30    List of audio and      0x1205
        parameters                            video resources        
                                              uploaded by the        
                                              terminal               

  8     Query terminal        0x8104    31    The platform issues a  0x9201
        parameters                            remote video playback  
                                              request                

  9     Query terminal        0x0104    32    Remote video playback  0x9202
        response parameters                   control issued by the  
                                              platform               

  10    terminal control      0x8105    33    file upload            0x9206
                                              instruction            

  11    location information  0x0200    34    File upload completion 0x1206
        report                                notification           

  12    Location information  0x8201    35    File upload control    0x9207
        query                                                        

  13    Location information  0x0201    36    Alarm attachment       0x9208
        query response                        upload instruction     

  14    Text message delivery 0x8300    37    Alarm attachment       0x1210
                                              information message    

  15    Multimedia event      0x0800    38    File information       0x1211
        information upload                    upload                 

  16    Multimedia data       0x0801    39    File upload complete   0x1212
        upload                                message                

  17    Multimedia data       0x8800    40    File upload complete   0x9212
        upload response                       message response       

  18    The camera shoots the 0x8801    41    set circular area      0x8600
        command immediately                                          

  19    Stored multimedia     0x8802    42    delete circular area   0x8601
        data retrieval                                               

  20    Store Multimedia Data 0x0802    43    set rectangular area   0x8602
        Retrieval Response                                           

  21    传Store multimedia    0x8803    44    delete rectangular     0x8603
        data upload                           area                   

  22    Data downlink         0x8900    45    set polygon area       0x8604
        transparent                                                  
        transmission                                                 

  23    Data uplink           0x0900    46    delete polygon area    0x8605
        transparent                                                  
        transmission                                                 

                                                                     

                                                                     

                                                                     

                                                                     
  ----- --------------------- --------- ----- ---------------------- ---------

**Appendix B BSJ Upstream Extension Instructions**

**BSJ extended instruction format:**

  ------------- ----------- --------------------------------------------------
      field      data type              Descriptions &requirements

     length        WORD        2 bytes, the length includes the instruction
                                       length plus the data length

   instruction     WORD                           2bytes

      data                  
  ------------- ----------- --------------------------------------------------

+:------------------:+:-------------------:+:-------------------:+:------------------------------------:+
| name               | length              | instruction         | data                                 |
+--------------------+---------------------+---------------------+--------------------------------------+
| Occupied bytes     | 2                   | 2                   | 3                                    |
+--------------------+---------------------+---------------------+--------------------------------------+
| Fuel consumption   | 0x0005              | 0x0001              | 0x00 0x00 0x00                       |
| data segment       |                     |                     |                                      |
+--------------------+---------------------+---------------------+--------------------------------------+
| explanation        | The fuel consumption value is in ohms, the first and second bytes represent      |
|                    | hexadecimal integers, and the third byte represents                              |
|                    |                                                                                  |
|                    | Number) For example: 0x01, 0x2A, 0x06 means 298.6 ohms                           |
+--------------------+----------------------------------------------------------------------------------+
| Example: The time it takes for the terminal to connect to the oil, the returned location data:        |
|                                                                                                       |
| [00 05 00 01 00 01 01]{.underline}                                                                    |
+-------------------------------------------------------------------------------------------------------+
|                                                                                                       |
+--------------------+---------------------+---------------------+--------------------------------------+
| name               | length              | instruction         | data                                 |
+--------------------+---------------------+---------------------+--------------------------------------+
| Occupied bytes     | 2                   | 2                   | 8                                    |
+--------------------+---------------------+---------------------+--------------------------------------+
| Temperature        | 0x000A              | 0x0003              | [VN1]{.underline} [VN2]{.underline}  |
| control data       |                     |                     | [VN3]{.underline} [VN4]{.underline}  |
| section            |                     |                     |                                      |
+--------------------+---------------------+---------------------+--------------------------------------+
| explanation        | 00 0A 00 03 80 00 0B 00 90 00 FF 00                                              |
|                    |                                                                                  |
|                    | Temperature control can support access to 4chanel temperature sensors            |
|                    |                                                                                  |
|                    | The 8-byte data represent the temperature values of 1, 2, 3, and 4 routes        |
|                    | respectively, and each route consists of 2 bytes                                 |
|                    |                                                                                  |
|                    | The temperature value z is in degrees, the first byte V is the degree, and the   |
|                    | second byte N is reserved                                                        |
|                    |                                                                                  |
|                    | V The data type is a signed integer, and the initialization value is 0xFF. When  |
|                    | the temperature probe is connected, it will change, corresponding to the         |
|                    | temperature value, such as -1 or -1 degree.                                      |
|                    |                                                                                  |
|                    | 00 0A 00 03 80 00 0B 00 90 00 FF 00                                              |
+--------------------+----------------------------------------------------------------------------------+
| Example:                                                                                              |
|                                                                                                       |
| When the terminal is connected to temperature control:                                                |
|                                                                                                       |
| [00 0A 00 03 80 00 0B 00 90 00 FF 00]{.underline}                                                     |
|                                                                                                       |
| Indicates that the first channel is 0 degrees, the second channel is 11 degrees, the third channel is |
| -16 degrees, and the fourth channel has no temperature probe.                                         |
+-------------------------------------------------------------------------------------------------------+
|                                                                                                       |
+-------------------------------------------------------------------------------------------------------+
