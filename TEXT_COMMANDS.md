# JL-IDD-Z4 Controller Text Command Reference

Source: http://www.ykt1.cn/news259.html
Section: 3.6 Supported Text Command Set

Text commands are sent via cmd 0x94 to the controller. Multiple commands can be combined with semicolons. Format: `prefix=value`. Values are case-sensitive.

Command prefixes in `a[x-y]` notation represent ranges (e.g. `open[1-4]` means `open1`, `open2`, `open3`, `open4`).

## Door Control

| Command     | Value          | Description                        |
| ----------- | -------------- | ---------------------------------- |
| `open[1-4]` | Duration in ms | Open door relay (1-4)              |
| `time[1-4]` | Count          | Number of door open cycles         |
| `stopn`     | Interval in ms | Delay between multi-cycle openings |

Example: `open1=300;time1=5;stopn=1000` — Door 1 opens 5 times, 1s delay between each.

## Audio

| Command    | Value                | Description                   |
| ---------- | -------------------- | ----------------------------- |
| `sound`    | 0-219 index          | Play onboard voice by index   |
| `soundNow` | Any value            | Play immediately (not queued) |
| `loop`     | Count                | Repeat count for sound        |
| `tts`      | Text ending with `$` | TTS voice output              |

Voice index 224-231: volume control (higher = louder). 242: loop current. 254: stop.

## Time

| Command                | Value                         | Description                                      |
| ---------------------- | ----------------------------- | ------------------------------------------------ |
| `settime` or `setTime` | `YYYY-MM-DD HH:MI:SS WEEKDAY` | Set controller time. WEEKDAY: 0=Sun, 1-6=Mon-Sat |

## Network Configuration

| Command         | Value                              | Description             |
| --------------- | ---------------------------------- | ----------------------- |
| `setReportIp`   | IP address                         | Set report server IP    |
| `setReportPort` | Port number                        | Set report server port  |
| `setGrpIndex`   | Integer                            | Set group serial number |
| `setIp`         | IP address                         | Set device IP           |
| `setMask`       | Subnet mask                        | Set subnet mask         |
| `setGate`       | Gateway IP                         | Set gateway             |
| `setIpMode`     | 0=TCP server, 1=TCP client, 2=UDP  | Set communication mode  |
| `setIpAlloc`    | 0=static IP, 1=DHCP                | Set IP allocation mode  |
| `setName`       | Text ending with `$`, max 14 bytes | Set device name         |

## Device Options (mask1/mask2/option1/option2)

Options are 64 boolean flags controlled via bitmask pairs.

| Command   | Value        | Description                            |
| --------- | ------------ | -------------------------------------- |
| `mask1`   | 0-0xFFFFFFFF | Which options 1-32 to modify (bitmask) |
| `option1` | 0-0xFFFFFFFF | New values for options 1-32 (1=on)     |
| `mask2`   | 0-0xFFFFFFFF | Which options 33-64 to modify          |
| `option2` | 0-0xFFFFFFFF | New values for options 33-64           |

Example: `mask1=3;option1=3` — Sets options 1 and 2 to ON.

## Door/Relay Configuration

| Command         | Value           | Description                                                                                               |
| --------------- | --------------- | --------------------------------------------------------------------------------------------------------- |
| `closeTimeout`  | Seconds (0=off) | Door open timeout before alarm                                                                            |
| `alarmNotClose` | 0-255           | Door-ajar alarm relay. 0=off, n<16=trigger relay n doors away from current, n>16=trigger relay (n mod 16) |
| `delay[1-4]`    | Duration in ms  | Default relay open duration                                                                               |

## Display Screen

| Command           | Value                                         | Description                                                                           |
| ----------------- | --------------------------------------------- | ------------------------------------------------------------------------------------- |
| `prompt`          | Text ending with `$`, fields separated by `^` | Display info on screen. Fields: Name^Gender^CardNo^Test^Result^Time^Reserved^Reserved |
| `prompt-dir`      | 1=entry, 2=exit, 3=both                       | Display direction                                                                     |
| `prompt-page`     | Page number                                   | Display page. 0=default, 1=ID card info, 2=IC/barcode result                          |
| `restore-page`    | Page number                                   | Page to restore after timeout                                                         |
| `restore-seconds` | Seconds                                       | Display duration before restore                                                       |

## Serial Port Configuration

| Command      | Value                                         | Description                |
| ------------ | --------------------------------------------- | -------------------------- |
| `baund[1-4]` | 9600/19200/38400/57600/115200                 | Serial port baud rate      |
| `dev[1-4]`   | Device type index (from config tool dropdown) | Device type on serial port |

## Reader Configuration

| Command       | Value                           | Description                                               |
| ------------- | ------------------------------- | --------------------------------------------------------- |
| `mode[1-8]`   | Mode value (from config tool)   | Reader work mode for ports 1-4 (serial) and 5-8 (Wiegand) |
| `dir[1-8]`    | 1=in, 2=out, 0=unknown          | Reader direction                                          |
| `door[1-8]`   | Relay number 1-4                | Relay triggered by this reader                            |
| `format[1-8]` | Format index (from config tool) | Wiegand format                                            |

## Signal Configuration

| Command       | Value                                | Description                    |
| ------------- | ------------------------------------ | ------------------------------ |
| `sig[1-8]`    | Signal type index (from config tool) | Signal terminal S1-S8 function |
| `sigdr[1-8]`  | Door number                          | Signal-associated door         |
| `sigdir[1-8]` | Direction index                      | Signal direction               |

## Time Zone Configuration

| Command       | Value                         | Description                                            |
| ------------- | ----------------------------- | ------------------------------------------------------ |
| `tztime[2-8]` | 8-digit HHMMHHMM (start-end)  | Time zone effective time range. tz1 is always disabled |
| `tzweek[2-8]` | 0-127 bitmask (Mon=1, Sun=64) | Time zone effective weekdays                           |

## System

| Command       | Value                                | Description                 |
| ------------- | ------------------------------------ | --------------------------- |
| `powerReset_` | 1                                    | Restart device              |
| `clearData_`  | 0=config, 1=auth, 2=ID cache, 3=logs | Clear data                  |
| `clearRptPos` | 0=reset to start, 1=reset to latest  | Reset report index          |
| `clearIoStat` | 1                                    | Clear entry/exit statistics |
| `debug`       | 0=off, 1=on                          | Debug toggle                |
| `domain`      | Hostname ending with `$`             | HTTP report host            |

## Protocol-level ACK

| Command              | Value      | Description          |
| -------------------- | ---------- | -------------------- |
| `runtimeAck`         | Message ID | Real-time report ACK |
| `heartbeatAck`       | 1          | Heartbeat ACK        |
| `acklog` or `logAck` | Log ID     | Async log ACK        |
