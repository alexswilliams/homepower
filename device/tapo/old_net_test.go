package tapo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOldLogin(t *testing.T) {
	server := &oldServer{t: t, username: "test@example.com", password: "test_password"}
	testServer, port := createOldServer(t, server)
	defer testServer.Close()

	dc, err := createOldTapoDeviceConnection(server.username, server.password, "127.0.0.1", port)
	assert.NoError(t, err)
	assert.Equal(t, false, dc.hasExchangedKeys())
	assert.Equal(t, false, dc.isLoggedIn())

	err = dc.doLogin()
	assert.NoError(t, err)
	assert.Equal(t, true, dc.hasExchangedKeys())
	assert.Equal(t, true, dc.isLoggedIn())
}
