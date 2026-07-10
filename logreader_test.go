package s4a

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestLogEntryToRecord(t *testing.T) {
	entry := &LogEntry{
		CardNumber: 12345,
		Door:       2,
		Reader:     3,
		Direction:  DirEntry,
		Date:       time.Date(2025, 7, 9, 15, 30, 0, 0, time.Local),
		Result:     0,
		LogType:    LogTypeSwipe,
		IsName:     false,
	}

	rec := logEntryToRecord(entry, 42)
	if rec.CardNumber != 12345 {
		t.Errorf("CardNumber: got %d, want 12345", rec.CardNumber)
	}
	if rec.Door != 2 {
		t.Errorf("Door: got %d, want 2", rec.Door)
	}
	if rec.Reader != 3 {
		t.Errorf("Reader: got %d, want 3", rec.Reader)
	}
	if rec.Direction != "Entry" {
		t.Errorf("Direction: got %q, want Entry", rec.Direction)
	}
	if rec.Seq != 42 {
		t.Errorf("Seq: got %d, want 42", rec.Seq)
	}
	if rec.ResultDesc != "Success" {
		t.Errorf("ResultDesc: got %q, want Success", rec.ResultDesc)
	}
	if rec.LogType != "Card swipe" {
		t.Errorf("LogType: got %q, want Card swipe", rec.LogType)
	}
}

