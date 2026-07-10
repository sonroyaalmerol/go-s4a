# go-s4a

Go SDK for S4A ACB series door access controllers (JL-IDD-Z4 OEM protocol).

```
github.com/sonroyaalmerol/go-s4a
```

Requires Go 1.26. MIT licensed.

## Protocol

UDP binary protocol on two ports:

| Direction            | Port  | Description                                             |
| -------------------- | ----- | ------------------------------------------------------- |
| Controller to server | 50000 | Events (card swipes, heartbeat, signal change, logs)    |
| Server to controller | 65534 | Commands (open door, manage cards, set time, poll logs) |

TCP transport is also supported with a 4-byte big-endian length prefix.

All frames share a common structure:

```
Request:  55 55 55 55 55 55 55 55 55 55 55 55 55 55 55 55  ff ff ff ff ff ff ff ff  [deviceID 2B BE] [seq 2B LE] [cmd 1B] [0x00] [dataLen 2B BE] [data NB] [cksum 1B]
Response: aa aa aa aa aa aa aa aa aa aa aa aa aa aa aa aa  ff ff ff ff ff ff ff ff  [0xFFFF] [seq 2B LE] [cmd+1 1B] [result 1B] [dataLen 2B BE] [data NB] [cksum 1B]
```

Checksum: low byte of sum of bytes from cmd through end of data.

## Commands

| Cmd    | Response | Function                                                           |
| ------ | -------- | ------------------------------------------------------------------ |
| `0x10` | `0x11`   | Open door (door number + duration in 10ms units)                   |
| `0x12` | `0x13`   | Add card authorization (24-byte xRight structure)                  |
| `0x14` | `0x15`   | Revoke card authorization (8-byte card number)                     |
| `0x18` | `0x19`   | Clear all authorizations                                           |
| `0x34` | `0x35`   | Query authorization (paginated, 24-byte xRight response)           |
| `0x38` | `0x39`   | Poll monitor/log (48-byte response with log entry + counts + time) |
| `0x26` | `0x27`   | Set time (7 bytes BCD)                                             |
| `0x94` | `0x95`   | Text command (520 bytes, 512-byte command string)                  |

## Events

Events arrive on port 50000. Two formats:

Real-time card swipe (pipe-delimited, no header):

```
card_data|type|controller_id|result|time|reader|door|direction|name|log_type|log_subtype|chip
```

All other events have an 8-byte RptCmdHead:

```
[c8] [type 1B] [dataLen 2B BE] [seq 4B LE] [payload NB]
```

| Type | Name                   | Payload format                                                              |
| ---- | ---------------------- | --------------------------------------------------------------------------- |
| 1    | General event          | `Pipe-delimited`                                                            |
| 2    | Card swipe (async log) | `Pipe-delimited`                                                            |
| 3    | ID card                | `14+256+1024+1 bytes binary`                                                |
| 4    | Heartbeat              | `flag\|timeout_cfg\|timeout_remain\|timeout_cnt\|name\|global_flag\|fw_ver` |
| 5    | Debug                  | `Pipe-delimited`                                                            |
| 6    | Signal change          | `prev;curr\|flag\|time\|config1..8\|name\|peer_addr`                        |
| 7    | Operation log          | `Pipe-delimited`                                                            |
| 8    | Pull auth request      | `Pipe-delimited`                                                            |
| 9    | Auth change result     | `Pipe-delimited`                                                            |
| 10   | Get time               | `Pipe-delimited`                                                            |

Heartbeat requires ACK (29 bytes). Async log records require ACK (37 bytes).

## Usage

### Low-level client (wire protocol)

```go
client, _ := s4a.NewClient("10.254.33.10:65534")
defer client.Close()

// Open door 1 for 3 seconds
client.OpenDoor(ctx, 1, 300)

// Add a card with 24/7 access to all readers
right := &s4a.AuthRight{
    CardLow:     1234567890,
    BeginDate:   s4a.BCDDateEncode(2025, 1, 1),
    EndDate:     s4a.BCDDateEncode(2030, 12, 31),
    EndTime:     s4a.BCDTimeEncode(23, 59, 58),
    ReaderMask:  0xFF,
    RemainCount: 0xFFFF,
}
client.Authorize(ctx, right)
```

### Event listener

