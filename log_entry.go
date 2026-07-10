package s4a

import (
	"encoding/binary"
	"fmt"
	"time"
)

type LogType uint8

const (
	LogTypeEvent     LogType = 1
	LogTypeSwipe     LogType = 2
	LogTypeOperation LogType = 3
)

func (t LogType) String() string {
	switch t {
	case LogTypeEvent:
		return "Event"
	case LogTypeSwipe:
		return "Card swipe"
	case LogTypeOperation:
		return "Operation"
	default:
		return "Unknown"
	}
}

type Direction uint8

const (
	DirUnknown Direction = 0
	DirEntry   Direction = 1
	DirExit    Direction = 2
)

func (d Direction) String() string {
	switch d {
	case DirEntry:
		return "Entry"
	case DirExit:
		return "Exit"
	default:
		return "Unknown"
	}
}

type LogEntry struct {
	CardNumber uint64
	Date       time.Time
	Door       uint8
	Reader     uint8
	Result     uint8
	Direction  Direction
	LogType    LogType
	SubType    uint8
	IsName     bool
	ExtReader  uint8
}

func (e *LogEntry) UnmarshalBinary(data []byte) error {
	if len(data) < 16 {
		return fmt.Errorf("%w: need 16 bytes for xLog, got %d", ErrFrameTooShort, len(data))
	}
	cardHigh := binary.LittleEndian.Uint32(data[0:4])
	cardLow := binary.LittleEndian.Uint32(data[4:8])
	e.CardNumber = uint64(cardHigh)<<32 | uint64(cardLow)
	dateVal := binary.LittleEndian.Uint16(data[8:10])
	timeVal := binary.LittleEndian.Uint16(data[10:12])
	yr, mo, dy := BCDDateDecode(dateVal)
	hr, mi, se := BCDTimeDecode(timeVal)
	e.Date = time.Date(yr, mo, dy, hr, mi, se, 0, time.Local)
	e.Door = data[12] & 0x07
	e.Reader = data[12] >> 3
	e.Result = data[13]
	e.Direction = Direction(data[14] & 0x03)
	e.LogType = LogType(data[14] >> 2)
	e.SubType = (data[15] >> 1) & 0x1f
	e.IsName = data[15]&0x01 != 0
	e.ExtReader = (data[15] >> 6) & 0x03
	return nil
}

func (e *LogEntry) IsCardSwipe() bool { return e.LogType == LogTypeSwipe }

func (e *LogEntry) ResultDescription() string { return ControllerErrorCode(e.Result) }

func (e *LogEntry) CardNumberString() string {
	return fmt.Sprintf("%d", e.CardNumber)
}
