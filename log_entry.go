package s4a

import (
	"encoding/binary"
	"fmt"
	"time"
)

type LogEntry struct {
	CardHigh uint32
	CardLow  uint32
	Date     time.Time
	Door     uint8
	Reader   uint8
	Result   uint8
	Dir      uint8
	Type     uint8
	SubType  uint8
}

func (e *LogEntry) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("%w: need 16 bytes for xLog, got %d", ErrFrameTooShort, len(data))
	}
	e.CardHigh = binary.LittleEndian.Uint32(data[0:4])
	e.CardLow = binary.LittleEndian.Uint32(data[4:8])
	dateVal := binary.LittleEndian.Uint16(data[8:10])
	timeVal := binary.LittleEndian.Uint16(data[10:12])
	yr, mo, dy := BCDDateDecode(dateVal)
	hr, mi, se := BCDTimeDecode(timeVal)
	e.Date = time.Date(yr, mo, dy, hr, mi, se, 0, time.Local)
	e.Door = data[12] & 0x07
	e.Reader = data[12] >> 3
	e.Result = data[13]
	e.Dir = data[14] & 0x03
	e.Type = data[14] >> 2
	e.SubType = data[15] & 0x1f
	return nil
}

func (e *LogEntry) CardNumber() uint64 {
	return uint64(e.CardHigh)<<32 | uint64(e.CardLow)
}

func (e *LogEntry) IsCardSwipe() bool { return e.Type == 2 }

func (e *LogEntry) CardNumberString() string {
	return fmt.Sprintf("%d", e.CardNumber())
}
