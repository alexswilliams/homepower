package kasa

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

func openConnection(host string, port uint16) net.Conn {
	dialer := &net.Dialer{Timeout: 2 * time.Second}
	connection, err := dialer.Dial("tcp", host+":"+strconv.Itoa(int(port)))
	if err != nil {
		panic("Could not open connection: " + err.Error())
	}
	return connection
}

func queryDevice(connection net.Conn, request string) (response []byte) {
	err := connection.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		panic("Could not set write timeout on connection: " + err.Error())
	}
	scrambledText := scramble([]byte(request))
	bytesWritten, err := connection.Write(scrambledText)
	if err != nil || bytesWritten != len(scrambledText) {
		panic("Could not write command to connection")
	}

	err = connection.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		panic("Could not set read timeout on connection: " + err.Error())
	}
	buffer := make([]byte, 2048)

	bytesRead, err := connection.Read(buffer)
	buffer = buffer[:bytesRead]
	if err != nil || bytesRead < 8 {
		panic("Could not read first response packet: " + err.Error())
	}
	expectedSize := expectedLinkiePacketSize(buffer)
	for len(buffer) < expectedSize && len(buffer) <= 4096 {
		tmpBuffer := make([]byte, 2048)
		bytesRead, err := connection.Read(tmpBuffer)
		tmpBuffer = tmpBuffer[:bytesRead]
		if err != nil {
			panic("Could not read from connection: " + err.Error())
		}
		buffer = append(buffer, tmpBuffer...)
	}
	response = unscramble(buffer)
	fmt.Printf("%s\n\n", response)
	return
}

func scramble(b []byte) []byte {
	var iv byte = 171
	buffer := make([]byte, 4+len(b))

	writeUInt32ToBufferBigEndian(buffer, uint32(len(b)))
	for i, ch := range b {
		iv = byte(iv ^ ch)
		buffer[i+4] = iv
	}
	return buffer
}

func unscramble(b []byte) []byte {
	var iv byte = 171
	buffer := make([]byte, len(b)-4)

	expectedSize := expectedLinkiePacketSize(b)
	if expectedSize != len(b)-4 {
		panic("Unexpected reply size - expected " + strconv.Itoa(expectedSize) +
			" bytes but received " + strconv.Itoa(len(b)-4) + " bytes")
	}
	for i, ch := range b[4:] {
		buffer[i] = byte(iv ^ ch)
		iv = ch
	}
	return buffer
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
