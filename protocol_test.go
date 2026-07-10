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

func TestParseQueryAuthResponse(t *testing.T) {
	right := &AuthRight{
		CardNumber:  0x12345678,
		ValidFrom:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Readers:  AllReaders,
		RemainCount: RemainUnlimited,
	}
	data, _ := right.AppendBinary(nil)
	f := &Frame{
		Preamble: respPreamble,
		DeviceID: 0xFFFF,
		Seq:      8,
		Cmd:      CmdQueryAuthResp,
		Result:   byte(ResultSuccess),
		Data:     data,
	}
	parsed, err := ParseQueryAuthResponse(f)
	if err != nil {
		t.Fatalf("ParseQueryAuthResponse: %v", err)
	}
	if parsed.CardNumber != 0x12345678 {
		t.Errorf("CardNumber: got 0x%08x", parsed.CardNumber)
	}
}

func TestParseQueryAuthResponseWrongCmd(t *testing.T) {
	f := &Frame{Cmd: 0x10, Result: byte(ResultSuccess)}
	_, err := ParseQueryAuthResponse(f)
	if err == nil {
		t.Error("expected error for wrong command")
	}
}

func TestParseTextCommandResponse(t *testing.T) {
	f := &Frame{Preamble: respPreamble, DeviceID: 0xFFFF, Seq: 8, Cmd: CmdTextCommandResp, Result: byte(ResultSuccess)}
	if err := ParseTextCommandResponse(f); err != nil {
		t.Fatalf("ParseTextCommandResponse: %v", err)
	}
}

func TestParseTextCommandResponseWrongCmd(t *testing.T) {
	f := &Frame{Cmd: 0x10, Result: byte(ResultSuccess)}
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
	f := NewRevokeAuthRequest(0x6833, 8, 0x499602D2)
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
	tm := time.Date(2024, 5, 3, 19, 48, 1, 0, time.Local)
	f := NewSetTimeRequest(0x6136, 8, tm)
	raw, _ := f.AppendBinary(nil)
	td := BuildTimeData(tm)
	if raw[32] != td[0] || raw[33] != td[1] || raw[34] != td[2] {
		t.Errorf("time data mismatch: got %v, want %v", raw[32:35], td[:3])
	}
}

func TestAuthorizeRequestKnownVector(t *testing.T) {
	right := &AuthRight{
		CardNumber:  0x499602D2,
		ValidFrom:   time.Date(0x20bf, 4, 16, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(0x3c21, 1, 1, 23, 59, 58, 0, time.Local),
		Readers:  AllReaders,
		RemainCount: RemainUnlimited,
	}
	f := NewAuthorizeRequest(0x6833, 4, right)
	raw, _ := f.AppendBinary(nil)
	if len(raw) < 56 {
		t.Fatalf("frame too short: %d bytes", len(raw))
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
		Cmd: CmdMonitorLogResp, Result: byte(ResultSuccess), Data: data,
	}
	resp, err := ParseMonitorLogResponse(f)
	if err != nil {
		t.Fatalf("ParseMonitorLogResponse: %v", err)
	}
	if resp.LogSeq != 1 || resp.LogCount != 2 || resp.AuthCount != 0 {
		t.Errorf("counts: seq=%d log=%d auth=%d", resp.LogSeq, resp.LogCount, resp.AuthCount)
	}
	if resp.SerialNum != "00255" {
		t.Errorf("SerialNum: got %q", resp.SerialNum)
	}
}

func TestTCPFrameRoundTrip(t *testing.T) {
	f := NewOpenDoorRequest(0xFFFF, 5, 2, 1500*time.Millisecond)
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

func TestAuthRightBitFields(t *testing.T) {
	orig := &AuthRight{
		CardNumber:  0x499602D2,
		ValidFrom:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Readers:  AllReaders,
		RemainCount: RemainUnlimited,
		Group:       5,
		Position:    2,
		PersonType:  9,
	}
	data, err := orig.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary: %v", err)
	}
	var parsed AuthRight
	if err := parsed.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if parsed.Group != 5 {
		t.Errorf("Group: got %d, want 5", parsed.Group)
	}
	if parsed.Position != 2 {
		t.Errorf("Position: got %d, want 2", parsed.Position)
	}
	if parsed.PersonType != 9 {
		t.Errorf("PersonType: got %d, want 9", parsed.PersonType)
	}
}

func TestAuthRightBitFieldsMaxValues(t *testing.T) {
	orig := &AuthRight{
		CardNumber:  1,
		RemainCount: 0x007F,
		Group:       7,
		Position:    3,
		PersonType:  15,
	}
	data, err := orig.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary: %v", err)
	}
	var parsed AuthRight
	if err := parsed.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if parsed.Group != 7 {
		t.Errorf("Group: got %d, want 7", parsed.Group)
	}
	if parsed.Position != 3 {
		t.Errorf("Position: got %d, want 3", parsed.Position)
	}
	if parsed.PersonType != 15 {
		t.Errorf("PersonType: got %d, want 15", parsed.PersonType)
	}
	if parsed.IsUnlimited() {
		t.Errorf("IsUnlimited: got true, want false for RemainCount=0x7f")
	}
}

