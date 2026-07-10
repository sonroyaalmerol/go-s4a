package s4a

import (
	"bytes"
	"testing"
	"time"
)

func TestFrameMarshalUnmarshal(t *testing.T) {
	f := NewOpenDoorRequest(0xFFFF, 4, 1, 300)
	raw := make([]byte, 0, FrameSize(len(f.Data)))
	raw, _ = f.AppendBinary(raw)

	expectedLen := FrameSize(3)
	if len(raw) != expectedLen {
		t.Fatalf("expected len %d, got %d", expectedLen, len(raw))
	}

	var decoded Frame
	if err := decoded.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if decoded.Cmd != CmdOpenDoor {
		t.Errorf("cmd: expected 0x10, got 0x%02x", decoded.Cmd)
	}
	if decoded.Seq != 4 {
		t.Errorf("seq: expected 4, got %d", decoded.Seq)
	}
	if len(decoded.Data) != 3 {
		t.Errorf("data len: expected 3, got %d", len(decoded.Data))
	}
}

func TestFrameMarshalRoundTrip(t *testing.T) {
	right := &AuthRight{
		CardLow:     0x12345678,
		BeginDate:   BCDDateEncode(2025, 1, 15),
		BeginTime:   0,
		EndDate:     BCDDateEncode(2030, 12, 31),
		EndTime:     BCDTimeEncode(23, 59, 58),
		ReaderMask:  0xFF,
		RemainCount: 0xFFFF,
	}

	f := NewAuthorizeRequest(0xFFFF, 1, right)
	raw := make([]byte, 0, FrameSize(len(f.Data)))
	raw, _ = f.AppendBinary(raw)

	var decoded Frame
	if err := decoded.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if decoded.Cmd != CmdAuthorize {
		t.Errorf("cmd: expected 0x12, got 0x%02x", decoded.Cmd)
	}

	var authRight AuthRight
	if err := authRight.UnmarshalBinary(decoded.Data); err != nil {
		t.Fatalf("UnmarshalBinary AuthRight: %v", err)
	}
	if authRight.CardLow != 0x12345678 {
		t.Errorf("CardLow: expected 0x12345678, got 0x%08x", authRight.CardLow)
	}
	if authRight.ReaderMask != 0xFF {
		t.Errorf("ReaderMask: expected 0xFF, got 0x%02x", authRight.ReaderMask)
	}
}

func TestOpenDoorKnownVectors(t *testing.T) {
	tests := []struct {
		name     string
		door     uint8
		duration uint16
		expected []byte
	}{
		{
			name:     "door1_3sec",
			door:     1,
			duration: 300,
			expected: append(
				reqPreamble[:],
				0xff, 0xff, 0x04, 0x00, 0x10, 0x00, 0x00, 0x03, 0x01, 0x01, 0x2c, 0x41,
			),
		},
		{
			name:     "door2_5sec",
			door:     2,
			duration: 500,
			expected: append(
				reqPreamble[:],
				0xff, 0xff, 0x04, 0x00, 0x10, 0x00, 0x00, 0x03, 0x02, 0x01, 0xf4, 0x0a,
			),
		},
		{
			name:     "door3_300ms",
			door:     3,
			duration: 30,
			expected: append(
				reqPreamble[:],
				0xff, 0xff, 0x04, 0x00, 0x10, 0x00, 0x00, 0x03, 0x03, 0x00, 0x1e, 0x34,
			),
		},
		{
			name:     "door4_restore_auto",
			door:     4,
			duration: OpenDoorRestoreAuto,
			expected: append(
				reqPreamble[:],
				0xff, 0xff, 0x04, 0x00, 0x10, 0x00, 0x00, 0x03, 0x04, 0xfd, 0xe9, 0xfd,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewOpenDoorRequest(0xFFFF, 4, tt.door, tt.duration)
			raw := make([]byte, 0, FrameSize(len(f.Data)))
			raw, _ = f.AppendBinary(raw)
			if !bytes.Equal(raw, tt.expected) {
				t.Errorf("\nexpected: %x\ngot:      %x", tt.expected, raw)
			}
		})
	}
}

func TestHeartbeatACK(t *testing.T) {
	expected := []byte{
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0x70, 0x00, 0x00, 0x00, 0x70,
	}
	if !bytes.Equal(HeartbeatACK, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, HeartbeatACK)
	}
}

func TestLogACK(t *testing.T) {
	ack := NewLogACK(0x6733, 4, 0x00000000)
	expected := []byte{
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0x67, 0x33, 0x04, 0x00, 0x44, 0x00, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00, 0x48,
	}
	if !bytes.Equal(ack, expected) {
		t.Errorf("\nexpected: %x\ngot:      %x", expected, ack)
	}
}

func TestCardSwipeEventParsing(t *testing.T) {
	raw := []byte("0008242637|1|26419|4|2015-04-30 15:20:25|5|1|1|26419|||")
	evt, err := ParseEvent(raw)
	if err != nil {
		t.Fatalf("ParseEvent: %v", err)
	}
	if evt.CardData != "0008242637" {
		t.Errorf("CardData: got %q", evt.CardData)
	}
	if evt.DoorNo != "1" {
		t.Errorf("DoorNo: got %q", evt.DoorNo)
	}
	if evt.ReaderNo != "5" {
		t.Errorf("ReaderNo: got %q", evt.ReaderNo)
	}
}

