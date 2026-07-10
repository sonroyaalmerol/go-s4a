package s4a

import (
	"fmt"
	"strings"
)

type TextCommand struct {
	parts []string
}

func NewTextCommand() *TextCommand {
	return &TextCommand{}
}

func (tc *TextCommand) String() string {
	return strings.Join(tc.parts, ";")
}

func (tc *TextCommand) BuildFrame(deviceID uint16, seq uint16) *Frame {
	return NewTextCommandRequest(deviceID, seq, tc.String())
}

func (tc *TextCommand) add(key, value string) {
	if value == "" {
		tc.parts = append(tc.parts, key)
	} else {
		tc.parts = append(tc.parts, fmt.Sprintf("%s=%s", key, value))
	}
}

func (tc *TextCommand) OpenDoor(door uint8, durationMs int) *TextCommand {
	tc.add(fmt.Sprintf("open%d", door), fmt.Sprintf("%d", durationMs))
	return tc
}

func (tc *TextCommand) OpenDoorMulti(door uint8, durationMs int, count int, intervalMs int) *TextCommand {
	tc.OpenDoor(door, durationMs)
	tc.add(fmt.Sprintf("time%d", door), fmt.Sprintf("%d", count))
	tc.add("stopn", fmt.Sprintf("%d", intervalMs))
	return tc
}

func (tc *TextCommand) Sound(idx int, loop int, immediate bool) *TextCommand {
	tc.add("sound", fmt.Sprintf("%d", idx))
	if loop > 0 {
		tc.add("loop", fmt.Sprintf("%d", loop))
	}
	if immediate {
		tc.add("soundNow", "1")
	}
	return tc
}

func (tc *TextCommand) TTS(text string) *TextCommand {
	tc.add("tts", text+"$")
	return tc
}

func (tc *TextCommand) SetTime(year, month, day, hour, minute, second int, weekday int) *TextCommand {
	val := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d %d", year, month, day, hour, minute, second, weekday)
	tc.add("settime", val)
	return tc
}

func (tc *TextCommand) SetReportIP(ip string) *TextCommand {
	tc.add("setReportIp", ip)
	return tc
}

func (tc *TextCommand) SetReportPort(port int) *TextCommand {
	tc.add("setReportPort", fmt.Sprintf("%d", port))
	return tc
}

func (tc *TextCommand) SetGroupIndex(idx int) *TextCommand {
	tc.add("setGrpIndex", fmt.Sprintf("%d", idx))
	return tc
}

func (tc *TextCommand) SetIP(ip string) *TextCommand {
	tc.add("setIp", ip)
	return tc
}

func (tc *TextCommand) SetMask(mask string) *TextCommand {
	tc.add("setMask", mask)
	return tc
}

func (tc *TextCommand) SetGateway(gw string) *TextCommand {
	tc.add("setGate", gw)
	return tc
}

const (
	IPModeTCPServer = 0
	IPModeTCPClient = 1
	IPModeUDP       = 2
)

func (tc *TextCommand) SetIPMode(mode int) *TextCommand {
	tc.add("setIpMode", fmt.Sprintf("%d", mode))
	return tc
}

const (
	IPAllocStatic = 0
	IPAllocDHCP   = 1
)

func (tc *TextCommand) SetIPAlloc(mode int) *TextCommand {
	tc.add("setIpAlloc", fmt.Sprintf("%d", mode))
	return tc
}

func (tc *TextCommand) SetName(name string) *TextCommand {
	tc.add("setName", name+"$")
	return tc
}

func (tc *TextCommand) SetOptions(mask1, option1, mask2, option2 uint32) *TextCommand {
	tc.add("mask1", fmt.Sprintf("%d", mask1))
	tc.add("option1", fmt.Sprintf("%d", option1))
	if mask2 != 0 || option2 != 0 {
		tc.add("mask2", fmt.Sprintf("%d", mask2))
		tc.add("option2", fmt.Sprintf("%d", option2))
	}
	return tc
}

func (tc *TextCommand) SetCloseTimeout(seconds int) *TextCommand {
	tc.add("closeTimeout", fmt.Sprintf("%d", seconds))
	return tc
}

func (tc *TextCommand) SetAlarmNotClose(relay int) *TextCommand {
	tc.add("alarmNotClose", fmt.Sprintf("%d", relay))
	return tc
}

func (tc *TextCommand) SetRelayDelay(door uint8, durationMs int) *TextCommand {
	tc.add(fmt.Sprintf("delay%d", door), fmt.Sprintf("%d", durationMs))
	return tc
}

