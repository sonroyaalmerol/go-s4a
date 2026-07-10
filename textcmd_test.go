package s4a

import (
	"testing"
)

func TestTextCommandOpenDoor(t *testing.T) {
	tc := NewTextCommand().OpenDoor(1, 300)
	if s := tc.String(); s != "open1=300" {
		t.Errorf("got %q, want %q", s, "open1=300")
	}
}

func TestTextCommandOpenDoorMulti(t *testing.T) {
	tc := NewTextCommand().OpenDoorMulti(2, 300, 5, 1000)
	if s := tc.String(); s != "open2=300;time2=5;stopn=1000" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandSound(t *testing.T) {
	tc := NewTextCommand().Sound(5, 3, true)
	if s := tc.String(); s != "sound=5;loop=3;soundNow=1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandTTS(t *testing.T) {
	tc := NewTextCommand().TTS("hello")
	if s := tc.String(); s != "tts=hello$" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandSetTime(t *testing.T) {
	tc := NewTextCommand().SetTime(2025, 6, 15, 14, 30, 0, 1)
	if s := tc.String(); s != "settime=2025-06-15 14:30:00 1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandNetwork(t *testing.T) {
	tc := NewTextCommand().
		SetIP("192.168.1.100").
		SetMask("255.255.255.0").
		SetGateway("192.168.1.1").
		SetIPMode(IPModeUDP).
		SetName("FrontGate")
	if s := tc.String(); s != "setIp=192.168.1.100;setMask=255.255.255.0;setGate=192.168.1.1;setIpMode=2;setName=FrontGate$" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandOptions(t *testing.T) {
	tc := NewTextCommand().SetOptions(3, 3, 0, 0)
	if s := tc.String(); s != "mask1=3;option1=3" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandOptionsFull(t *testing.T) {
	tc := NewTextCommand().SetOptions(0xFF, 0x55, 0x01, 0x01)
	if s := tc.String(); s != "mask1=255;option1=85;mask2=1;option2=1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandDisplay(t *testing.T) {
	tc := NewTextCommand().DisplayScreen("Name^Gender^CardNo^^Result^Time^^", 3, 2, 0, 5)
	if s := tc.String(); s != "prompt=Name^Gender^CardNo^^Result^Time^^$;prompt-dir=3;prompt-page=2;restore-seconds=5" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandReaderConfig(t *testing.T) {
	tc := NewTextCommand().
		SetReaderMode(PortWiegand1, 3).
		SetReaderDir(PortWiegand1, DirEntry).
		SetReaderDoor(PortWiegand1, 1).
		SetWiegandFormat(PortWiegand1, 1)
	if s := tc.String(); s != "mode5=3;dir5=1;door5=1;format5=1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandRelayConfig(t *testing.T) {
	tc := NewTextCommand().SetRelayDelay(1, 3000).SetCloseTimeout(15)
	if s := tc.String(); s != "delay1=3000;closeTimeout=15" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandSignalConfig(t *testing.T) {
	tc := NewTextCommand().SetSignal(1, 2).SetSignalDoor(1, 1).SetSignalDir(1, DirEntry)
	if s := tc.String(); s != "sig1=2;sigdr1=1;sigdir1=1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandTimeZone(t *testing.T) {
	tc := NewTextCommand().SetTimeZone(2, "0830", "1730", 127)
	if s := tc.String(); s != "tztime2=08301730;tzweek2=127" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandSystem(t *testing.T) {
	tc := NewTextCommand().Restart().ClearData(ClearLogs).ClearIOStats().Debug(true)
	if s := tc.String(); s != "powerReset_=1;clearData_=3;clearIoStat=1;debug=1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandACK(t *testing.T) {
	tc := NewTextCommand().AckRuntime(2356).AckLog(1237).AckHeartbeat()
	if s := tc.String(); s != "runtimeAck=2356;acklog=1237;heartbeatAck=1" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandBaud(t *testing.T) {
	tc := NewTextCommand().SetBaud(1, Baud115200).SetSerialDevice(1, 16)
	if s := tc.String(); s != "baund1=115200;dev1=16" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandDomain(t *testing.T) {
	tc := NewTextCommand().Domain("www.example.com")
	if s := tc.String(); s != "domain=www.example.com$" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandBuildFrame(t *testing.T) {
	tc := NewTextCommand().OpenDoor(1, 300)
	f := tc.BuildFrame(0xFFFF, 1)
	if f.Cmd != CmdTextCommand {
		t.Errorf("cmd: got 0x%02x, want 0x94", f.Cmd)
	}
	if f.DeviceID != 0xFFFF {
		t.Errorf("deviceID: got 0x%04x", f.DeviceID)
	}
}

func TestTextCommandSetReport(t *testing.T) {
	tc := NewTextCommand().SetReportIP("10.0.0.1").SetReportPort(50000).SetGroupIndex(12345)
	if s := tc.String(); s != "setReportIp=10.0.0.1;setReportPort=50000;setGrpIndex=12345" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandResetReportIndex(t *testing.T) {
	tc := NewTextCommand().ResetReportIndex(true)
	if s := tc.String(); s != "clearRptPos=1" {
		t.Errorf("got %q", s)
	}
	tc2 := NewTextCommand().ResetReportIndex(false)
	if s := tc2.String(); s != "clearRptPos=0" {
		t.Errorf("got %q", s)
	}
}

func TestTextCommandCombined(t *testing.T) {
	tc := NewTextCommand().
		SetIP("10.0.0.100").
		SetMask("255.0.0.0").
		SetGateway("10.0.0.1").
		SetIPMode(IPModeTCPClient).
		SetName("MainDoor").
		SetReaderMode(PortSerial1, 3).
		SetReaderDir(PortSerial1, DirEntry).
		SetReaderDoor(PortSerial1, 1)
	expected := "setIp=10.0.0.100;setMask=255.0.0.0;setGate=10.0.0.1;setIpMode=1;setName=MainDoor$;mode1=3;dir1=1;door1=1"
	if s := tc.String(); s != expected {
		t.Errorf("got %q, want %q", s, expected)
	}
}
