package tapo

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"github.com/mergermarket/go-pkcs7"
	"github.com/mitchellh/mapstructure"
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

type oldServer struct {
	t        *testing.T
	username string
	password string
	handler  func(t *testing.T, method string, params any) ([]byte, error)
}

func createOldServer(t *testing.T, server *oldServer) (*httptest.Server, uint16) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /app", server.handleRequest)
	mux.Handle("POST /app/handshake1", http.NotFoundHandler())
	mux.Handle("POST /app/handshake2", http.NotFoundHandler())
	testServer := httptest.NewServer(mux)
	port, err := strconv.Atoi(strings.Split(testServer.URL, ":")[2])
	assert.NoError(t, err)
	return testServer, uint16(port)
}

func (s *oldServer) assertNoErrorOrFailWithCode(originalError error, writer http.ResponseWriter, code int) bool {
	if !assert.NoError(s.t, originalError) {
		s.failWithCode(code, writer)
		return true
	}
	return false
}

func (s *oldServer) failWithCode(code int, writer http.ResponseWriter) {
	responseBytes := s.failureForCode(code)
	writer.WriteHeader(http.StatusOK)
	_, err := writer.Write(responseBytes)
	require.NoError(s.t, err)
}

func (s *oldServer) failureForCode(code int) []byte {
	responseBytes, err := json.Marshal(struct {
		ErrorCode int `json:"error_code"`
	}{ErrorCode: code})
	require.NoError(s.t, err)
	return responseBytes
}

func (s *oldServer) handleRequest(writer http.ResponseWriter, request *http.Request) {
	bodyBytes, err := io.ReadAll(request.Body)
	if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
		return
	}
	var bodyMap struct {
		Method string `json:"method"`
		Params any    `json:"params"`
	}
	err = json.Unmarshal(bodyBytes, &bodyMap)
	if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
		return
	}
	s.t.Logf("Mock Server Received: %+v", bodyMap)

	if bodyMap.Method == "handshake" {
		innerKeyRand, _, _ := s.getKeyDataFromCookie(writer, request)
		s.doHandshake(writer, bodyMap.Params, innerKeyRand)

	} else if bodyMap.Method == "securePassthrough" {
		var params struct {
			Request string `mapstructure:"request"`
		}
		err = mapstructure.Decode(bodyMap.Params, &params)
		if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
			return
		}
		_, aesCipher, iv := s.getKeyDataFromCookie(writer, request)
		clearText, err := s.decrypt(params.Request, aesCipher, iv)
		if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
			return
		}
		var innerBodyMap struct {
			Method string `json:"method"`
			Params any    `json:"params"`
		}
		err = json.Unmarshal(clearText, &innerBodyMap)
		if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
			return
		}
		s.t.Logf("Clear Text: %s", string(clearText))

		var response []byte
		if innerBodyMap.Method == "login_device" {
			response = s.handleLoginRequest(innerBodyMap.Params)
		} else if s.handler != nil {
			if assert.True(s.t, request.URL.Query().Has("token")) &&
				assert.Equal(s.t, request.URL.Query().Get("token"), "abc123") {
				response, err = (s.handler)(s.t, innerBodyMap.Method, innerBodyMap.Params)
				require.NoError(s.t, err)
			} else {
				response = s.failureForCode(9999)
			}
		} else {
			s.t.Errorf("Unexpected inner method: %s", innerBodyMap.Method)
			response = s.failureForCode(1002)
		}

		responseBytes, err := json.Marshal(struct {
			Result    any `json:"result"`
			ErrorCode int `json:"error_code"`
		}{
			ErrorCode: 0,
			Result: struct {
				Response string `json:"response"`
			}{
				Response: base64.StdEncoding.EncodeToString(
					s.encrypt(response, aesCipher, iv)),
			},
		})
		require.NoError(s.t, err)
		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write(responseBytes)
		require.NoError(s.t, err)
	} else {
		s.t.Errorf("Unexpected method: %s", bodyMap.Method)
		s.failWithCode(-1010, writer)
	}
}

