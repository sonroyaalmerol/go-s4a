package s4a

import (
	"encoding/binary"
	"time"
)

const RemainUnlimited uint16 = 0xFFFF

type Schedule uint8

const (
	ScheduleAny Schedule = 0
	Schedule2   Schedule = 1 << 1
	Schedule3   Schedule = 1 << 2
	Schedule4   Schedule = 1 << 3
	Schedule5   Schedule = 1 << 4
	Schedule6   Schedule = 1 << 5
	Schedule7   Schedule = 1 << 6
	Schedule8   Schedule = 1 << 7
)

type Readers uint8

const (
	Reader1       Readers = 1 << 0
	Reader2       Readers = 1 << 1
	Reader3       Readers = 1 << 2
	Reader4       Readers = 1 << 3
	Reader5       Readers = 1 << 4
	Reader6       Readers = 1 << 5
	Reader7       Readers = 1 << 6
	Reader8       Readers = 1 << 7
	AllReaders    Readers = 0xFF
)

func NewReaders(readers ...int) Readers {
	var m Readers
	for _, r := range readers {
		if r >= 1 && r <= 8 {
			m |= 1 << (r - 1)
		}
	}
	return m
}

func DirectionalRemain(entry, exit int) uint16 {
	return uint16(60000 + entry*100 + exit)
}

type AuthRight struct {
	CardNumber  uint64
	ValidFrom   time.Time
	ValidUntil  time.Time
	Schedule    Schedule
	Readers  Readers
	RemainCount uint16

	IsName       bool
	HasPackage   bool
	HasDebt      bool
	HasFlag1     bool
	HasFlag2     bool
	HasFlag3     bool
	AntiPassback bool

	Group      uint8
	Position   uint8
	PersonType uint8
}

func (a *AuthRight) IsUnlimited() bool {
	return a.RemainCount == RemainUnlimited
}

func (a *AuthRight) AppendBinary(b []byte) ([]byte, error) {
	n := len(b) + 24
	if cap(b) < n {
		b2 := make([]byte, len(b), n)
		copy(b2, b)
		b = b2
	}
	b = b[:n]
	off := n - 24
	binary.LittleEndian.PutUint32(b[off:off+4], uint32(a.CardNumber>>32))
	binary.LittleEndian.PutUint32(b[off+4:off+8], uint32(a.CardNumber&0xFFFFFFFF))
	binary.LittleEndian.PutUint16(b[off+8:off+10], BCDDateEncode(a.ValidFrom.Year(), a.ValidFrom.Month(), a.ValidFrom.Day()))
	binary.LittleEndian.PutUint16(b[off+10:off+12], BCDTimeEncode(a.ValidFrom.Hour(), a.ValidFrom.Minute(), a.ValidFrom.Second()))
	binary.LittleEndian.PutUint16(b[off+12:off+14], BCDDateEncode(a.ValidUntil.Year(), a.ValidUntil.Month(), a.ValidUntil.Day()))
	binary.LittleEndian.PutUint16(b[off+14:off+16], BCDTimeEncode(a.ValidUntil.Hour(), a.ValidUntil.Minute(), a.ValidUntil.Second()))
	b[off+16] = byte(a.Schedule)
	b[off+17] = byte(a.Readers)
	binary.LittleEndian.PutUint16(b[off+18:off+20], a.RemainCount)
	flags := uint16(0)
	if a.IsName {
		flags |= 1 << 0
	}
	if a.HasPackage {
		flags |= 1 << 1
	}
	if a.HasDebt {
		flags |= 1 << 2
	}
	if a.HasFlag1 {
		flags |= 1 << 3
	}
	if a.HasFlag2 {
		flags |= 1 << 4
	}
	if a.HasFlag3 {
		flags |= 1 << 5
	}
	if a.AntiPassback {
		flags |= 1 << 6
	}
	bits := uint32(flags) | uint32(a.Group)<<23 | uint32(a.Position)<<26 | uint32(a.PersonType)<<28
	binary.LittleEndian.PutUint32(b[off+20:off+24], bits)
	return b, nil
}

func (a *AuthRight) UnmarshalBinary(data []byte) error {
	if len(data) < 24 {
		return ErrFrameTooShort
	}
	cardHigh := binary.LittleEndian.Uint32(data[0:4])
	cardLow := binary.LittleEndian.Uint32(data[4:8])
	a.CardNumber = uint64(cardHigh)<<32 | uint64(cardLow)
	beginDate := binary.LittleEndian.Uint16(data[8:10])
	beginTime := binary.LittleEndian.Uint16(data[10:12])
	endDate := binary.LittleEndian.Uint16(data[12:14])
	endTime := binary.LittleEndian.Uint16(data[14:16])
	yr, mo, dy := BCDDateDecode(beginDate)
	hr, mi, se := BCDTimeDecode(beginTime)
	a.ValidFrom = time.Date(yr, mo, dy, hr, mi, se, 0, time.Local)
	yr, mo, dy = BCDDateDecode(endDate)
	hr, mi, se = BCDTimeDecode(endTime)
	a.ValidUntil = time.Date(yr, mo, dy, hr, mi, se, 0, time.Local)
	a.Schedule = Schedule(data[16])
	a.Readers = Readers(data[17])
	a.RemainCount = binary.LittleEndian.Uint16(data[18:20])
	bits := binary.LittleEndian.Uint32(data[20:24])
	flags := uint16(bits & 0xffff)
	a.IsName = flags&(1<<0) != 0
	a.HasPackage = flags&(1<<1) != 0
	a.HasDebt = flags&(1<<2) != 0
	a.HasFlag1 = flags&(1<<3) != 0
	a.HasFlag2 = flags&(1<<4) != 0
	a.HasFlag3 = flags&(1<<5) != 0
	a.AntiPassback = flags&(1<<6) != 0
	a.Group = uint8((bits >> 23) & 0x07)
	a.Position = uint8((bits >> 26) & 0x03)
	a.PersonType = uint8((bits >> 28) & 0x0f)
	return nil
}
