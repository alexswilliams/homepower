package tapo_klap

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
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

	validatedSessions map[string]*testEncryption
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
	server.validatedSessions = map[string]*testEncryption{}

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
	hash := sha256.Sum256(append(append(bytes.Clone(clientSeed), serverSeed...), s.authHash...))

	http.SetCookie(writer, &http.Cookie{
		Name:    "TP_SESSIONID",
		Value:   sessionIdFromSeeds(clientSeed, serverSeed),
		Expires: time.Now().Add(24 * time.Hour),
	})
	writer.WriteHeader(http.StatusOK)
	written, err := writer.Write(append(bytes.Clone(serverSeed), hash[:]...))
	require.NoError(s.t, err)
	require.Equal(s.t, 48, written)
	s.t.Logf("Handshake 1 complete: client = %x, server = %x, session = %s", clientSeed, serverSeed, sessionIdFromSeeds(clientSeed, serverSeed))
}

func sessionIdFromSeeds(clientSeed []byte, serverSeed []byte) string {
	return base64.StdEncoding.EncodeToString(append(bytes.Clone(clientSeed), serverSeed...))
}
func (s *klapServer) handshake2(writer http.ResponseWriter, request *http.Request) {
	clientSeed, serverSeed := s.getSeedsFromCookie(request)
	challenge, err := io.ReadAll(request.Body)
	require.NoError(s.t, err)
	require.Len(s.t, challenge, 32)

	expected := sha256.Sum256(append(append(bytes.Clone(serverSeed), clientSeed...), s.authHash...))
	if !bytes.Equal(challenge, expected[:]) {
		http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	s.validatedSessions[sessionIdFromSeeds(clientSeed, serverSeed)], err =
		setupTestEncryption(s.t, append(append(bytes.Clone(clientSeed), serverSeed...), s.authHash...))
	require.NoError(s.t, err)
	s.t.Logf("Handshake 2 complete: marking session %s as valid\n", sessionIdFromSeeds(clientSeed, serverSeed))

	writer.WriteHeader(http.StatusOK)
	written, err := writer.Write([]byte{1})
	require.NoError(s.t, err)
	require.Equal(s.t, 1, written)
}

func (s *klapServer) handleRequest(writer http.ResponseWriter, request *http.Request) {
	if s.handler == nil {
		s.t.Fatalf("No handler specified in test\n")
	}
	clientSeed, serverSeed := s.getSeedsFromCookie(request)
	encryption, found := s.validatedSessions[sessionIdFromSeeds(clientSeed, serverSeed)]
	if !found {
		s.t.Errorf("Could not find encryption context linked to session")
		http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	requestBytes, err := io.ReadAll(request.Body)
	require.NoError(s.t, err)
	requestClearText, err := encryption.Decrypt(s.t, requestBytes)
	require.NoError(s.t, err)

	var requestBody struct {
		Method string `json:"method"`
		Params string `json:"params"`
	}
	err = json.Unmarshal(requestClearText, &requestBody)
	if err != nil {
		s.failWithCode(1003, writer)
		return
	}

	responseBody, err := s.handler(s.t, requestBody.Method, requestBody.Params)
	require.NoError(s.t, err)

	responseCipherText := encryption.Encrypt(responseBody)
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(responseCipherText)
	require.NoError(s.t, err)

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

func (s *klapServer) getSeedsFromCookie(request *http.Request) ([]byte, []byte) {
	sessionCookie, err := request.Cookie("TP_SESSIONID")
	require.NoError(s.t, err)
	seeds, err := base64.StdEncoding.DecodeString(sessionCookie.Value)
	require.NoError(s.t, err)
	require.Len(s.t, seeds, 32)
	clientSeed := seeds[0:16]
	serverSeed := seeds[16:32]
	return clientSeed, serverSeed
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

type testEncryption struct {
	block          cipher.Block
	iv             []byte
	signature      []byte
	sequenceNumber int32
}

func setupTestEncryption(t *testing.T, localRemoteAuthBuffer []byte) (*testEncryption, error) {
	keyHash := sha256.Sum256(append([]byte("lsk"), localRemoteAuthBuffer...))
	ivHash := sha256.Sum256(append([]byte("iv"), localRemoteAuthBuffer...))
	sequence := int32(binary.BigEndian.Uint32(ivHash[sha256.Size-4 : sha256.Size]))
	sigHash := sha256.Sum256(append([]byte("ldk"), localRemoteAuthBuffer...))
	aesCipher, err := aes.NewCipher(keyHash[:16])
	if err != nil {
		return nil, err
	}
	t.Logf("Starting Sequence: %x\n", sequence)
	return &testEncryption{
		block:          aesCipher,
		iv:             ivHash[:12],
		signature:      sigHash[:28],
		sequenceNumber: sequence,
	}, nil
}

func (ec *testEncryption) getIv() []byte {
	return binary.BigEndian.AppendUint32(bytes.Clone(ec.iv), uint32(ec.sequenceNumber))
}

func (ec *testEncryption) sign(cipherText []byte) []byte {
	hash := sha256.Sum256(
		append(binary.BigEndian.AppendUint32(bytes.Clone(ec.signature), uint32(ec.sequenceNumber)), cipherText...))
	return hash[:]
}

func (ec *testEncryption) Encrypt(data []byte) []byte {
	padded, _ := pkcs7.Pad(data, aes.BlockSize)
	cipherText := make([]byte, len(padded))
	cipher.NewCBCEncrypter(ec.block, ec.getIv()).CryptBlocks(cipherText, padded)
	return append(ec.sign(cipherText), cipherText...)
}

func (ec *testEncryption) Decrypt(t *testing.T, cipherText []byte) ([]byte, error) {
	ec.sequenceNumber++
	if len(cipherText) < 32 {
		return nil, fmt.Errorf("cipherText must be at least 32 bytes")
	}
	signature := ec.sign(cipherText[32:])
	require.EqualValues(t, signature, cipherText[0:32], "Bad signature for message received by server")

	plainText := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(ec.block, ec.getIv()).CryptBlocks(plainText, cipherText[32:])
	return pkcs7.Unpad(plainText[:len(plainText)-32], aes.BlockSize)
}