func TestAuthRightBoolFlags(t *testing.T) {
	orig := &AuthRight{
		CardNumber:   1,
		IsName:       true,
		HasPackage:   true,
		HasDebt:      false,
		HasFlag1:     true,
		HasFlag2:     false,
		HasFlag3:     true,
		AntiPassback: false,
	}
	data, err := orig.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary: %v", err)
	}
	var parsed AuthRight
	if err := parsed.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if !parsed.IsName {
		t.Error("IsName: got false, want true")
	}
	if !parsed.HasPackage {
		t.Error("HasPackage: got false, want true")
	}
	if parsed.HasDebt {
		t.Error("HasDebt: got true, want false")
	}
	if !parsed.HasFlag1 {
		t.Error("HasFlag1: got false, want true")
	}
	if parsed.HasFlag2 {
		t.Error("HasFlag2: got true, want false")
	}
	if !parsed.HasFlag3 {
		t.Error("HasFlag3: got false, want true")
	}
	if parsed.AntiPassback {
		t.Error("AntiPassback: got true, want false")
	}
}

func TestAuthRightUnlimited(t *testing.T) {
	right := &AuthRight{RemainCount: RemainUnlimited}
	if !right.IsUnlimited() {
		t.Error("IsUnlimited should be true for RemainUnlimited")
	}
	right.RemainCount = 100
	if right.IsUnlimited() {
		t.Error("IsUnlimited should be false for RemainCount=100")
	}
}

func TestDirectionalRemain(t *testing.T) {
	dr := DirectionalRemain(1, 2)
	if dr != 60102 {
		t.Errorf("DirectionalRemain(1,2): got %d, want 60102", dr)
	}
}

func TestLogEntrySubTypeBitExtraction(t *testing.T) {
	raw := []byte{
		0x00, 0x00, 0x00, 0x00, 0x20, 0xfb, 0x6e, 0x20,
		0x9c, 0x24, 0x37, 0x5d, 0x29, 0x04, 0x09, 0x4D,
	}
	var e LogEntry
	if err := e.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if e.SubType != 6 {
		t.Errorf("SubType: got %d, want 6", e.SubType)
	}
	if !e.IsName {
		t.Errorf("IsName: got %v, want true", e.IsName)
	}
	if e.ExtReader != 1 {
		t.Errorf("ExtReader: got %d, want 1", e.ExtReader)
	}
}

func TestLogEntryCardNumberString(t *testing.T) {
	e := &LogEntry{CardNumber: 1234567890}
	if s := e.CardNumberString(); s != "1234567890" {
		t.Errorf("CardNumberString: got %q, want \"1234567890\"", s)
	}
}

func TestRevokeAuthCardNumber(t *testing.T) {
	f := NewRevokeAuthRequest(0xFFFF, 1, 0x12345678ABCDEF00)
	data := f.Data
	high := uint32(0x12345678)
	low := uint32(0xABCDEF00)
	gotHigh := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	gotLow := uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24
	if gotHigh != high {
		t.Errorf("card high: got 0x%08x, want 0x%08x", gotHigh, high)
	}
	if gotLow != low {
		t.Errorf("card low: got 0x%08x, want 0x%08x", gotLow, low)
	}
}

