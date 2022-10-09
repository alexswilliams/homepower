package kasa

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type deviceConnection struct {
	address      string
	dialer       *net.Dialer
	connection   net.Conn
	writeTimeout time.Duration
	readTimeout  time.Duration
}

func newDeviceConnection(ip string, port uint16) *deviceConnection {
	return &deviceConnection{
		address:      ip + ":" + strconv.Itoa(int(port)),
		dialer:       &net.Dialer{Timeout: 1 * time.Second},
		connection:   nil,
		writeTimeout: 1 * time.Second,
		readTimeout:  2 * time.Second,
	}
}

func (dc *deviceConnection) closeCurrentConnection() {
	if dc.connection != nil {
		_ = dc.connection.Close()
		dc.connection = nil
	}
}
func (dc *deviceConnection) openNewConnection() error {
	dc.closeCurrentConnection()
	connection, err := dc.dialer.Dial("tcp", dc.address)
	if err != nil {
		return fmt.Errorf("could not dial address: %w", err)
	}
	dc.connection = connection
	return nil
}

func (dc *deviceConnection) queryDevice(request string) ([]byte, error) {
	if err := dc.connection.SetWriteDeadline(time.Now().Add(dc.writeTimeout)); err != nil {
		return nil, fmt.Errorf("could not set write timeout: %w", err)
	}
	scrambledText := scramble([]byte(request))
	if bytesWritten, err := dc.connection.Write(scrambledText); err != nil || bytesWritten != len(scrambledText) {
		return nil, fmt.Errorf("could not write to socket: %w", err)
	}

	if err := dc.connection.SetReadDeadline(time.Now().Add(dc.readTimeout)); err != nil {
		return nil, fmt.Errorf("could not set read timeout: %w", err)
	}
	if buffer, err := dc._readLinkieResponse(); err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	} else {
		return unscramble(buffer)
	}
}

func (dc *deviceConnection) _readLinkieResponse() ([]byte, error) {
	buffer := make([]byte, 2048)

	bytesRead, err := dc.connection.Read(buffer)
	buffer = buffer[:bytesRead]
	if err != nil || bytesRead < 8 {
		return nil, fmt.Errorf("could not read first response packet: %w", err)
	}
	expectedSize := expectedLinkiePacketSize(buffer)

	for len(buffer) < expectedSize && len(buffer) <= 8192 {
		tmpBuffer := make([]byte, 2048)
		if bytesRead, err := dc.connection.Read(tmpBuffer); err == nil {
			tmpBuffer = tmpBuffer[:bytesRead]
			buffer = append(buffer, tmpBuffer...)
		} else {
			return nil, fmt.Errorf("could not read subsequent response packet: %w", err)
		}
	}
	return buffer, nil
}