```go
listener, _ := s4a.NewEventListener(":50000")
defer listener.Close()

listener.ListenEvents(ctx, func(evt *s4a.Event) error {
    switch evt.Type {
    case s4a.EventTypeCardSwipe:
        fmt.Printf("card %s at door %s\n", evt.CardData, evt.DoorNo)
    case s4a.EventTypeHeartbeat:
        // send HeartbeatACK back to controller
    }
    return nil
})
```

### Controller wrapper (stateful)

```go
ct, _ := s4a.NewController("10.254.33.10:65534")
defer ct.Close()

// Fetch current status
ct.RefreshInfo(ctx)
info := ct.Info()
fmt.Printf("device %s, fw %s, %d cards, %d records\n",
    info.SerialNum, info.FirmwareVer, info.AuthCount, info.LogCount)

ct.SyncTime(ctx)
ct.OpenDoor(ctx, 1, 300)
```

### Multi-controller system (inter-lock, multi-card, first-card-open)

```go
sys := s4a.NewSystem()
defer sys.Shutdown()

ct1, _ := s4a.NewController("10.254.33.10:65534")
ct2, _ := s4a.NewController("10.254.33.11:65534")
sys.AddController(ct1)
sys.AddController(ct2)

// Door 1 on ct1 and door 1 on ct2 can never be open simultaneously
sys.EnableInterlock("10.254.33.10:65534", 1, "10.254.33.11:65534", 1)

// Require 3 different cards to open door 2
sys.EnableMultiCard("10.254.33.10:65534", 2, 3)

// Door stays unlocked after first authorized card
sys.EnableFirstCardOpen("10.254.33.10:65534", 1)

// Feed card swipe events through the system to trigger multi-card and first-card-open
sys.HandleCardSwipe(ctx, "10.254.33.10:65534", evt)
```

### Discovery

```go
controllers, _ := s4a.Discover(ctx, 3*time.Second)
for _, c := range controllers {
    fmt.Printf("%s at %s\n", c.String())
}
```

Discovery listens on port 50000 for heartbeat broadcasts. Controllers must have option 08 (network connection detection) enabled.

### Special relay control

Duration field is in 10ms units. Special values control relay state:

```go
client.OpenDoor(ctx, 1, s4a.OpenDoorRestoreAuto) // restore normal control
client.OpenDoor(ctx, 1, s4a.OpenDoorKeepOpen)    // relay disconnected until restored
client.OpenDoor(ctx, 1, s4a.OpenDoorKeepClosed)   // relay connected until restored
```

### BCD date/time encoding

```go
// Date: (year-2000)*512 + month*32 + day
// Time: hour*2048 + minute*32 + second/2
date := s4a.BCDDateEncode(2025, 6, 15)  // 0x324f
time := s4a.BCDTimeEncode(14, 30, 0)     // 0x3c40
```

## Authorization structure (24 bytes)

```
Offset  Size  Field        Description
0       4     CardHigh      Card number high word (LE), 0 for standard IC cards
4       4     CardLow       Card number low word (LE)
8       2     BeginDate     BCD date, activation start
10      2     BeginTime     BCD time, activation start
12      2     EndDate       BCD date, activation end
14      2     EndTime       BCD time, activation end
16      1     TimeZone      0=any, bitmask for 8 zones
17      1     ReaderMask    255=all, bitmask for readers 1-8
18      2     RemainCount   65535=unlimited, 1-59999=count, 60000+=directional
20      2     Flags         Bit: isName, hasAntiback, etc.
22      1     Group/Pos/Type Bits 0-2=group, 3-4=position, 5-7=person type
23      1     Reserved      0
```

## Log entry structure (16 bytes)

```
Offset  Size  Field        Description
0       4     CardHigh      Card number high word (LE)
4       4     CardLow       Card number low word (LE)
8       2     Date          BCD date
10      2     Time          BCD time
12      1     Door/Reader   Bits 0-2=door (1-4), bits 3-7=reader (1-4 serial, 5-8 Wiegand)
13      1     Result        Error code (0=success, 4=no permission, etc.)
14      1     Dir/Type      Bits 0-1=direction (1=in, 2=out), bits 2-7=type (1=event, 2=card, 3=op)
15      1     SubType/Ext   Bits 0-4=subtype, bits 5-6=ext reader
```

