package tapo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKlapLogin(t *testing.T) {
	server := &klapServer{t: t, username: "test@example.com", password: "test_password"}
	testServer, port := createKlapServer(t, server)
	defer testServer.Close()

	dc, err := createKlapDeviceConnection(server.username, server.password, "127.0.0.1", port)
	assert.NoError(t, err)
	assert.Equal(t, false, dc.hasExchangedKeys())

	err = dc.doKeyExchange()
	assert.NoError(t, err)
	assert.Equal(t, true, dc.hasExchangedKeys())
}
