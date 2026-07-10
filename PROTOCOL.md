# JL-IDD-Z4 Integrated Access Controller — Development Manual 3: TCP/UDP Protocol

**Source:** http://www.ykt1.cn/news261.html
**Vendor:** Guangzhou Qidun Electronics Technology Co., Ltd.
**Applies to:** JL-IDD-Z4 series controllers, S4A ACB-001/002/004, and compatible OEM devices

---

## 1 Document Overview

This document describes direct protocol communication with the controller without using an SDK. Only a subset of functionality is exposed; no technical support is provided; backward compatibility with future devices or SDKs is not guaranteed.

### 1.1 Special Notes

- Card numbers below 255 are reserved for events; IC/ID cards with numbers below 255 cannot be used.
- Card number `666666` represents a simulated swipe (triggered by a configured signal).
- Card number `7777777` represents a gate timeout (swipe without passing through the gate within the configured window).
- Card number `111111111111111110` represents all national ID cards; granting access to this number grants access to all ID cards.

---

## 4 TCP/UDP Protocol Interaction

### Transport Modes

| Mode       | Controller Side                     | Server Side                           | Notes                                               |
| ---------- | ----------------------------------- | ------------------------------------- | --------------------------------------------------- |
| UDP        | Listens on port 65534 for commands  | Listens on port 50000 for events      | Controller sends events directly to server IP:50000 |
| TCP Client | Controller connects to server:50000 | Server listens on port 50000          | Most common mode                                    |
| TCP Server | Controller listens on port 50000    | Software connects to controller:50000 | Rare                                                |

Data format is identical for UDP and TCP. For TCP, messages are prefixed with a 4-byte big-endian length field.

### Event Header (RptCmdHead)

```c
typedef struct RptCmdHead {
    unsigned char flag;   // Fixed: 0xc8
    unsigned char type;   // 1=event 2=card swipe 3=ID card 4=heartbeat 5=debug 6=signal change 7=operation log 8=pull auth request 9=auth change result 10=get time
    unsigned char lenH;   // Data length high byte
    unsigned char lenL;   // Data length low byte
    unsigned int sn;       // Sequence number (4 bytes)
} RptCmdHead;
```

**Note:** Real-time card swipe reports do NOT use RptCmdHead. They are sent directly as pipe-delimited strings.

---

### 4.1.1.1 Real-time Card Swipe Report (TCP/UDP)

```
[4-byte TCP length prefix]CardData|Type|ControllerID|Result|Time|ReaderNo|DoorNo|Direction|DeviceName|LogType|LogSubType|IDCardChip
```

Example: `0008242637|1|26419|4|2015-04-30 15:20:25|5|1|1|26419|||`

Field descriptions:

- **CardData:** IC card number or QR code data. Social security card format: `CardNumber-Name-IDNumber`. Bank card format: `First6Last4Digits-VoucherNo-ReferenceNo-TransactionTime-Amount-ReadMethod-BatchNo`.
- **National ID card report:** 14-byte prefix (AA AA AA...) + 256 bytes of UNICODE text (Name 30B / Gender 2B / Ethnicity 4B / Birth Date 16B / Address 70B / ID Number 36B / Issuing Authority 30B / Expiry 32B) + 1024 bytes photo data (starts with "WLf") + 1 byte reserved.
- **Type:** 1 = barcode/card, 2 = national ID card.
- **Result:** Controller verification result. See Appendix 6.2 Error Codes (0 = success, 4 = no permission, etc.).
- **ReaderNo:** 1–8 (1–4 = serial ports, 5–8 = Wiegand ports).
- **DoorNo:** 1–4.
- **Direction:** 1 = entry, 2 = exit.

---

### 4.1.1.2 Heartbeat / Connection Detection Message

Format: `[4-byte TCP length prefix] + RptCmdHead (8 bytes, type=4) + pipe-delimited payload`

```
c8 04 00 30 00 00 00 00 | ControllerFlag|TimeoutConfig|TimeoutRemaining|TimeoutCount|ControllerName|GlobalFlag|FirmwareVersion
```

**Heartbeat ACK (33 bytes):**

```
55 55 55 55 55 55 55 55 55 55 55 55 55 55 55 55
ff ff ff ff ff ff ff ff ff ff ff ff 70 00 00 00 70
```

---

### 4.1.1.3 Signal Change Message (TCP/UDP)

Format: `RptCmdHead (type=6) + pipe-delimited payload`

```
PrevSignals;CurrSignals|ControllerFlag|Time|Config1|Config2|Config3|Config4|Config5|Config6|Config7|Config8|DeviceName|PeerAddr
```