func (tc *TextCommand) DisplayScreen(content string, dir int, page int, restorePage int, restoreSec int) *TextCommand {
	tc.add("prompt", content+"$")
	if dir > 0 {
		tc.add("prompt-dir", fmt.Sprintf("%d", dir))
	}
	if page > 0 {
		tc.add("prompt-page", fmt.Sprintf("%d", page))
	}
	if restorePage > 0 {
		tc.add("restore-page", fmt.Sprintf("%d", restorePage))
	}
	if restoreSec > 0 {
		tc.add("restore-seconds", fmt.Sprintf("%d", restoreSec))
	}
	return tc
}

const (
	Baud9600   = 9600
	Baud19200  = 19200
	Baud38400  = 38400
	Baud57600  = 57600
	Baud115200 = 115200
)

func (tc *TextCommand) SetBaud(port uint8, baud int) *TextCommand {
	tc.add(fmt.Sprintf("baund%d", port), fmt.Sprintf("%d", baud))
	return tc
}

func (tc *TextCommand) SetSerialDevice(port uint8, devType int) *TextCommand {
	tc.add(fmt.Sprintf("dev%d", port), fmt.Sprintf("%d", devType))
	return tc
}

const (
	PortSerial1  = 1
	PortSerial2  = 2
	PortSerial3  = 3
	PortSerial4  = 4
	PortWiegand1 = 5
	PortWiegand2 = 6
	PortWiegand3 = 7
	PortWiegand4 = 8
)

const (
	DirUnknown = 0
	DirEntry   = 1
	DirExit    = 2
)

func (tc *TextCommand) SetReaderMode(port uint8, mode int) *TextCommand {
	tc.add(fmt.Sprintf("mode%d", port), fmt.Sprintf("%d", mode))
	return tc
}

func (tc *TextCommand) SetReaderDir(port uint8, dir int) *TextCommand {
	tc.add(fmt.Sprintf("dir%d", port), fmt.Sprintf("%d", dir))
	return tc
}

func (tc *TextCommand) SetReaderDoor(port uint8, door uint8) *TextCommand {
	tc.add(fmt.Sprintf("door%d", port), fmt.Sprintf("%d", door))
	return tc
}

func (tc *TextCommand) SetWiegandFormat(port uint8, format int) *TextCommand {
	tc.add(fmt.Sprintf("format%d", port), fmt.Sprintf("%d", format))
	return tc
}

func (tc *TextCommand) SetSignal(signal uint8, sigType int) *TextCommand {
	tc.add(fmt.Sprintf("sig%d", signal), fmt.Sprintf("%d", sigType))
	return tc
}

func (tc *TextCommand) SetSignalDoor(signal uint8, door uint8) *TextCommand {
	tc.add(fmt.Sprintf("sigdr%d", signal), fmt.Sprintf("%d", door))
	return tc
}

func (tc *TextCommand) SetSignalDir(signal uint8, dir int) *TextCommand {
	tc.add(fmt.Sprintf("sigdir%d", signal), fmt.Sprintf("%d", dir))
	return tc
}

func (tc *TextCommand) SetTimeZone(zone uint8, timeStart, timeEnd string, weekMask int) *TextCommand {
	tc.add(fmt.Sprintf("tztime%d", zone), timeStart+timeEnd)
	tc.add(fmt.Sprintf("tzweek%d", zone), fmt.Sprintf("%d", weekMask))
	return tc
}

const (
	ClearConfig  = 0
	ClearAuth    = 1
	ClearIDCache = 2
	ClearLogs    = 3
)

func (tc *TextCommand) ClearData(dataType int) *TextCommand {
	tc.add("clearData_", fmt.Sprintf("%d", dataType))
	return tc
}

func (tc *TextCommand) ResetReportIndex(toLatest bool) *TextCommand {
	if toLatest {
		tc.add("clearRptPos", "1")
	} else {
		tc.add("clearRptPos", "0")
	}
	return tc
}

func (tc *TextCommand) ClearIOStats() *TextCommand {
	tc.add("clearIoStat", "1")
	return tc
}

func (tc *TextCommand) Restart() *TextCommand {
	tc.add("powerReset_", "1")
	return tc
}

func (tc *TextCommand) Debug(on bool) *TextCommand {
	if on {
		tc.add("debug", "1")
	} else {
		tc.add("debug", "0")
	}
	return tc
}

func (tc *TextCommand) Domain(host string) *TextCommand {
	tc.add("domain", host+"$")
	return tc
}

func (tc *TextCommand) AckRuntime(msgID int) *TextCommand {
	tc.add("runtimeAck", fmt.Sprintf("%d", msgID))
	return tc
}

func (tc *TextCommand) AckLog(logID int) *TextCommand {
	tc.add("acklog", fmt.Sprintf("%d", logID))
	return tc
}

func (tc *TextCommand) AckHeartbeat() *TextCommand {
	tc.add("heartbeatAck", "1")
	return tc
}