func TestBCDDateEncodeDecode(t *testing.T) {
	encoded := BCDDateEncode(2018, 4, 16)
	if encoded != 0x2490 {
		t.Errorf("BCDDateEncode(2018,4,16): expected 0x2490, got 0x%04x", encoded)
	}
	y, m, d := BCDDateDecode(encoded)
	if y != 2018 || m != 4 || d != 16 {
		t.Errorf("BCDDateDecode: got %d-%d-%d, want 2018-4-16", y, m, d)
	}
}

func TestBCDTimeEncodeDecode(t *testing.T) {
	encoded := BCDTimeEncode(11, 40, 26)
	if encoded != 0x5D0D {
		t.Errorf("BCDTimeEncode(11,40,26): expected 0x5D0D, got 0x%04x", encoded)
	}
	h, mi, s := BCDTimeDecode(encoded)
	if h != 11 || mi != 40 || s != 26 {
		t.Errorf("BCDTimeDecode: got %d:%d:%d, want 11:40:26", h, mi, s)
	}
}

func TestBuildTimeData(t *testing.T) {
	tm := time.Date(2025, 6, 9, 14, 30, 45, 0, time.UTC)
	td := BuildTimeData(tm)
	expected := [7]byte{25, 6, 9, 14, 30, 45, 1}
	if td != expected {
		t.Errorf("BuildTimeData: got %v, want %v", td, expected)
	}
}

func TestMonitorLogResponseParsing(t *testing.T) {
	data := make([]byte, 48)
	data[0] = 1
	data[20] = 2
	copy(data[43:48], []byte("00255"))

	f := &Frame{
		Preamble: respPreamble,
		DeviceID: 0xFFFF,
		Seq:      8,
		Cmd:      CmdMonitorLogResp,
		Result:   ResultSuccess,
		Data:     data,
	}

	resp, err := ParseMonitorLogResponse(f)
	if err != nil {
		t.Fatalf("ParseMonitorLogResponse: %v", err)
	}
	if resp.LogSeq != 1 {
		t.Errorf("LogSeq: expected 1, got %d", resp.LogSeq)
	}
	if resp.LogCount != 2 {
		t.Errorf("LogCount: expected 2, got %d", resp.LogCount)
	}
	if string(resp.DeviceFlag) != "00255" {
		t.Errorf("DeviceFlag: got %q", string(resp.DeviceFlag))
	}
}

func TestChecksumRejection(t *testing.T) {
	f := NewOpenDoorRequest(0xFFFF, 4, 1, 300)
	raw := make([]byte, 0, FrameSize(len(f.Data)))
	raw, _ = f.AppendBinary(raw)
	raw[len(raw)-1] ^= 0xFF

	err := new(Frame).UnmarshalBinary(raw)
	if err != ErrChecksumMismatch {
		t.Errorf("expected ErrChecksumMismatch, got %v", err)
	}
}

func TestUnmarshalLogEntry(t *testing.T) {
	raw := []byte{
		0x00, 0x00, 0x00, 0x00, 0x20, 0xfb, 0x6e, 0x20,
		0x9c, 0x24, 0x37, 0x5d, 0x29, 0x04, 0x09, 0x00,
	}
	var e LogEntry
	if err := e.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary LogEntry: %v", err)
	}
	if e.CardNumber() != 544144160 {
		t.Errorf("CardNumber: got %d, want 544144160", e.CardNumber())
	}
	if e.Date.Year() != 2018 || e.Date.Month() != 4 || e.Date.Day() != 28 {
		t.Errorf("Date: got %s, want 2018-04-28", e.Date.Format("2006-01-02"))
	}
	if e.Door != 1 {
		t.Errorf("Door: got %d, want 1", e.Door)
	}
	if e.Reader != 5 {
		t.Errorf("Reader: got %d, want 5", e.Reader)
	}
	if e.Result != 4 {
		t.Errorf("Result: got %d, want 4", e.Result)
	}
}

func TestUnmarshalLogEntryTooShort(t *testing.T) {
	var le LogEntry
	err := le.UnmarshalBinary(make([]byte, 10))
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestAuthRightBitFields(t *testing.T) {
	orig := &AuthRight{
		CardHigh:    0,
		CardLow:     0x499602D2,
		BeginDate:   BCDDateEncode(2025, 1, 15),
		EndDate:     BCDDateEncode(2030, 12, 31),
		ReaderMask:  0xFF,
		RemainCount: 0xFFFF,
		Flags:       0,
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
		CardLow:    1,
		Flags:      0x007F,
		Group:      7,
		Position:   3,
		PersonType: 15,
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
	if parsed.Flags != 0x007F {
		t.Errorf("Flags: got 0x%04x, want 0x007f", parsed.Flags)
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
	if e.IsName != 1 {
		t.Errorf("IsName: got %d, want 1", e.IsName)
	}
	if e.ExtReader != 1 {
		t.Errorf("ExtReader: got %d, want 1", e.ExtReader)
	}
}

func TestLogEntryCardNumberString(t *testing.T) {
	e := &LogEntry{CardHigh: 0, CardLow: 1234567890}
	if s := e.CardNumberString(); s != "1234567890" {
		t.Errorf("CardNumberString: got %q, want \"1234567890\"", s)
	}
}
