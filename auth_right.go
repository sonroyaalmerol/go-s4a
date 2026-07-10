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
	bits := uint32(a.Flags) | uint32(a.Group)<<23 | uint32(a.Position)<<26 | uint32(a.PersonType)<<28
	binary.LittleEndian.PutUint32(b[off+20:off+24], bits)
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
	bits := binary.LittleEndian.Uint32(data[20:24])
	a.Flags = uint16(bits & 0xffff)
	a.Group = uint8((bits >> 23) & 0x07)
	a.Position = uint8((bits >> 26) & 0x03)
	a.PersonType = uint8((bits >> 28) & 0x0f)
	return nil
}
