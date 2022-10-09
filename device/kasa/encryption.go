package kasa

import (
	"errors"
	"strconv"
)

const initialPad byte = 171

func scramble(b []byte) []byte {
	var pad = initialPad
	buffer := make([]byte, 4+len(b))

	writeUInt32ToBufferBigEndian(buffer, uint32(len(b)))
	for i, ch := range b {
		pad = pad ^ ch
		buffer[i+4] = pad
	}
	return buffer
}

func unscramble(b []byte) ([]byte, error) {
	var pad = initialPad
	buffer := make([]byte, len(b)-4)

	expectedSize := expectedLinkiePacketSize(b)
	if expectedSize != len(b)-4 {
		return nil, errors.New("unexpected reply size: expected " + strconv.Itoa(expectedSize) +
			" bytes but received " + strconv.Itoa(len(b)-4) + " bytes")
	}
	for i, ch := range b[4:] {
		buffer[i] = byte(pad ^ ch)
		pad = ch
	}
	return buffer, nil
}

func expectedLinkiePacketSize(b []byte) int {
	return int(b[3]) + int(b[2])<<8 + int(b[1])<<16 + int(b[0])<<24
}

func writeUInt32ToBufferBigEndian(b []byte, i uint32) {
	b[0] = byte((i >> 24) & 0xff)
	b[1] = byte((i >> 16) & 0xff)
	b[2] = byte((i >> 8) & 0xff)
	b[3] = byte(i & 0xff)
}