## Monitor/log response structure (48 bytes)

```
Offset  Size  Field        Description
0       4     LogSeq        Current record index
4       16    LogDetail     16-byte xLog structure (see above)
20      4     LogCount      Total log records
24      4     AuthCount     Total authorizations
28      7     CurrentTime   BCD: year-2000, month, day, hour, minute, second, weekday
35      8     ReaderRelay   Reader direction and relay state
43      5     DeviceFlag    ASCII device serial number
```

## Error codes

0=Success, 2=Schedule error, 3=Exceeded limit, 4=No permission, 5=Reader error,
6=Expired, 7=Work mode disabled, 8=Internal error, 9=Number decode failed,
10=Gate timeout, 11=Anti-passback, 12=Not supported, 13=Unknown error, 14=Failed,
16=Not registered/expired, 17=Password error, 18=Invalid sync type,
19=Invalid sync message format, 20=Sync data limit, 21=Invalid sync data count,
22=Network state unknown, 23=Network disconnected, 24=Network restored,
25=Network check reboot device, 26=Network check reboot chip, 27=Anti-collision,
28=Manual lock, 29=Multi-door interlock, 30=Card read/write failed,
31=Group ID error, 32=System status detail, 33=Blacklist, 34=Storage error,
35=Not authorized, 36=Too many people inside, 37=Age restriction, 38=ID expired.

## Protocol reference

Full protocol documentation: `PROTOCOL.md`

Source: JL-IDD-Z4 Integrated Access Controller Development Manual 3 (ykt1.cn)
Applies to: S4A ACB-001, ACB-002, ACB-004 and compatible OEM controllers.

## S4A Software Feature Equivalents

Every operation the Windows S4A Access Control software performs:
https://github.com/sonroyaalmerol/go-s4a

### Add Controller + Set IP

S4A software: Configuration > Controllers > Search > Configure IP

```go
// Discover controllers on the network
controllers, _ := s4a.Discover(ctx, 3*time.Second)
for _, c := range controllers {
    fmt.Println(c.String())
}

// Configure a discovered controller's network via text command
tc := s4a.NewTextCommand().
    SetIP("10.254.33.10").
    SetMask("255.255.255.0").
    SetGateway("10.254.33.1").
    SetIPMode(s4a.IPModeUDP).
    SetReportIP("10.254.33.14").
    SetReportPort(50000).
    SetName("FrontDoor")
f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client := ... // see below
client.sendAndWait(ctx, f)
```

### Open a Door

S4A software: Operation > Console > Remote Open

```go
client, _ := s4a.NewClient("10.254.33.10:65534")
defer client.Close()

// Door 1, 3 seconds
client.OpenDoor(ctx, 1, 300)

// Door 2, 500ms (turnstile)
client.OpenDoor(ctx, 2, 50)

// Keep door unlocked indefinitely
client.OpenDoor(ctx, 1, s4a.OpenDoorKeepOpen)
// Restore normal operation
client.OpenDoor(ctx, 1, s4a.OpenDoorRestoreAuto)
```

### Add a Card

S4A software: Configuration > Personnel > Add user, then Configuration > Access Privilege > Upload

```go
right := &s4a.AuthRight{
    CardLow:     1234567890,
    BeginDate:   s4a.BCDDateEncode(2025, 1, 1),
    EndDate:     s4a.BCDDateEncode(2030, 12, 31),
    EndTime:     s4a.BCDTimeEncode(23, 59, 58),
    ReaderMask:  0xFF,       // all readers
    RemainCount: 0xFFFF,     // unlimited
    TimeZone:    0,          // any time
    Flags:       0,          // no anti-passback
}
client.Authorize(ctx, right)
```

### Card Lost / Replace

S4A software: Configuration > Personnel > Card Lost

```go
// Revoke the lost card
client.RevokeAuth(ctx, 0, 1234567890)

// Issue a new card
newRight := &s4a.AuthRight{
    CardLow:     9876543210,
    BeginDate:   s4a.BCDDateEncode(2025, 1, 1),
    EndDate:     s4a.BCDDateEncode(2030, 12, 31),
    EndTime:     s4a.BCDTimeEncode(23, 59, 58),
    ReaderMask:  0xFF,
    RemainCount: 0xFFFF,
}
client.Authorize(ctx, newRight)
```