func TestLogRecordJSON(t *testing.T) {
	rec := LogRecord{
		CardNumber: 999,
		Door:       1,
		Reader:     2,
		Direction:  "Entry",
		Time:       time.Date(2025, 7, 9, 15, 30, 0, 0, time.UTC),
		Result:     0,
		ResultDesc: "Success",
		LogType:    "Card swipe",
		Seq:        10,
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"card_number":999`) {
		t.Errorf("JSON missing card_number: %s", data)
	}
	if !strings.Contains(string(data), `"door":1`) {
		t.Errorf("JSON missing door: %s", data)
	}

	var parsed LogRecord
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed.CardNumber != 999 {
		t.Errorf("roundtrip CardNumber: got %d, want 999", parsed.CardNumber)
	}
	if parsed.Door != 1 {
		t.Errorf("roundtrip Door: got %d, want 1", parsed.Door)
	}
}

func TestLogReaderSaveRestore(t *testing.T) {
	reader := NewLogReader(LogReaderConfig{})
	reader.seq = 42

	var buf bytes.Buffer
	if err := reader.Save(&buf); err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(buf.String(), "42") {
		t.Errorf("Save wrote %q, want starting with 42", buf.String())
	}

	reader2 := NewLogReader(LogReaderConfig{})
	rdr := bytes.NewReader(buf.Bytes())
	seq, err := reader2.Restore(rdr)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 42 {
		t.Errorf("Restore: got %d, want 42", seq)
	}
	if reader2.Seq() != 42 {
		t.Errorf("Restore Seq(): got %d, want 42", reader2.Seq())
	}
}

func TestLogReaderSeekSeq(t *testing.T) {
	reader := NewLogReader(LogReaderConfig{StartSeq: 10})
	if reader.Seq() != 10 {
		t.Errorf("initial Seq: got %d, want 10", reader.Seq())
	}

	reader.SeekSeq(99)
	if reader.Seq() != 99 {
		t.Errorf("after SeekSeq: got %d, want 99", reader.Seq())
	}
}

func TestLogReaderDefaultStartSeq(t *testing.T) {
	reader := NewLogReader(LogReaderConfig{})
	if reader.Seq() != 1 {
		t.Errorf("default StartSeq: got %d, want 1", reader.Seq())
	}
}

func TestLogReaderConfigDefaults(t *testing.T) {
	cfg := LogReaderConfig{Client: nil}
	cfg.defaults()
	if cfg.Logger == nil {
		t.Error("Logger should not be nil after defaults()")
	}
}

func TestLogReaderErr(t *testing.T) {
	reader := NewLogReader(LogReaderConfig{})
	if err := reader.Err(); err != nil {
		t.Errorf("initial Err should be nil, got %v", err)
	}
}

func TestLogRecordResultDescription(t *testing.T) {
	rec := LogRecord{
		CardNumber: 123,
		Result:     4,
		ResultDesc: "No permission",
	}
	if rec.ResultDesc != "No permission" {
		t.Errorf("ResultDesc: got %q, want No permission", rec.ResultDesc)
	}
}

type fakeServer struct {
	ln    net.PacketConn
	addr  string
	logs  []LogEntry
	count uint32
	done  chan struct{}
}

func startFakeServer(t *testing.T, logs []LogEntry) *fakeServer {
	t.Helper()
	ln, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{
		ln:    ln,
		addr:  ln.LocalAddr().String(),
		logs:  logs,
		count: uint32(len(logs)),
		done:  make(chan struct{}),
	}
	go s.serve()
	return s
}

func (s *fakeServer) serve() {
	buf := make([]byte, 4096)
	for {
		s.ln.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, peer, err := s.ln.ReadFrom(buf)
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		_ = n

		f := &Frame{}
		if err := f.UnmarshalBinary(buf[:n]); err != nil {
			continue
		}
		if f.Cmd != CmdMonitorLog {
			continue
		}

		seq := binary.LittleEndian.Uint32(f.Data[0:4])

		var respData []byte
		if seq == 0 || seq > s.count {
			respData = make([]byte, 48)
			binary.LittleEndian.PutUint32(respData[0:4], 0)
			binary.LittleEndian.PutUint32(respData[20:24], s.count)
			binary.LittleEndian.PutUint32(respData[24:28], s.count)
		} else {
			entry := s.logs[seq-1]
			entryData := make([]byte, 16)
			binary.LittleEndian.PutUint32(entryData[0:4], uint32(entry.CardNumber>>32))
			binary.LittleEndian.PutUint32(entryData[4:8], uint32(entry.CardNumber&0xFFFFFFFF))
			binary.LittleEndian.PutUint16(entryData[8:10], BCDDateEncode(entry.Date.Year(), entry.Date.Month(), entry.Date.Day()))
			binary.LittleEndian.PutUint16(entryData[10:12], BCDTimeEncode(entry.Date.Hour(), entry.Date.Minute(), entry.Date.Second()))
			entryData[12] = entry.Door | entry.Reader<<3
			entryData[13] = entry.Result
			entryData[14] = uint8(entry.Direction) | uint8(entry.LogType)<<2
			entryData[15] = (entry.SubType << 1)
			if entry.IsName {
				entryData[15] |= 0x01
			}
			entryData[15] |= entry.ExtReader << 6
			respData = make([]byte, 48)
			binary.LittleEndian.PutUint32(respData[0:4], seq)
			copy(respData[4:20], entryData)
			binary.LittleEndian.PutUint32(respData[20:24], s.count)
			binary.LittleEndian.PutUint32(respData[24:28], s.count)
		}

		resp := &Frame{
			Preamble: respPreamble,
			DeviceID: f.DeviceID,
			Seq:      f.Seq,
			Cmd:      CmdMonitorLogResp,
			Result:   byte(ResultSuccess),
			Data:     respData,
		}
		out, _ := resp.AppendBinary(nil)
		s.ln.WriteTo(out, peer)
	}
}

func (s *fakeServer) Close() {
	close(s.done)
	s.ln.Close()
}

func (s *fakeServer) Client(t *testing.T) *Client {
	t.Helper()
	client, err := NewClient(s.addr)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestLogReaderNextWithFakeServer(t *testing.T) {
	logs := []LogEntry{
		{
			CardNumber: 111,
			Door:       1,
			Reader:     1,
			Direction:  DirEntry,
			Date:       time.Date(2025, 7, 9, 10, 0, 0, 0, time.Local),
			Result:     0,
			LogType:    LogTypeSwipe,
		},
		{
			CardNumber: 222,
			Door:       2,
			Reader:     2,
			Direction:  DirExit,
			Date:       time.Date(2025, 7, 9, 11, 0, 0, 0, time.Local),
			Result:     4,
			LogType:    LogTypeSwipe,
		},
	}

	srv := startFakeServer(t, logs)
	defer srv.Close()
	client := srv.Client(t)
	defer client.Close()

	reader := NewLogReader(LogReaderConfig{Client: client})
	ctx := context.Background()

	if !reader.Next(ctx) {
		t.Fatalf("Next() returned false: %v", reader.Err())
	}
	rec := reader.Record()
	if rec.CardNumber != 111 {
		t.Errorf("first record CardNumber: got %d, want 111", rec.CardNumber)
	}
	if rec.Door != 1 {
		t.Errorf("first record Door: got %d, want 1", rec.Door)
	}

	if !reader.Next(ctx) {
		t.Fatalf("Next() returned false: %v", reader.Err())
	}
	rec = reader.Record()
	if rec.CardNumber != 222 {
		t.Errorf("second record CardNumber: got %d, want 222", rec.CardNumber)
	}

	if reader.Next(ctx) {
		t.Error("expected Next() to return false after all logs consumed")
	}
}

func TestLogReaderReadWithFakeServer(t *testing.T) {
	logs := []LogEntry{
		{
			CardNumber: 555,
			Door:       3,
			Reader:     1,
			Direction:  DirEntry,
			Date:       time.Date(2025, 7, 9, 12, 0, 0, 0, time.Local),
			Result:     0,
			LogType:    LogTypeSwipe,
		},
	}

	srv := startFakeServer(t, logs)
	defer srv.Close()
	client := srv.Client(t)
	defer client.Close()

	reader := NewLogReader(LogReaderConfig{Client: client})
	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	if err != nil {
		t.Fatalf("io.Copy: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"card_number":555`) {
		t.Errorf("output missing card_number: %s", output)
	}
	if !strings.Contains(output, `"door":3`) {
		t.Errorf("output missing door: %s", output)
	}

	var rec LogRecord
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("invalid JSON: %v\nline: %s", err, line)
		}
	}
	if rec.CardNumber != 555 {
		t.Errorf("roundtrip CardNumber: got %d, want 555", rec.CardNumber)
	}
}

func TestLogReaderWriteToWithFakeServer(t *testing.T) {
	logs := []LogEntry{
		{
			CardNumber: 777,
			Door:       1,
			Reader:     2,
			Direction:  DirExit,
			Date:       time.Date(2025, 7, 9, 14, 0, 0, 0, time.Local),
			Result:     0,
			LogType:    LogTypeSwipe,
		},
	}

	srv := startFakeServer(t, logs)
	defer srv.Close()
	client := srv.Client(t)
	defer client.Close()

	reader := NewLogReader(LogReaderConfig{Client: client})
	var buf bytes.Buffer
	_, err := reader.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"card_number":777`) {
		t.Errorf("output missing card_number: %s", output)
	}
}

func TestLogReaderSaveRestoreRoundTrip(t *testing.T) {
	reader := NewLogReader(LogReaderConfig{StartSeq: 100})
	var buf bytes.Buffer
	if err := reader.Save(&buf); err != nil {
		t.Fatal(err)
	}

	reader2 := NewLogReader(LogReaderConfig{})
	rdr := bytes.NewReader(buf.Bytes())
	seq, err := reader2.Restore(rdr)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 100 {
		t.Errorf("Restore: got %d, want 100", seq)
	}
}
