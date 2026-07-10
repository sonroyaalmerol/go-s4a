package s4a

import (
	"encoding/binary"
	"fmt"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	EventTypeEvent        byte = 1
	EventTypeCardSwipe    byte = 2
	EventTypeIDCard       byte = 3
	EventTypeHeartbeat    byte = 4
	EventTypeDebug        byte = 5
	EventTypeSignalChange byte = 6
	EventTypeOpLog        byte = 7
	EventTypePullAuth     byte = 8
	EventTypeAuthResult   byte = 9
	EventTypeGetTime      byte = 10
)

func EventTypeString(t byte) string {
	switch t {
	case EventTypeEvent:
		return "event"
	case EventTypeCardSwipe:
		return "card_swipe"
	case EventTypeIDCard:
		return "id_card"
	case EventTypeHeartbeat:
		return "heartbeat"
	case EventTypeDebug:
		return "debug"
	case EventTypeSignalChange:
		return "signal_change"
	case EventTypeOpLog:
		return "op_log"
	case EventTypePullAuth:
		return "pull_auth"
	case EventTypeAuthResult:
		return "auth_result"
	case EventTypeGetTime:
		return "get_time"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

type IDCardData struct {
	Name        string
	Gender      string
	Ethnicity   string
	BirthDate   string
	Address     string
	IDNumber    string
	IssuingAuth string
	Expiry      string
	PhotoData   []byte
}

type Event struct {
	Type       byte
	Seq        uint32
	RawHeader  []byte
	RawPayload []byte

	CardData     string
	CardType     string
	DeviceID     string
	Result       string
	SwipeTime    string
	ReaderNo     string
	DoorNo       string
	DirectionStr string
	DeviceName   string
	LogType      string
	LogSubType   string
	IDCardChip   string

	IDCard *IDCardData

	HBControllerFlag  string
	HBTimeoutConfig   string
	HBTimeoutRemain   string
	HBTimeoutCount    string
	HBControllerName  string
	HBGlobalFlag      string
	HBFirmwareVersion string

	SCPrevSignals    string
	SCCurrSignals    string
	SCControllerFlag string
	SCChangeTime     string
	SCPeerAddr       string
}

const rptHeaderMagic byte = 0xc8

func ParseEvent(raw []byte) (*Event, error) {
	if len(raw) < 1 {
		return nil, ErrFrameTooShort
	}

	if raw[0] != rptHeaderMagic {
		evt := &Event{Type: EventTypeCardSwipe}
		parsePipeFields(raw, pipeFieldCount, func(idx int, start, end int) {
			assignCardSwipeField(evt, idx, b2s(raw[start:end]))
		})
		evt.RawPayload = raw
		return evt, nil
	}

	if len(raw) < 8 {
		return nil, ErrFrameTooShort
	}

	evt := &Event{}
	evt.Type = raw[1]
	evt.Seq = binary.LittleEndian.Uint32(raw[4:8])
	evt.RawHeader = raw[:8]

	dataLen := int(raw[2])<<8 | int(raw[3])
	payloadEnd := min(8+dataLen, len(raw))

	if payloadEnd > 8 {
		evt.RawPayload = raw[8:payloadEnd]
		switch evt.Type {
		case EventTypeHeartbeat:
			parsePipeFields(evt.RawPayload, 7, func(idx int, start, end int) {
				assignHeartbeatField(evt, idx, b2s(raw[8+start:8+end]))
			})
		case EventTypeSignalChange:
			parsePipeFields(evt.RawPayload, 13, func(idx int, start, end int) {
				assignSignalField(evt, idx, b2s(raw[8+start:8+end]))
			})
			semiIdx := byteIndex(evt.RawPayload, ';')
			pipeIdx := byteIndex(evt.RawPayload, '|')
			if semiIdx >= 0 && (pipeIdx < 0 || semiIdx < pipeIdx) {
				evt.SCPrevSignals = b2s(evt.RawPayload[:semiIdx])
				if pipeIdx >= 0 {
					evt.SCCurrSignals = b2s(evt.RawPayload[semiIdx+1 : pipeIdx])
				} else {
					evt.SCCurrSignals = b2s(evt.RawPayload[semiIdx+1:])
				}
			}
		case EventTypeCardSwipe:
			parsePipeFields(evt.RawPayload, 9, func(idx int, start, end int) {
				assignLogField(evt, idx, b2s(raw[8+start:8+end]))
			})
		case EventTypeIDCard:
			parseIDCardPayload(evt)
		case EventTypeEvent, EventTypeDebug, EventTypeOpLog, EventTypePullAuth, EventTypeAuthResult, EventTypeGetTime:
			parsePipeFields(evt.RawPayload, pipeFieldCount, func(idx int, start, end int) {
				assignCardSwipeField(evt, idx, b2s(evt.RawPayload[start:end]))
			})
		}
	}
	return evt, nil
}

func parseIDCardPayload(evt *Event) {
	p := evt.RawPayload
	if len(p) < 14 {
		return
	}
	prefix := p[:14]
	if !isIDCardPrefix(prefix) {
		parsePipeFields(evt.RawPayload, pipeFieldCount, func(idx int, start, end int) {
			assignCardSwipeField(evt, idx, b2s(evt.RawPayload[start:end]))
		})
		return
	}

	rest := p[14:]
	if len(rest) < 256 {
		evt.IDCard = &IDCardData{PhotoData: rest}
		return
	}

	unicodeData := rest[:256]
	text := decodeUTF16LE(unicodeData)

	photoStart := 256
	photoEnd := len(rest)
	if photoStart+4 < photoEnd && rest[photoStart] == 'W' && rest[photoStart+1] == 'L' && rest[photoStart+2] == 'f' {
		photoEnd = len(rest)
	}

	evt.IDCard = &IDCardData{
		Name:        extractIDCardField(text, 0, 30),
		Gender:      extractIDCardField(text, 30, 2),
		Ethnicity:   extractIDCardField(text, 32, 4),
		BirthDate:   extractIDCardField(text, 36, 16),
		Address:     extractIDCardField(text, 52, 70),
		IDNumber:    extractIDCardField(text, 122, 36),
		IssuingAuth: extractIDCardField(text, 158, 30),
		Expiry:      extractIDCardField(text, 188, 32),
		PhotoData:   rest[photoStart:photoEnd],
	}
}

func isIDCardPrefix(p []byte) bool {
	for _, b := range p {
		if b != 0xaa {
			return false
		}
	}
	return true
}

func decodeUTF16LE(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	u16 := make([]uint16, len(data)/2)
	for i := range u16 {
		u16[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
	}
	runes := utf16.Decode(u16)
	return string(runes)
}

func extractIDCardField(text string, start, length int) string {
	runes := []rune(text)
	end := min(start+length, len(runes))
	if start >= len(runes) {
		return ""
	}
	result := string(runes[start:end])
	for len(result) > 0 && result[len(result)-1] == 0 {
		result = result[:len(result)-1]
	}
	return result
}

const pipeFieldCount = 12

func parsePipeFields(data []byte, maxFields int, fn func(idx int, start, end int)) {
	idx := 0
	start := 0
	for i, b := range data {
		if b == '|' {
			fn(idx, start, i)
			idx++
			start = i + 1
			if idx >= maxFields {
				return
			}
		}
	}
	if start < len(data) && idx < maxFields {
		fn(idx, start, len(data))
	}
}

func byteIndex(data []byte, c byte) int {
	for i, b := range data {
		if b == c {
			return i
		}
	}
	return -1
}

func assignCardSwipeField(evt *Event, idx int, val string) {
	switch idx {
	case 0:
		evt.CardData = val
	case 1:
		evt.CardType = val
	case 2:
		evt.DeviceID = val
	case 3:
		evt.Result = val
	case 4:
		evt.SwipeTime = val
	case 5:
		evt.ReaderNo = val
	case 6:
		evt.DoorNo = val
	case 7:
		evt.DirectionStr = val
	case 8:
		evt.DeviceName = val
	case 9:
		evt.LogType = val
	case 10:
		evt.LogSubType = val
	case 11:
		evt.IDCardChip = val
	}
}

func assignHeartbeatField(evt *Event, idx int, val string) {
	switch idx {
	case 0:
		evt.HBControllerFlag = val
	case 1:
		evt.HBTimeoutConfig = val
	case 2:
		evt.HBTimeoutRemain = val
	case 3:
		evt.HBTimeoutCount = val
	case 4:
		evt.HBControllerName = val
	case 5:
		evt.HBGlobalFlag = val
	case 6:
		evt.HBFirmwareVersion = val
	}
}

func assignSignalField(evt *Event, idx int, val string) {
	switch idx {
	case 1:
		evt.SCControllerFlag = val
	case 2:
		evt.SCChangeTime = val
	case 12:
		evt.SCPeerAddr = val
	}
}

func assignLogField(evt *Event, idx int, val string) {
	switch idx {
	case 0:
		evt.CardData = val
	case 1:
		evt.LogType = val
	case 3:
		evt.Result = val
	case 4:
		evt.SwipeTime = val
	case 5:
		evt.ReaderNo = val
	case 6:
		evt.DoorNo = val
	case 7:
		evt.DirectionStr = val
	case 8:
		evt.DeviceName = val
	}
}

func (e *Event) Door() uint8 {
	return parseUintField(e.DoorNo)
}

func (e *Event) Reader() uint8 {
	return parseUintField(e.ReaderNo)
}

func (e *Event) ResultCode() uint8 {
	return parseUintField(e.Result)
}

func (e *Event) Direction() Direction {
	switch parseUintField(e.DirectionStr) {
	case 1:
		return DirEntry
	case 2:
		return DirExit
	default:
		return DirUnknown
	}
}

func (e *Event) Time() (time.Time, error) {
	if e.SwipeTime == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	return time.ParseInLocation("2006-01-02 15:04:05", e.SwipeTime, time.Local)
}

func parseUintField(s string) uint8 {
	var n uint8
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint8(c-'0')
		}
	}
	return n
}

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func BCDDateDecode(v uint16) (year int, month time.Month, day int) {
	year = int(v>>9) + 2000
	month = time.Month((v >> 5) & 0x0f)
	day = int(v & 0x1f)
	return
}

func BCDTimeDecode(v uint16) (hour, minute, second int) {
	hour = int(v >> 11)
	minute = int((v >> 5) & 0x3f)
	second = int(v&0x1f) * 2
	return
}

func BCDDateEncode(year int, month time.Month, day int) uint16 {
	return uint16((year-2000)<<9 | int(month)<<5 | day)
}

func BCDTimeEncode(hour, minute, second int) uint16 {
	return uint16(hour<<11 | minute<<5 | second/2)
}

func BuildTimeData(t time.Time) [7]byte {
	var d [7]byte
	d[0] = byte(t.Year() - 2000)
	d[1] = byte(t.Month())
	d[2] = byte(t.Day())
	d[3] = byte(t.Hour())
	d[4] = byte(t.Minute())
	d[5] = byte(t.Second())
	d[6] = byte(t.Weekday())
	return d
}
