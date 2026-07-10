package s4a

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestControllerErrorCode(t *testing.T) {
	tests := []struct {
		code     uint8
		expected string
	}{
		{0, "Success"},
		{4, "No permission"},
		{14, "Failed"},
		{38, "ID expired"},
		{99, "Unknown"},
	}
	for _, tt := range tests {
		got := ControllerErrorCode(tt.code)
		if got != tt.expected {
			t.Errorf("ControllerErrorCode(%d): got %q, want %q", tt.code, got, tt.expected)
		}
	}
}

func TestDoorDirection(t *testing.T) {
	if s := DoorDirection(1); s != "Entry" {
		t.Errorf("DoorDirection(1): got %q", s)
	}
	if s := DoorDirection(2); s != "Exit" {
		t.Errorf("DoorDirection(2): got %q", s)
	}
}

func TestLogTypeString(t *testing.T) {
	if s := LogTypeString(2); s != "Card swipe" {
		t.Errorf("LogTypeString(2): got %q", s)
	}
}

func TestParseQueryAuthResponse(t *testing.T) {
	right := &AuthRight{
		CardLow:     0x12345678,
		BeginDate:   BCDDateEncode(2025, 1, 15),
		EndDate:     BCDDateEncode(2030, 12, 31),
		EndTime:     BCDTimeEncode(23, 59, 58),
		ReaderMask:  0xFF,
		RemainCount: 0xFFFF,
	}
	data, _ := right.AppendBinary(nil)
	f := &Frame{
		Preamble: respPreamble,
		DeviceID: 0xFFFF,
		Seq:      8,
		Cmd:      CmdQueryAuthResp,
		Result:   ResultSuccess,
		Data:     data,
	}
	parsed, err := ParseQueryAuthResponse(f)
	if err != nil {
		t.Fatalf("ParseQueryAuthResponse: %v", err)
	}
	if parsed.CardLow != 0x12345678 {
		t.Errorf("CardLow: got 0x%08x", parsed.CardLow)
	}
}

func TestParseQueryAuthResponseWrongCmd(t *testing.T) {
	f := &Frame{Cmd: 0x10, Result: ResultSuccess}
	_, err := ParseQueryAuthResponse(f)
	if err == nil {
		t.Error("expected error for wrong command")
	}
}

func TestParseTextCommandResponse(t *testing.T) {
	f := &Frame{Preamble: respPreamble, DeviceID: 0xFFFF, Seq: 8, Cmd: CmdTextCommandResp, Result: ResultSuccess}
	if err := ParseTextCommandResponse(f); err != nil {
		t.Fatalf("ParseTextCommandResponse: %v", err)
	}
}

func TestParseTextCommandResponseWrongCmd(t *testing.T) {
	f := &Frame{Cmd: 0x10, Result: ResultSuccess}
	if err := ParseTextCommandResponse(f); err == nil {
		t.Error("expected error for wrong command")
	}
}

func TestHeartbeatEventFullParse(t *testing.T) {
	payload := []byte("24884|120|33|1|24884|7879047689384705|v0.1551-10")
	raw := make([]byte, 8+len(payload))
	raw[0] = 0xc8
	raw[1] = 0x04
	raw[2] = 0x00
	raw[3] = byte(len(payload))
	copy(raw[8:], payload)

	evt, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent heartbeat: %v", err)
	}
	if evt.Type != EventTypeHeartbeat {
		t.Errorf("Type: got %d, want %d", evt.Type, EventTypeHeartbeat)
	}
	if evt.HBControllerFlag != "24884" {
		t.Errorf("HBControllerFlag: got %q", evt.HBControllerFlag)
	}
	if evt.HBTimeoutConfig != "120" {
		t.Errorf("HBTimeoutConfig: got %q", evt.HBTimeoutConfig)
	}
	if evt.HBFirmwareVersion != "v0.1551-10" {
		t.Errorf("HBFirmwareVersion: got %q", evt.HBFirmwareVersion)
	}
}

func TestSignalChangeEventFullParse(t *testing.T) {
	payloadStr := "255;251|25138|2016-07-21 16:09:47|258|259|514|515|770|771|1026|1027|25138|192.168.0.155:50000"
	payload := []byte(payloadStr)
	raw := make([]byte, 8+len(payload))
	raw[0] = 0xc8
	raw[1] = 0x06
	raw[2] = 0x00
	raw[3] = byte(len(payload))
	copy(raw[8:], payload)

	evt, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent signal change: %v", err)
	}
	if evt.Type != EventTypeSignalChange {
		t.Errorf("Type: got %d, want %d", evt.Type, EventTypeSignalChange)
	}
	if evt.SCPrevSignals != "255" {
		t.Errorf("SCPrevSignals: got %q, want \"255\"", evt.SCPrevSignals)
	}
	if evt.SCCurrSignals != "251" {
		t.Errorf("SCCurrSignals: got %q, want \"251\"", evt.SCCurrSignals)
	}
	if evt.SCControllerFlag != "25138" {
		t.Errorf("SCControllerFlag: got %q", evt.SCControllerFlag)
	}
	if evt.SCChangeTime != "2016-07-21 16:09:47" {
		t.Errorf("SCChangeTime: got %q", evt.SCChangeTime)
	}
	if evt.SCPeerAddr != "192.168.0.155:50000" {
		t.Errorf("SCPeerAddr: got %q", evt.SCPeerAddr)
	}
}

