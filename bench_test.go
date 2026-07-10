package s4a

import (
	"fmt"
	"testing"
	"time"
)

var benchCardSwipeRaw = []byte("0008242637|1|26419|4|2015-04-30 15:20:25|5|1|1|26419|||")

var benchHeartbeatRaw = func() []byte {
	payload := []byte("24884|120|33|1|TestController|7879047689384705|v1.0.0-test")
	raw := make([]byte, 8+len(payload))
	raw[0] = 0xc8
	raw[1] = EventTypeHeartbeat
	raw[2] = 0x00
	raw[3] = byte(len(payload))
	copy(raw[8:], payload)
	return raw
}()

func BenchmarkParseCardSwipeEvent(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseEvent(benchCardSwipeRaw)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseHeartbeatEvent(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := ParseEvent(benchHeartbeatRaw)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrameAppendBinary(b *testing.B) {
	f := NewOpenDoorRequest(0xFFFF, 4, 1, 3*time.Second)
	buf := make([]byte, 0, FrameSize(len(f.Data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ = f.AppendBinary(buf[:0])
	}
}

func BenchmarkFrameAppendBinaryAuthorize(b *testing.B) {
	right := &AuthRight{
		CardNumber:  0x12345678,
		ValidFrom:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Readers:  0xFF,
		RemainCount: RemainUnlimited,
	}
	f := NewAuthorizeRequest(0xFFFF, 1, right)
	buf := make([]byte, 0, FrameSize(len(f.Data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ = right.AppendBinary(buf[:0])
	}
}

func BenchmarkFrameUnmarshalBinary(b *testing.B) {
	f := NewOpenDoorRequest(0xFFFF, 4, 1, 3*time.Second)
	raw, _ := f.AppendBinary(nil)
	var ff Frame
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ff.UnmarshalBinary(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuthRightAppendBinary(b *testing.B) {
	right := &AuthRight{
		CardNumber:  0x12345678,
		ValidFrom:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Readers:  0xFF,
		RemainCount: RemainUnlimited,
	}
	buf := make([]byte, 0, 24)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, _ = right.AppendBinary(buf[:0])
	}
}

func BenchmarkAuthRightUnmarshalBinary(b *testing.B) {
	right := &AuthRight{
		CardNumber:  0x12345678,
		ValidFrom:   time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		ValidUntil:  time.Date(2030, 12, 31, 23, 59, 58, 0, time.Local),
		Readers:  0xFF,
		RemainCount: RemainUnlimited,
	}
	raw, _ := right.AppendBinary(nil)
	var ar AuthRight
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ar.UnmarshalBinary(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLogEntryUnmarshalBinary(b *testing.B) {
	raw := []byte{
		0x00, 0x00, 0x00, 0x00, 0x20, 0xfb, 0x6e, 0x20,
		0x9c, 0x24, 0x37, 0x5d, 0x29, 0x04, 0x09, 0x00,
	}
	var e LogEntry
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := e.UnmarshalBinary(raw); err != nil {
			b.Fatal(err)
		}
	}
}

var sinkFrame Frame
var sinkEvent *Event
var sinkAuthRight AuthRight
var sinkLogEntry LogEntry
var sinkStr string
var sinkBytes []byte

func init() {
	_ = fmt.Sprintf("%v %v %v %v %v %v", sinkFrame, sinkEvent, sinkAuthRight, sinkLogEntry, sinkStr, sinkBytes)
}