Signal values are bitmasks representing S1–S8 terminal states. 255 (11111111b) = all terminals open/disconnected.

---

### 4.1.2 Asynchronous Log Report

The controller sends historical log records one by one. An acknowledgment message (ACK) must be sent before the controller proceeds to the next record.

**Log ACK Format (37 bytes):**

```
55 55 55 55 55 55 55 55 55 55 55 55 55 55 55 55
ff ff ff ff ff ff ff ff [DeviceID 2B BE] [Seq 2B LE]
44 00 00 04 [LogSeq 4B LE] [Checksum 1B]
```

Checksum: low byte of the sum of all bytes from the command byte (0x44) through the end of the message data.

---

## 4.2 Software Request Command Interface

### General Frame Format

**Request:**

```
55 55 55 55 55 55 55 55 55 55 55 55 55 55 55 55  (16 bytes)
ff ff ff ff ff ff ff ff                           (8 bytes)
[DeviceID 2B BE] [Seq 2B LE] [Cmd 1B] [0x00] [DataLen 2B BE] [Data N bytes] [Checksum 1B]
```

**Response:**

```
aa aa aa aa aa aa aa aa aa aa aa aa aa aa aa aa  (16 bytes)
ff ff ff ff ff ff ff ff                           (8 bytes)
[0xffff] [Seq 2B LE] [Cmd+1 1B] [Result 1B] [DataLen 2B BE] [Data N bytes] [Checksum 1B]
```

**Checksum:** Low byte of the sum of all bytes from the command byte through the end of the data.

- DeviceID can be set to `0xFFFF` for compatibility mode.
- Result `0x01` = success.
- The response command byte is always the request command byte + 1.

---

### 4.2.1 Remote Open Door (Cmd=0x10)

Data: 3 bytes — `[Door 1B] [DurationHi 1B] [DurationLo 1B]`

Duration is in units of **10 milliseconds**. 300 = 3 seconds; 30 = 300 ms; 0 = use the relay's default duration.

#### Open Door Examples (DeviceID=0xFFFF, Seq=4)

| Action            | Last 3 data bytes + checksum |
| ----------------- | ---------------------------- |
| Door 1, 3 seconds | 01 01 2c 41                  |
| Door 1, 5 seconds | 01 01 f4 09                  |
| Door 1, 300 ms    | 01 00 1e 32                  |
| Door 2, 3 seconds | 02 01 2c 42                  |
| Door 3, 3 seconds | 03 01 2c 43                  |
| Door 4, 3 seconds | 04 01 2c 44                  |

#### Special Relay Control

| Operation                      | Duration param (ms) | Data bytes (door+duration) |
| ------------------------------ | ------------------- | -------------------------- |
| Restore autonomous control     | 650010              | 01 fd e9                   |
| Keep open (relay disconnected) | 650020              | 01 fd ea                   |
| Keep closed (relay connected)  | 650030              | 01 fd eb                   |
| Close relay only               | 650040              | 01 fd ec                   |
| Open relay only                | 650050              | 01 fd ed                   |

> **Note:** The checksum values for special relay control in the original documentation may contain errors. Compute the correct checksum as: `sum(cmd=0x10, result=0x00, dataLen=0x0003, door, 0xfd, 0xe9) mod 256`.

---

### 4.2.2 Authorize (Cmd=0x12)

Data: 24 bytes (see Appendix 6.5 Authorization Structure). Response cmd = 0x13.

### 4.2.3 Revoke Authorization (Cmd=0x14)

Data: 8 bytes — `[CardHigh 4B LE] [CardLow 4B LE]`. Response cmd = 0x15.

### 4.2.4 Query Authorization (Cmd=0x34)

Data: 8 bytes — `[Position 4B, starts at 1] [Fixed: 1 4B]`. Response data = 24-byte authorization structure. Response cmd = 0x35.

### 4.2.5 Clear All Authorizations (Cmd=0x18)

Data: 0 bytes. Response cmd = 0x19.

### 4.2.6 Active Monitor & Log Extraction (Cmd=0x38)

Data: 4 bytes LE — log index position (0 = latest record). Response cmd = 0x39.

Response data: 48 bytes:

```
[CurrentRecordSeq 4B] [LogDetail 16B: CardHigh CardLow Date Time Door/Reader Result Dir/Type SubType]
[LogCount 4B] [AuthCount 4B]
[CurrentTime 7B BCD: Year(offset from 2000) Month Day Hour Minute Second Weekday(0=Sun)]
[ReaderRelay 8B] [DeviceFlag 5B ASCII]
```

### 4.2.7 Set Time (Cmd=0x26)