func TestRevokeAuthRequestKnownVector(t *testing.T) {
	f := NewRevokeAuthRequest(0x6833, 8, 0x00000000, 0x499602D2)
	raw, _ := f.AppendBinary(nil)
	expected := append(
		reqPreamble[:],
		0x68, 0x33, 0x08, 0x00, 0x14, 0x00, 0x00, 0x08,
		0x00, 0x00, 0x00, 0x00, 0xd2, 0x02, 0x96, 0x49, 0xcf,
	)
	if !bytes.Equal(raw, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, raw)
	}
}

func TestClearAuthRequestKnownVector(t *testing.T) {
	f := NewClearAuthRequest(0x6136, 2)
	raw, _ := f.AppendBinary(nil)
	expected := append(reqPreamble[:], 0x61, 0x36, 0x02, 0x00, 0x18, 0x00, 0x00, 0x00, 0x18)
	if !bytes.Equal(raw, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, raw)
	}
}

func TestMonitorLogRequestKnownVector(t *testing.T) {
	f := NewMonitorLogRequest(0x6136, 8, 1)
	raw, _ := f.AppendBinary(nil)
	expected := append(
		reqPreamble[:],
		0x61, 0x36, 0x08, 0x00, 0x38, 0x00, 0x00, 0x04,
		0x01, 0x00, 0x00, 0x00, 0x3D,
	)
	if !bytes.Equal(raw, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, raw)
	}
}

func TestSetTimeRequestKnownVector(t *testing.T) {
	td := [7]byte{0x18, 0x05, 0x04, 0x13, 0x30, 0x01, 0x05}
	f := NewSetTimeRequest(0x6136, 8, td)
	raw, _ := f.AppendBinary(nil)
	expected := append(
		reqPreamble[:],
		0x61, 0x36, 0x08, 0x00, 0x26, 0x00, 0x00, 0x07,
		0x18, 0x05, 0x04, 0x13, 0x30, 0x01, 0x05, 0x97,
	)
	if !bytes.Equal(raw, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, raw)
	}
}

func TestAuthorizeRequestKnownVector(t *testing.T) {
	right := &AuthRight{
		CardLow:     0x499602D2,
		BeginDate:   0x20bf,
		EndDate:     0x3c21,
		EndTime:     0xbf7d,
		ReaderMask:  0xFF,
		RemainCount: 0xFFFF,
	}
	f := NewAuthorizeRequest(0x6833, 4, right)
	raw, _ := f.AppendBinary(nil)
	expected := append(
		reqPreamble[:],
		0x68, 0x33, 0x04, 0x00, 0x12, 0x00, 0x00, 0x18,
		0x00, 0x00, 0x00, 0x00, 0xd2, 0x02, 0x96, 0x49,
		0xbf, 0x20, 0x00, 0x00, 0x21, 0x3c, 0x7d, 0xbf,
		0x00, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00,
		0x52,
	)
	if !bytes.Equal(raw, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, raw)
	}
}

func TestMonitorLogResponseFullParse(t *testing.T) {
	data := []byte{
		0x01, 0x00, 0x00, 0x00,
		0xe2, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x9c, 0x24, 0xc7, 0x1a, 0x01, 0x00, 0x0c, 0x0e,
		0x02, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x18, 0x04, 0x28, 0x11, 0x18, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x59, 0x61, 0x7a, 0x82,
		0x30, 0x30, 0x32, 0x35, 0x35,
	}
	f := &Frame{
		Preamble: respPreamble, DeviceID: 0xFFFF, Seq: 8,
		Cmd: CmdMonitorLogResp, Result: ResultSuccess, Data: data,
	}
	resp, err := ParseMonitorLogResponse(f)
	if err != nil {
		t.Fatalf("ParseMonitorLogResponse: %v", err)
	}
	if resp.LogSeq != 1 || resp.LogCount != 2 || resp.AuthCount != 0 {
		t.Errorf("counts: seq=%d log=%d auth=%d", resp.LogSeq, resp.LogCount, resp.AuthCount)
	}
	if string(resp.DeviceFlag) != "00255" {
		t.Errorf("DeviceFlag: got %q", string(resp.DeviceFlag))
	}
}

func TestTCPFrameRoundTrip(t *testing.T) {
	f := NewOpenDoorRequest(0xFFFF, 5, 2, 150)
	raw := f.MarshalTCP()
	tcpLen := binary.BigEndian.Uint32(raw[0:4])
	innerLen := len(raw) - 4
	if uint32(innerLen) != tcpLen {
		t.Errorf("TCP length prefix: got %d, want %d", tcpLen, innerLen)
	}
	var decoded Frame
	if err := decoded.UnmarshalBinary(raw[4:]); err != nil {
		t.Fatalf("UnmarshalBinary from TCP: %v", err)
	}
	if decoded.Cmd != CmdOpenDoor {
		t.Errorf("cmd: got 0x%02x, want 0x10", decoded.Cmd)
	}
}

func TestDurationUnit(t *testing.T) {
	if 300*time.Millisecond*10 != 3*time.Second {
		t.Error("duration unit assumption broken")
	}
}

func TestAllErrorCodesCovered(t *testing.T) {
	codes := []uint8{
		0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
		16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27,
		28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38,
	}
	for _, c := range codes {
		s := ControllerErrorCode(c)
		if s == "" || (s == "Unknown" && c != 99) {
			t.Errorf("error code %d returns %q", c, s)
		}
	}
	if ControllerErrorCode(1) != "Unknown" {
		t.Error("code 1 should be Unknown")
	}
}