### Time-Based Access (Time Profile)

S4A software: Configuration > Time Profile > Assign to user

```go
// Mon-Fri 08:30-17:30, no weekends
right := &s4a.AuthRight{
    CardLow:     1234567890,
    BeginDate:   s4a.BCDDateEncode(2025, 1, 1),
    BeginTime:   s4a.BCDTimeEncode(8, 30, 0),
    EndDate:     s4a.BCDDateEncode(2025, 12, 31),
    EndTime:     s4a.BCDTimeEncode(17, 30, 0),
    TimeZone:    2,  // timezone 2 (user-defined in config tool)
    ReaderMask:  0xFF,
    RemainCount: 0xFFFF,
}
client.Authorize(ctx, right)
```

### Anti-passback

S4A software: Configuration > Anti-passback

```go
// Set the anti-passback bit in the authorization flags
right := &s4a.AuthRight{
    CardLow:     1234567890,
    BeginDate:   s4a.BCDDateEncode(2025, 1, 1),
    EndDate:     s4a.BCDDateEncode(2030, 12, 31),
    EndTime:     s4a.BCDTimeEncode(23, 59, 58),
    ReaderMask:  0xFF,
    RemainCount: 0xFFFF,
    Flags:       0x40,  // hasAntiback bit set
}
client.Authorize(ctx, right)
```

### Live Card Swipe Monitoring

S4A software: Operation > Console > Monitor

```go
listener, _ := s4a.NewEventListener(":50000")
defer listener.Close()

listener.ListenEvents(ctx, func(evt *s4a.Event) error {
    switch evt.Type {
    case s4a.EventTypeCardSwipe:
        fmt.Printf("[%s] card=%s door=%s reader=%s result=%s dir=%s\n",
            evt.SwipeTime, evt.CardData, evt.DoorNo,
            evt.ReaderNo, evt.Result, evt.Direction)
    case s4a.EventTypeHeartbeat:
        fmt.Printf("[heartbeat] controller=%s fw=%s\n",
            evt.HBControllerFlag, evt.HBFirmwareVersion)
    case s4a.EventTypeSignalChange:
        fmt.Printf("[signal] prev=%s curr=%s\n",
            evt.SCPrevSignals, evt.SCCurrSignals)
    }
    return nil
})
```

### Download Swipe Records

S4A software: Operation > Console > Download, then Operation > Query Swipe Records

```go
// Poll logs starting from index 1
index := uint32(1)
for {
    resp, err := client.MonitorLog(ctx, index)
    if err != nil {
        break
    }
    if resp.LogHigh == 0 && resp.LogLow == 0 {
        break // no more records
    }
    // Process the log entry
    var entry s4a.LogEntry
    entry.UnmarshalBinary(rawLogBytes(resp))
    fmt.Printf("[%s] card=%d door=%d reader=%d result=%s\n",
        entry.Date, entry.CardNumber(), entry.Door, entry.Reader,
        s4a.ControllerErrorCode(entry.Result))
    index++
}
```

### Controller Info Check

S4A software: Operation > Console > Check

```go
ct, _ := s4a.NewController("10.254.33.10:65534")
defer ct.Close()

ct.RefreshInfo(ctx)
info := ct.Info()
fmt.Printf("serial=%s fw=%s auth=%d logs=%d time=%s\n",
    info.SerialNum, info.FirmwareVer,
    info.AuthCount, info.LogCount, info.CurrentTime)
```

### Set Controller Time

S4A software: Operation > Console > Adjust Time

```go
// Sync to current system time
ct.SyncTime(ctx)

// Or set manually via text command
tc := s4a.NewTextCommand().SetTime(2025, 7, 9, 15, 30, 0, 3)
f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)
```

### Inter-lock (Two doors never open simultaneously)

S4A software: Configuration > Inter Lock (Extended Functions)

```go
sys := s4a.NewSystem()
defer sys.Shutdown()

ct1, _ := s4a.NewController("10.254.33.10:65534")
ct2, _ := s4a.NewController("10.254.33.11:65534")
sys.AddController(ct1)
sys.AddController(ct2)

// Door 1 on ct1 and door 1 on ct2 inter-locked
sys.EnableInterlock("10.254.33.10:65534", 1, "10.254.33.11:65534", 1)

// Open via system to enforce inter-lock
err := sys.OpenDoor(ctx, "10.254.33.10:65534", 1, 300)
if err != nil {
    fmt.Println("blocked:", err) // blocked if ct2's door 1 is open
}
```