func (s *oldServer) getKeyDataFromCookie(writer http.ResponseWriter, request *http.Request) ([]byte, cipher.Block, []byte) {
	// The real server doesn't do this (directly), but it's convenient for keeping the mock server stateless
	// In reality there are triggers inside the server that would invalidate sessions, at which point a response is
	// returned along the lines of 200 OK with a body {"error_code":9999}.  The mock server will just FailNow.
	sessionCookie, err := request.Cookie("TP_SESSIONID")
	var innerKeyRand []byte
	if errors.Is(err, http.ErrNoCookie) {
		innerKeyRand = s.generateNewKey()
		http.SetCookie(writer, &http.Cookie{
			Name:    "TP_SESSIONID",
			Value:   base64.StdEncoding.EncodeToString(innerKeyRand),
			Expires: time.Now().Add(1440 * time.Second),
		})
	} else {
		innerKeyRand, err = base64.StdEncoding.DecodeString(sessionCookie.Value)
		require.NoError(s.t, err)
		require.Len(s.t, innerKeyRand, 32)
	}
	aesCipher, err := aes.NewCipher(innerKeyRand[0:16])
	require.NoError(s.t, err)
	iv := innerKeyRand[16:32]
	require.Len(s.t, iv, aesCipher.BlockSize())
	return innerKeyRand, aesCipher, iv
}

func (s *oldServer) doHandshake(writer http.ResponseWriter, Params any, innerKeyRand []byte) {
	var params struct {
		Key string `mapstructure:"key"`
	}
	err := mapstructure.Decode(Params, &params)
	if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
		return
	}
	clientKey := s.readClientPublicKey(params.Key, writer)
	if clientKey == nil {
		return
	}
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, clientKey, innerKeyRand)
	if s.assertNoErrorOrFailWithCode(err, writer, -1010) {
		return
	}
	responseBytes, err := json.Marshal(struct {
		ErrorCode int `json:"error_code"`
		Result    any `json:"result"`
	}{
		ErrorCode: 0,
		Result: struct {
			Key string `json:"key"`
		}{
			Key: base64.StdEncoding.EncodeToString(cipherText),
		},
	})
	require.NoError(s.t, err)
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(responseBytes)
	require.NoError(s.t, err)
}

func (s *oldServer) readClientPublicKey(key string, writer http.ResponseWriter) *rsa.PublicKey {
	block, _ := pem.Decode([]byte(key))
	if !assert.Equal(s.t, "PUBLIC KEY", block.Type) {
		s.t.Errorf("Expected PEM to be type PUBLIC KEY")
		s.failWithCode(-1010, writer)
		return nil
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if !assert.NoError(s.t, err) {
		s.failWithCode(-1010, writer)
		return nil
	}
	return publicKey.(*rsa.PublicKey)
}

func (s *oldServer) generateNewKey() []byte {
	innerKeyRand := make([]byte, 32)
	bytesGenerated, err := rand.Read(innerKeyRand)
	require.NoError(s.t, err)
	require.Equal(s.t, 32, bytesGenerated)
	return innerKeyRand
}

func (s *oldServer) decrypt(base64CipherText string, aesCipher cipher.Block, iv []byte) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(base64CipherText)
	require.NoError(s.t, err)
	clearText := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(aesCipher, iv).CryptBlocks(clearText, cipherText)
	return pkcs7.Unpad(clearText, len(iv))
}

func (s *oldServer) encrypt(clearText []byte, aesCipher cipher.Block, iv []byte) []byte {
	padded, err := pkcs7.Pad(clearText, len(iv))
	require.NoError(s.t, err)
	cipherText := make([]byte, len(padded))
	cipher.NewCBCEncrypter(aesCipher, iv).CryptBlocks(cipherText, padded)
	return cipherText
}

func (s *oldServer) handleLoginRequest(Params any) []byte {
	var params struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	}
	err := mapstructure.Decode(Params, &params)
	if !assert.NoError(s.t, err) {
		return s.failureForCode(-1501)
	}
	s.t.Logf("Username: %s, Password: %s", params.Username, params.Password)

	hashedUsername, err := base64.StdEncoding.DecodeString(params.Username)
	assert.NoError(s.t, err)
	if !assert.NoError(s.t, err) {
		return s.failureForCode(-1501)
	}
	clearPassword, err := base64.StdEncoding.DecodeString(params.Password)
	if !assert.NoError(s.t, err) {
		return s.failureForCode(-1501)
	}
	hashedExpectedUsername := sha1.Sum([]byte(s.username))
	if !(hex.EncodeToString(hashedExpectedUsername[:]) == string(hashedUsername)) || !(s.password == string(clearPassword)) {
		return s.failureForCode(-1501)
	}

	response, err := json.Marshal(struct {
		Result    any `json:"result"`
		ErrorCode int `json:"error_code"`
	}{
		ErrorCode: 0,
		Result: struct {
			Token string `json:"token"`
		}{
			Token: "abc123",
		},
	})
	require.NoError(s.t, err)
	return response
}
