package s4a

import "errors"

var (
	ErrFrameTooShort    = errors.New("s4a: frame too short")
	ErrInvalidPreamble  = errors.New("s4a: invalid frame preamble")
	ErrChecksumMismatch = errors.New("s4a: checksum mismatch")
	ErrCommandFailed    = errors.New("s4a: command failed")
	ErrInvalidResponse  = errors.New("s4a: invalid response frame")
	ErrNotConnected     = errors.New("s4a: not connected")
	ErrReadTimeout      = errors.New("s4a: read timeout")
)
