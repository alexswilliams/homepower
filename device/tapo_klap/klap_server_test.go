package tapo_klap

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/mergermarket/go-pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type klapServer struct {
	t *testing.T

	username string
	password string
	authHash []byte

	validatedSessions map[string]bool
	handler           func(t *testing.T, method string, params any) ([]byte, error)
}

func createKlapServer(t *testing.T, server *klapServer) (*httptest.Server, uint16) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /app", server.respondWith1003)
	mux.HandleFunc("POST /app/handshake1", server.handshake1)
	mux.HandleFunc("POST /app/handshake2", server.handshake2)
	mux.HandleFunc("POST /app/request", server.handleRequest)
	testServer := httptest.NewServer(mux)
	port, err := strconv.Atoi(strings.Split(testServer.URL, ":")[2])

	userHash := sha1.Sum([]byte(server.username))
	passHash := sha1.Sum([]byte(server.password))
	authHash := sha256.Sum256(append(userHash[:], passHash[:]...))
	server.authHash = authHash[:]

	assert.NoError(t, err)
	return testServer, uint16(port)
}

func (s *klapServer) respondWith1003(writer http.ResponseWriter, request *http.Request) {
	s.failWithCode(1003, writer)
}
func (s *klapServer) handshake1(writer http.ResponseWriter, request *http.Request) {
	clientSeed, err := io.ReadAll(request.Body)
	require.NoError(s.t, err)
	require.Len(s.t, clientSeed, 16)
	serverSeed := s.generateNewServerSeed()
	buffer := append(append(bytes.Clone(clientSeed), serverSeed...), s.authHash...)
	hash := sha256.Sum256(buffer)
	writer.WriteHeader(http.StatusOK)
	written, err := writer.Write(append(bytes.Clone(serverSeed), hash[:]...))
	require.NoError(s.t, err)
	require.Equal(s.t, 48, written)
}
func (s *klapServer) handshake2(writer http.ResponseWriter, request *http.Request) {
	s.t.Fatalf("Unimplemented")
}

func (s *klapServer) handleRequest(writer http.ResponseWriter, request *http.Request) {
	s.t.Fatalf("Not yet implemented\n")
}

func (s *klapServer) assertNoErrorOrFailWithCode(originalError error, writer http.ResponseWriter, code int) bool {
	if !assert.NoError(s.t, originalError) {
		s.failWithCode(code, writer)
		return true
	}
	return false
}

func (s *klapServer) failWithCode(code int, writer http.ResponseWriter) {
	responseBytes := s.failureForCode(code)
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write(responseBytes)
	require.NoError(s.t, err)
}

func (s *klapServer) failureForCode(code int) []byte {
	responseBytes, err := json.Marshal(struct {
		ErrorCode int `json:"error_code"`
	}{ErrorCode: code})
	require.NoError(s.t, err)
	return responseBytes
}

func (s *klapServer) getKeyDataFromCookie(writer http.ResponseWriter, request *http.Request) ([]byte, cipher.Block, []byte) {
	sessionCookie, err := request.Cookie("TP_SESSIONID")
	var innerKeyRand []byte
	if errors.Is(err, http.ErrNoCookie) {
		innerKeyRand = s.generateNewServerSeed()
		http.SetCookie(writer, &http.Cookie{
			Name:    "TP_SESSIONID",
			Value:   base64.StdEncoding.EncodeToString(innerKeyRand),
			Expires: time.Now().Add(86400 * time.Second),
		})
	} else {
		innerKeyRand, err = base64.StdEncoding.DecodeString(sessionCookie.Value)
		require.NoError(s.t, err)
		require.Len(s.t, innerKeyRand, 16)
	}
	aesCipher, err := aes.NewCipher(innerKeyRand[0:16])
	require.NoError(s.t, err)
	iv := innerKeyRand[16:32]
	require.Len(s.t, iv, aesCipher.BlockSize())
	return innerKeyRand, aesCipher, iv
}

func (s *klapServer) generateNewServerSeed() []byte {
	serverSeed := make([]byte, 16)
	bytesGenerated, err := rand.Read(serverSeed)
	require.NoError(s.t, err)
	require.Equal(s.t, 16, bytesGenerated)
	return serverSeed
}

func (s *klapServer) decrypt(base64CipherText string, aesCipher cipher.Block, iv []byte) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(base64CipherText)
	require.NoError(s.t, err)
	clearText := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(aesCipher, iv).CryptBlocks(clearText, cipherText)
	return pkcs7.Unpad(clearText, len(iv))
}

func (s *klapServer) encrypt(clearText []byte, aesCipher cipher.Block, iv []byte) []byte {
	padded, err := pkcs7.Pad(clearText, len(iv))
	require.NoError(s.t, err)
	cipherText := make([]byte, len(padded))
	cipher.NewCBCEncrypter(aesCipher, iv).CryptBlocks(cipherText, padded)
	return cipherText
}