func TestEventHelpers(t *testing.T) {
	raw := []byte("0008242637|1|26419|4|2015-04-30 15:20:25|5|1|1|26419|||")
	evt, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if evt.Door() != 1 {
		t.Errorf("Door(): got %d, want 1", evt.Door())
	}
	if evt.Reader() != 5 {
		t.Errorf("Reader(): got %d, want 5", evt.Reader())
	}
	if evt.ResultCode() != 4 {
		t.Errorf("ResultCode(): got %d, want 4", evt.ResultCode())
	}
	if evt.Direction() != DirEntry {
		t.Errorf("Direction(): got %v, want DirEntry", evt.Direction())
	}
	tm, err := evt.Time()
	if err != nil {
		t.Fatalf("Time(): %v", err)
	}
	if tm.Year() != 2015 || tm.Month() != 4 || tm.Day() != 30 {
		t.Errorf("Time(): got %v", tm)
	}
}

func TestReadersConstants(t *testing.T) {
	if Reader1 != Readers(1<<0) {
		t.Errorf("Reader1: got %d, want %d", Reader1, Readers(1<<0))
	}
	if Reader2 != Readers(1<<1) {
		t.Errorf("Reader2: got %d, want %d", Reader2, Readers(1<<1))
	}
	if Reader3 != Readers(1<<2) {
		t.Errorf("Reader3: got %d, want %d", Reader3, Readers(1<<2))
	}
	if Reader4 != Readers(1<<3) {
		t.Errorf("Reader4: got %d, want %d", Reader4, Readers(1<<3))
	}
	if Reader5 != Readers(1<<4) {
		t.Errorf("Reader5: got %d, want %d", Reader5, Readers(1<<4))
	}
	if Reader6 != Readers(1<<5) {
		t.Errorf("Reader6: got %d, want %d", Reader6, Readers(1<<5))
	}
	if Reader7 != Readers(1<<6) {
		t.Errorf("Reader7: got %d, want %d", Reader7, Readers(1<<6))
	}
	if Reader8 != Readers(1<<7) {
		t.Errorf("Reader8: got %d, want %d", Reader8, Readers(1<<7))
	}
	if AllReaders != Readers(0xFF) {
		t.Errorf("AllReaders: got %d, want %d", AllReaders, Readers(0xFF))
	}
	all := Reader1 | Reader2 | Reader3 | Reader4 | Reader5 | Reader6 | Reader7 | Reader8
	if all != AllReaders {
		t.Errorf("all readers OR'd: got %d, want %d", all, AllReaders)
	}
}

func TestNewReaders(t *testing.T) {
	m := NewReaders(1, 3, 5)
	want := Reader1 | Reader3 | Reader5
	if m != want {
		t.Errorf("NewReaders(1,3,5): got %d, want %d", m, want)
	}
	m = NewReaders()
	if m != 0 {
		t.Errorf("NewReaders(): got %d, want 0", m)
	}
	m = NewReaders(1, 2, 3, 4, 5, 6, 7, 8)
	if m != AllReaders {
		t.Errorf("NewReaders(1-8): got %d, want %d", m, AllReaders)
	}
	m = NewReaders(9, 0, -1)
	if m != 0 {
		t.Errorf("NewReaders(out of range): got %d, want 0", m)
	}
}

func TestQueryAuth(t *testing.T) {
	right := &AuthRight{
		CardNumber:  0x499602D2,
		ValidFrom:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Schedule:    ScheduleAny,
		Readers:     AllReaders,
		RemainCount: RemainUnlimited,
	}
	data, _ := right.AppendBinary(nil)
	f := &Frame{
		Preamble: respPreamble,
		DeviceID: 0xFFFF,
		Seq:      8,
		Cmd:      CmdQueryAuthResp,
		Result:   byte(ResultSuccess),
		Data:     data,
	}
	parsed, err := ParseQueryAuthResponse(f)
	if err != nil {
		t.Fatalf("ParseQueryAuthResponse: %v", err)
	}
	if parsed.CardNumber != 0x499602D2 {
		t.Errorf("CardNumber: got 0x%08x, want 0x%08x", parsed.CardNumber, 0x499602D2)
	}
	if parsed.Readers != AllReaders {
		t.Errorf("Readers: got %d, want %d", parsed.Readers, AllReaders)
	}
	if parsed.Schedule != ScheduleAny {
		t.Errorf("Schedule: got %d, want %d", parsed.Schedule, ScheduleAny)
	}
}