Data: 7 bytes BCD — `[Year-2000] [Month] [Day] [Hour] [Minute] [Second] [Weekday: 0=Sun 1=Mon ... 6=Sat]`. Response cmd = 0x27.

### 4.2.8 Run Text Command (Cmd=0x94)

Data: 520 bytes — first 8 bytes are 0x00, followed by 512 bytes of command string (zero-padded if shorter). Response cmd = 0x95.

---

## 6 Appendices

### 6.2 Controller Error Codes

| Code | Meaning                  | Code | Meaning                          |
| ---- | ------------------------ | ---- | -------------------------------- |
| 0    | Success                  | 18   | Invalid sync type                |
| 2    | Schedule error           | 19   | Invalid sync message format      |
| 3    | Exceeded limit           | 20   | Sync data limit                  |
| 4    | No permission            | 21   | Invalid sync data count/sequence |
| 5    | Reader error             | 22   | Network state unknown            |
| 6    | Expired                  | 23   | Network disconnected             |
| 7    | Work mode disabled       | 24   | Network restored                 |
| 8    | Internal error           | 25   | Network check — reboot device    |
| 9    | Number decode failed     | 26   | Network check — reboot chip      |
| 10   | Gate timeout             | 27   | Anti-collision                   |
| 11   | Anti-passback            | 28   | Manual lock                      |
| 12   | Not supported            | 29   | Multi-door interlock             |
| 13   | Unknown error            | 30   | Card read/write failed           |
| 14   | Failed                   | 31   | Group ID error                   |
| 16   | Not registered / expired | 32   | System status detail             |
| 17   | Password error           | 33   | Blacklist                        |
| 34   | Storage error            | 37   | Age restriction                  |
| 35   | Not authorized           | 38   | ID expired                       |
| 36   | Too many people inside   |      |                                  |

### 6.3 Date & Time Structure (BCD, 2 bytes each, little-endian on wire)

**Date:** `(Year - 2000) * 512 + Month * 32 + Day`

Example: 2018-04-16 → 0x2490, transmitted as bytes `90 24` (little-endian).

**Time:** `Hour * 2048 + Minute * 32 + Second / 2`

Example: 11:40:26 → 0x5D0D, transmitted as bytes `0D 5D` (little-endian).

### 6.4 Log Structure (xLog, 16 bytes)

```c
typedef struct xLog {
    unsigned int high;       // Card number high (4B LE)
    unsigned int low;        // Card number low (4B LE)
    xDate rDate;             // Date (2B)
    xTime rTime;             // Time (2B)
    unsigned char door:3;    // Door number 1–4
    unsigned char reader:5;  // Reader 1–4=serial, 5–8=Wiegand
    unsigned char result;    // Verification result (see Error Codes)
    unsigned char dir:2;     // Direction 1=entry 2=exit 0=unknown
    unsigned char type:6;    // 1=event 2=card swipe log 3=operation log
    unsigned char isName:1;
    unsigned char subType:5;
    unsigned char extReader:2;
} xLog;  // 16 bytes total
```

Example: `00 00 00 00 20 fb 6e 20 9c 24 37 5d 29 04 09 00`

- Card number = 0544144160
- Time = 2018-04-28 11:41:46
- Door = 1, Reader = 5, Result = 4 (No permission), Direction = entry, Type = card swipe log

### 6.5 Authorization Structure (xRight, 24 bytes, all fields little-endian)

```c
typedef struct xRight {
    unsigned int high;       // Card number high (4B)
    unsigned int low;        // Card number low (4B)
    xDate bDate;             // Begin date (2B)
    xTime bTime;             // Begin time (2B)
    xDate eDate;             // End date (2B)
    xTime eTime;             // End time (2B)
    unsigned char tz;        // Timezone (0 = any, bitmask for 8 zones)
    unsigned char reader;    // Allowed readers (255 = all, bitmask)
    unsigned short remain;   // Remaining count (65535 = unlimited, 1–59999 = total count, 60000+ = directional count)
    unsigned int isName:1;   // Use name as card number
    unsigned int hasPackage:1;
    unsigned int hasDebt:1;
    unsigned int hasFlag1:1;
    unsigned int hasFlag2:1;
    unsigned int hasFlag3:1;
    unsigned int hasAntiback:1; // Anti-passback flag
    unsigned int reserved:16;
    unsigned int grp:3;      // Person group
    unsigned int pos:2;      // Person position
    unsigned int manType:4;  // Person type
} xRight;  // 24 bytes total
```

---

## Related Documents

- JL-IDD-Z4 Development Manual 1 — Overview
- JL-IDD-Z4 Development Manual 2 — Using the SDK
- JL-IDD-Z4 Development Manual — HTTP Protocol Interaction
