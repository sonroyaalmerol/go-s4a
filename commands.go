package s4a

import (
	"encoding/binary"
	"fmt"
	"time"
)

func durationToWire(d time.Duration) uint16 {
	if d == 0 {
		return 0
	}
	return uint16(d / (10 * time.Millisecond))
}

type DoorControl uint16

const (
	RestoreAuto DoorControl = 65001
	KeepOpen    DoorControl = 65002
	KeepClosed  DoorControl = 65003
	PulseClose  DoorControl = 65004
	PulseOpen   DoorControl = 65005
)

func NewRequestFrame(deviceID uint16, seq uint16, cmd byte, data []byte) *Frame {
	dataLen := len(data)
	buf := make([]byte, 32+dataLen+1)
	copy(buf[0:24], reqPreamble[:])
	binary.BigEndian.PutUint16(buf[24:26], deviceID)
	binary.LittleEndian.PutUint16(buf[26:28], seq)
	buf[28] = cmd
	buf[29] = 0x00
	binary.BigEndian.PutUint16(buf[30:32], uint16(dataLen))
	copy(buf[32:], data)
	buf[32+dataLen] = frameChecksum(cmd, 0x00, data)
	return &Frame{
		Preamble: reqPreamble,
		DeviceID: deviceID,
		Seq:      seq,
		Cmd:      cmd,
		Result:   0x00,
		Data:     data,
	}
}

func NewOpenDoorRequest(deviceID uint16, seq uint16, door uint8, duration time.Duration) *Frame {
	wireDur := durationToWire(duration)
	data := []byte{door, byte(wireDur >> 8), byte(wireDur & 0xff)}
	return NewRequestFrame(deviceID, seq, CmdOpenDoor, data)
}

func NewControlDoorRequest(deviceID uint16, seq uint16, door uint8, cmd DoorControl) *Frame {
	wireCmd := uint16(cmd)
	data := []byte{door, byte(wireCmd >> 8), byte(wireCmd & 0xff)}
	return NewRequestFrame(deviceID, seq, CmdOpenDoor, data)
}