func TestScheduleConstants(t *testing.T) {
	if ScheduleAny != 0 {
		t.Errorf("ScheduleAny: got %d, want 0", ScheduleAny)
	}
	if Schedule2 != 1<<1 {
		t.Errorf("Schedule2: got %d, want %d", Schedule2, 1<<1)
	}
	if Schedule8 != 1<<7 {
		t.Errorf("Schedule8: got %d, want %d", Schedule8, 1<<7)
	}
	combined := Schedule2 | Schedule3 | Schedule5
	if combined != Schedule(1<<1|1<<2|1<<4) {
		t.Errorf("combined schedules: got %d, want %d", combined, Schedule(1<<1|1<<2|1<<4))
	}
}

func TestResultCodeString(t *testing.T) {
	tests := []struct {
		code   ResultCode
		expect string
	}{
		{ResultSuccess, "Success"},
		{ResultNoPermission, "No permission"},
		{ResultAntiPassback, "Anti-passback"},
		{ResultIDExpired, "ID expired"},
	}
	for _, tt := range tests {
		if got := tt.code.String(); got != tt.expect {
			t.Errorf("ResultCode(%d).String(): got %q, want %q", tt.code, got, tt.expect)
		}
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		typ    byte
		expect string
	}{
		{EventTypeCardSwipe, "card_swipe"},
		{EventTypeHeartbeat, "heartbeat"},
		{EventTypeIDCard, "id_card"},
		{EventTypeSignalChange, "signal_change"},
		{EventTypeOpLog, "op_log"},
		{99, "unknown(99)"},
	}
	for _, tt := range tests {
		if got := EventTypeString(tt.typ); got != tt.expect {
			t.Errorf("EventTypeString(%d): got %q, want %q", tt.typ, got, tt.expect)
		}
	}
}

func TestIDCardParsing(t *testing.T) {
	prefix := make([]byte, 14)
	for i := range prefix {
		prefix[i] = 0xaa
	}
	unicodeData := make([]byte, 256)
	for i := range unicodeData {
		unicodeData[i] = 0x20 // space in UTF-16LE low byte
	}
	copy(unicodeData[0:2], []byte{0x41, 0x0}) // "A" in UTF-16LE
	payload := append(prefix, unicodeData...)
	payload = append(payload, []byte("WLftest")...)
	raw := make([]byte, 8+len(payload))
	raw[0] = rptHeaderMagic
	raw[1] = EventTypeIDCard
	raw[2] = byte(len(payload) >> 8)
	raw[3] = byte(len(payload) & 0xff)
	copy(raw[8:], payload)

	evt, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent ID card: %v", err)
	}
	if evt.Type != EventTypeIDCard {
		t.Errorf("Type: got %d, want %d", evt.Type, EventTypeIDCard)
	}
	if evt.IDCard == nil {
		t.Fatal("IDCard should not be nil")
	}
}

func TestAuthRightScheduleReaders(t *testing.T) {
	right := &AuthRight{
		CardNumber:  12345,
		ValidFrom:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Schedule:    Schedule2 | Schedule3,
		Readers:     Reader1 | Reader3 | Reader5,
		RemainCount: RemainUnlimited,
	}
	data, err := right.AppendBinary(nil)
	if err != nil {
		t.Fatalf("AppendBinary: %v", err)
	}
	var parsed AuthRight
	if err := parsed.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if parsed.Schedule != Schedule2|Schedule3 {
		t.Errorf("Schedule: got %d, want %d", parsed.Schedule, Schedule2|Schedule3)
	}
	if parsed.Readers != Reader1|Reader3|Reader5 {
		t.Errorf("Readers: got %d, want %d", parsed.Readers, Reader1|Reader3|Reader5)
	}
}
