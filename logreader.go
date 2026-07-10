package s4a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

type LogReaderConfig struct {
	Client *Client

	StartSeq uint32

	Logger *slog.Logger
}

func (c *LogReaderConfig) defaults() {
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}

type LogRecord struct {
	CardNumber uint64    `json:"card_number"`
	Door       uint8     `json:"door"`
	Reader     uint8     `json:"reader"`
	Direction  string    `json:"direction"`
	Time       time.Time `json:"time"`
	Result     uint8     `json:"result"`
	ResultDesc string    `json:"result_desc"`
	LogType    string    `json:"log_type"`
	IsName     bool      `json:"is_name"`
	Seq        uint32    `json:"seq"`
}

func logEntryToRecord(entry *LogEntry, seq uint32) LogRecord {
	return LogRecord{
		CardNumber: entry.CardNumber,
		Door:       entry.Door,
		Reader:     entry.Reader,
		Direction:  entry.Direction.String(),
		Time:       entry.Date,
		Result:     entry.Result,
		ResultDesc: entry.ResultDescription(),
		LogType:    entry.LogType.String(),
		IsName:     entry.IsName,
		Seq:        seq,
	}
}

type LogReader struct {
	cfg    LogReaderConfig
	client *Client

	mu      sync.Mutex
	seq     uint32
	current LogRecord
	err     error
	done    bool

	buf bytes.Buffer
}

func NewLogReader(cfg LogReaderConfig) *LogReader {
	cfg.defaults()
	seq := cfg.StartSeq
	if seq == 0 {
		seq = 1
	}
	return &LogReader{
		cfg:    cfg,
		client: cfg.Client,
		seq:    seq,
	}
}

func (r *LogReader) Next(ctx context.Context) bool {
	if r.err != nil || r.done {
		return false
	}

	resp, err := r.client.MonitorLog(ctx, r.seq)
	if err != nil {
		r.err = fmt.Errorf("download log seq %d: %w", r.seq, err)
		return false
	}

	if resp.LogCount == 0 || resp.LogSeq == 0 {
		r.done = true
		return false
	}

	r.current = logEntryToRecord(&resp.Log, resp.LogSeq)
	r.seq = resp.LogSeq + 1
	return true
}

func (r *LogReader) Record() LogRecord {
	return r.current
}

func (r *LogReader) Err() error {
	return r.err
}

func (r *LogReader) Seq() uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seq
}

func (r *LogReader) SeekSeq(seq uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq = seq
	r.done = false
	r.err = nil
}

func (r *LogReader) WriteTo(w io.Writer) (int64, error) {
	var total int64
	enc := json.NewEncoder(w)
	ctx := context.Background()
	for r.Next(ctx) {
		enc.Encode(r.current)
	}
	return total, r.err
}

func (r *LogReader) Read(p []byte) (int, error) {
	if r.buf.Len() > 0 {
		return r.buf.Read(p)
	}

	ctx := context.Background()
	if !r.Next(ctx) {
		if r.err != nil {
			return 0, r.err
		}
		return 0, io.EOF
	}

	line, err := json.Marshal(r.current)
	if err != nil {
		return 0, fmt.Errorf("marshal log record: %w", err)
	}
	r.buf.Write(line)
	r.buf.WriteByte('\n')
	return r.buf.Read(p)
}

func (r *LogReader) Restore(pos io.ReadSeeker) (uint32, error) {
	data, err := io.ReadAll(pos)
	if err != nil {
		return 0, fmt.Errorf("read position: %w", err)
	}
	var seq uint32
	if _, err := fmt.Sscanf(string(data), "%d", &seq); err != nil {
		return 0, fmt.Errorf("parse position %q: %w", string(data), err)
	}
	r.SeekSeq(seq)
	return seq, nil
}

func (r *LogReader) Save(pos io.Writer) error {
	_, err := fmt.Fprintf(pos, "%d\n", r.Seq())
	return err
}
