package s4a

import "encoding/binary"

const (
	DefaultCommandPort = 65534
	DefaultEventPort   = 50000
	DefaultDeviceID    = 0xFFFF
)

var (
	reqPreamble = [24]byte{
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
	respPreamble = [24]byte{
		0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa,
		0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
)

const (
	CmdOpenDoor    byte = 0x10
	CmdAuthorize   byte = 0x12
	CmdRevokeAuth  byte = 0x14
	CmdClearAuth   byte = 0x18
	CmdQueryAuth   byte = 0x34
	CmdMonitorLog  byte = 0x38
	CmdSetTime     byte = 0x26
	CmdTextCommand byte = 0x94

	CmdOpenDoorResp    byte = 0x11
	CmdAuthorizeResp   byte = 0x13
	CmdRevokeAuthResp  byte = 0x15
	CmdClearAuthResp   byte = 0x19
	CmdQueryAuthResp   byte = 0x35
	CmdMonitorLogResp  byte = 0x39
	CmdSetTimeResp     byte = 0x27
	CmdTextCommandResp byte = 0x95
)

const FrameOverhead = 24 + 2 + 2 + 1 + 1 + 2 + 1

func FrameSize(dataLen int) int { return FrameOverhead + dataLen }

type Frame struct {
	Preamble [24]byte
	DeviceID uint16
	Seq      uint16
	Cmd      byte
	Result   byte
	Data     []byte
}

func (f *Frame) AppendBinary(b []byte) ([]byte, error) {
	dataLen := len(f.Data)
	n := len(b) + FrameOverhead + dataLen
	if cap(b) < n {
		b2 := make([]byte, len(b), n)
		copy(b2, b)
		b = b2
	}
	b = b[:n]
	off := n - FrameOverhead - dataLen

	copy(b[off:off+24], f.Preamble[:])
	off += 24
	binary.BigEndian.PutUint16(b[off:], f.DeviceID)
	off += 2
	binary.LittleEndian.PutUint16(b[off:], f.Seq)
	off += 2
	b[off] = f.Cmd
	off++
	b[off] = f.Result
	off++
	binary.BigEndian.PutUint16(b[off:], uint16(dataLen))
	off += 2
	if dataLen > 0 {
		copy(b[off:off+dataLen], f.Data)
	}

	var sum byte
	start := off - 4
	end := start + 4 + dataLen
	for i := start; i < end; i++ {
		sum += b[i]
	}
	b[end] = sum
	return b, nil
}

func (f *Frame) UnmarshalBinary(raw []byte) error {
	const minLen = FrameOverhead
	if len(raw) < minLen {
		return ErrFrameTooShort
	}
	if raw[0] != 0x55 && raw[0] != 0xaa {
		return ErrInvalidPreamble
	}

	copy(f.Preamble[:], raw[0:24])
	f.DeviceID = binary.BigEndian.Uint16(raw[24:26])
	f.Seq = binary.LittleEndian.Uint16(raw[26:28])
	f.Cmd = raw[28]
	f.Result = raw[29]
	dataLen := binary.BigEndian.Uint16(raw[30:32])

	if len(raw) < int(32+dataLen+1) {
		return ErrFrameTooShort
	}

	var sum byte
	for i := 28; i < int(32+dataLen); i++ {
		sum += raw[i]
	}
	if sum != raw[32+dataLen] {
		return ErrChecksumMismatch
	}

	if dataLen > 0 {
		f.Data = raw[32 : 32+dataLen]
	} else {
		f.Data = nil
	}
	return nil
}

func (f *Frame) MarshalTCP() []byte {
	b, _ := f.AppendBinary(nil)
	out := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(out[0:4], uint32(len(b)))
	copy(out[4:], b)
	return out
}

func frameChecksum(cmd, result byte, data []byte) byte {
	var sum byte
	sum = cmd + result + byte(len(data)>>8) + byte(len(data))
	for _, b := range data {
		sum += b
	}
	return sum
}
