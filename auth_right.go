package s4a

import "encoding/binary"

type AuthRight struct {
	CardHigh    uint32
	CardLow     uint32
	BeginDate   uint16
	BeginTime   uint16
	EndDate     uint16
	EndTime     uint16
	TimeZone    uint8
	ReaderMask  uint8
	RemainCount uint16
	Flags       uint16
	Group       uint8
	Position    uint8
	PersonType  uint8
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
	binary.LittleEndian.PutUint32(b[off:off+4], a.CardHigh)
	binary.LittleEndian.PutUint32(b[off+4:off+8], a.CardLow)
	binary.LittleEndian.PutUint16(b[off+8:off+10], a.BeginDate)
	binary.LittleEndian.PutUint16(b[off+10:off+12], a.BeginTime)
	binary.LittleEndian.PutUint16(b[off+12:off+14], a.EndDate)
	binary.LittleEndian.PutUint16(b[off+14:off+16], a.EndTime)
	b[off+16] = a.TimeZone
	b[off+17] = a.ReaderMask
	binary.LittleEndian.PutUint16(b[off+18:off+20], a.RemainCount)
	binary.LittleEndian.PutUint16(b[off+20:off+22], a.Flags)
	b[off+22] = a.Group | (a.Position << 3) | (a.PersonType << 5)
	b[off+23] = 0
	return b, nil
}

func (a *AuthRight) UnmarshalBinary(data []byte) error {
	if len(data) < 24 {
		return ErrFrameTooShort
	}
	a.CardHigh = binary.LittleEndian.Uint32(data[0:4])
	a.CardLow = binary.LittleEndian.Uint32(data[4:8])
	a.BeginDate = binary.LittleEndian.Uint16(data[8:10])
	a.BeginTime = binary.LittleEndian.Uint16(data[10:12])
	a.EndDate = binary.LittleEndian.Uint16(data[12:14])
	a.EndTime = binary.LittleEndian.Uint16(data[14:16])
	a.TimeZone = data[16]
	a.ReaderMask = data[17]
	a.RemainCount = binary.LittleEndian.Uint16(data[18:20])
	a.Flags = binary.LittleEndian.Uint16(data[20:22])
	a.Group = data[22] & 0x07
	a.Position = (data[22] >> 3) & 0x03
	a.PersonType = (data[22] >> 5) & 0x0f
	return nil
}