func ParseOpenDoorResponse(f *Frame) error {
	if f.Cmd != CmdOpenDoorResp {
		return fmt.Errorf("%w: expected cmd 0x11, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	return nil
}

func NewAuthorizeRequest(deviceID uint16, seq uint16, right *AuthRight) *Frame {
	data, _ := right.AppendBinary(make([]byte, 0, 24))
	return NewRequestFrame(deviceID, seq, CmdAuthorize, data)
}

func ParseAuthorizeResponse(f *Frame) error {
	if f.Cmd != CmdAuthorizeResp {
		return fmt.Errorf("%w: expected cmd 0x13, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	return nil
}

func NewRevokeAuthRequest(deviceID uint16, seq uint16, cardNumber uint64) *Frame {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[0:4], uint32(cardNumber>>32))
	binary.LittleEndian.PutUint32(data[4:8], uint32(cardNumber&0xFFFFFFFF))
	return NewRequestFrame(deviceID, seq, CmdRevokeAuth, data)
}

func ParseRevokeAuthResponse(f *Frame) error {
	if f.Cmd != CmdRevokeAuthResp {
		return fmt.Errorf("%w: expected cmd 0x15, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	return nil
}

func NewClearAuthRequest(deviceID uint16, seq uint16) *Frame {
	return NewRequestFrame(deviceID, seq, CmdClearAuth, nil)
}

func ParseClearAuthResponse(f *Frame) error {
	if f.Cmd != CmdClearAuthResp {
		return fmt.Errorf("%w: expected cmd 0x19, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	return nil
}

func NewQueryAuthRequest(deviceID uint16, seq uint16, position uint32) *Frame {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[0:4], position)
	binary.LittleEndian.PutUint32(data[4:8], 1)
	return NewRequestFrame(deviceID, seq, CmdQueryAuth, data)
}

type MonitorLogResponse struct {
	LogSeq    uint32
	Log       LogEntry
	LogCount  uint32
	AuthCount uint32

	CurrentTime time.Time
	ReaderRelay []byte
	SerialNum   string
}

func NewMonitorLogRequest(deviceID uint16, seq uint16, index uint32) *Frame {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, index)
	return NewRequestFrame(deviceID, seq, CmdMonitorLog, data)
}

func ParseMonitorLogResponse(f *Frame) (*MonitorLogResponse, error) {
	if f.Cmd != CmdMonitorLogResp {
		return nil, fmt.Errorf("%w: expected cmd 0x39, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return nil, fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	d := f.Data
	if len(d) < 48 {
		return nil, fmt.Errorf("%w: monitor response too short (%d bytes)", ErrInvalidResponse, len(d))
	}
	r := &MonitorLogResponse{}
	r.LogSeq = binary.LittleEndian.Uint32(d[0:4])
	if err := r.Log.UnmarshalBinary(d[4:20]); err != nil {
		return nil, fmt.Errorf("parse log entry: %w", err)
	}
	r.LogCount = binary.LittleEndian.Uint32(d[20:24])
	r.AuthCount = binary.LittleEndian.Uint32(d[24:28])
	r.CurrentTime = time.Date(
		2000+int(d[28]), time.Month(d[29]), int(d[30]),
		int(d[31]), int(d[32]), int(d[33]),
		0, time.Local,
	)
	r.ReaderRelay = make([]byte, 8)
	copy(r.ReaderRelay, d[35:43])
	r.SerialNum = string(d[43:48])
	return r, nil
}

func NewSetTimeRequest(deviceID uint16, seq uint16, t time.Time) *Frame {
	td := BuildTimeData(t)
	return NewRequestFrame(deviceID, seq, CmdSetTime, td[:])
}

func ParseSetTimeResponse(f *Frame) error {
	if f.Cmd != CmdSetTimeResp {
		return fmt.Errorf("%w: expected cmd 0x27, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	return nil
}

func ParseQueryAuthResponse(f *Frame) (*AuthRight, error) {
	if f.Cmd != CmdQueryAuthResp {
		return nil, fmt.Errorf("%w: expected cmd 0x35, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return nil, fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	var ar AuthRight
	if err := ar.UnmarshalBinary(f.Data); err != nil {
		return nil, err
	}
	return &ar, nil
}

func ParseTextCommandResponse(f *Frame) error {
	if f.Cmd != CmdTextCommandResp {
		return fmt.Errorf("%w: expected cmd 0x95, got 0x%02x", ErrInvalidResponse, f.Cmd)
	}
	if f.Result != ResultSuccess {
		return fmt.Errorf("%w: result 0x%02x", ErrCommandFailed, f.Result)
	}
	return nil
}

func NewTextCommandRequest(deviceID uint16, seq uint16, command string) *Frame {
	data := make([]byte, 8+512)
	copy(data[8:], command)
	return NewRequestFrame(deviceID, seq, CmdTextCommand, data)
}

var HeartbeatACK = []byte{
	0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff,
	0x70, 0x00, 0x00, 0x00, 0x70,
}

var LogACKPrefix = []byte{
	0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

func NewLogACK(deviceID uint16, seq uint16, logSeq uint32) []byte {
	buf := make([]byte, 37)
	copy(buf[0:24], LogACKPrefix)
	binary.BigEndian.PutUint16(buf[24:26], deviceID)
	binary.LittleEndian.PutUint16(buf[26:28], seq)
	buf[28] = 0x44
	buf[29] = 0x00
	binary.BigEndian.PutUint16(buf[30:32], 0x0004)
	binary.LittleEndian.PutUint32(buf[32:36], logSeq)
	var sum byte
	for i := 28; i < 36; i++ {
		sum += buf[i]
	}
	buf[36] = sum
	return buf
}