### Multi-Card Access (2+ people required)

S4A software: Configuration > Multi-card (Extended Functions)

```go
sys.EnableMultiCard("10.254.33.10:65534", 1, 3) // 3 cards required

// In your event listener, feed card swipes to the system
listener.ListenEvents(ctx, func(evt *s4a.Event) error {
    if evt.Type == s4a.EventTypeCardSwipe {
        handled, _ := sys.HandleCardSwipe(ctx, "10.254.33.10:65534", evt)
        if handled {
            fmt.Println("card queued for multi-card access")
        }
    }
    return nil
})
```

### First Card Open (door unlocks at first swipe)

S4A software: Configuration > First Card (Extended Functions)

```go
sys.EnableFirstCardOpen("10.254.33.10:65534", 1)

// On first authorized swipe, door stays unlocked
// To lock again:
sys.LockDoor(ctx, "10.254.33.10:65534", 1)
```

### Configure Reader / Port / Signal

S4A software: Configuration > Controllers > Door Config

```go
// Configure Wiegand reader on port 5: entry direction, triggers relay 1
tc := s4a.NewTextCommand().
    SetReaderMode(s4a.PortWiegand1, 3).
    SetReaderDir(s4a.PortWiegand1, s4a.DirEntry).
    SetReaderDoor(s4a.PortWiegand1, 1).
    SetWiegandFormat(s4a.PortWiegand1, 1).
    SetSignal(2, 2).          // S2 = exit button
    SetSignalDoor(2, 1).      // S2 triggers relay 1
    SetSignalDir(2, s4a.DirExit).
    SetRelayDelay(1, 3000).   // Relay 1 default 3s
    SetCloseTimeout(15)       // Door-open alarm after 15s

f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)
```

### Bulk Authorize from File

S4A software: Configuration > Access Privilege > full upload

```go
// Read card numbers from a file, one per line
cards, _ := os.ReadFile("cards.txt")
for _, line := range strings.Split(strings.TrimSpace(string(cards)), "\n") {
    cardLow, _ := strconv.ParseUint(line, 10, 32)
    right := &s4a.AuthRight{
        CardLow:     uint32(cardLow),
        BeginDate:   s4a.BCDDateEncode(2025, 1, 1),
        EndDate:     s4a.BCDDateEncode(2030, 12, 31),
        EndTime:     s4a.BCDTimeEncode(23, 59, 58),
        ReaderMask:  0xFF,
        RemainCount: 0xFFFF,
    }
    if err := client.Authorize(ctx, right); err != nil {
        log.Printf("failed to authorize %d: %v", cardLow, err)
    }
}

// For very large batches (>500), use full-upload approach:
// 1. Clear all existing authorizations
client.ClearAuth(ctx)
// 2. Upload each card
// 3. Restart controller
tc := s4a.NewTextCommand().Restart()
f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)
```

### Reboot / Clear Data

S4A software: Configuration > Controllers > Reboot / Clear All Data

```go
tc := s4a.NewTextCommand().Restart()
f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)

// Clear all logs
tc = s4a.NewTextCommand().ClearData(s4a.ClearLogs)
f = tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)

// Clear all authorizations
tc = s4a.NewTextCommand().ClearData(s4a.ClearAuth)
f = tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)
```

### Display Text on Screen

S4A software: This is part of the hardware config, triggered automatically on swipe

```go
tc := s4a.NewTextCommand().DisplayScreen(
    "John Doe^Male^1234567890^^Access Granted^2025-07-09 15:30:00^^",
    s4a.DirEntry, // show on entry screen
    2,            // page 2 (IC/barcode result)
    0,            // restore default page after
    5,            // show for 5 seconds
)
f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)
```

### Play Voice / Audio

```go
// Play voice index 5 three times immediately
tc := s4a.NewTextCommand().Sound(5, 3, true)
f := tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)

// Text-to-speech
tc = s4a.NewTextCommand().TTS("Welcome")
f = tc.BuildFrame(s4a.DefaultDeviceID, seq)
client.sendAndWait(ctx, f)
```
