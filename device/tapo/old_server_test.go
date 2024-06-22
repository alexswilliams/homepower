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
	testServer := httptest.NewServer(mux)
	port, err := strconv.Atoi(strings.Split(testServer.URL, ":")[2])
	assert.NoError(t, err)
	return testServer, uint16(port)
}

func (s *oldServer) handleRequest(writer http.ResponseWriter, request *http.Request) {
	innerKeyRand, aesCipher, iv := s.getKeyDataFromCookie(writer, request)

	bodyBytes, err := io.ReadAll(request.Body)
	assert.NoError(s.t, err)
	var bodyMap struct {
		Method string `json:"method"`
		Params any    `json:"params"`
	}
	err = json.Unmarshal(bodyBytes, &bodyMap)
	assert.NoError(s.t, err)
	s.t.Logf("Mock Server Received: %+v", bodyMap)

	if bodyMap.Method == "handshake" {
		s.doHandshake(writer, bodyMap.Params, innerKeyRand)

	} else if bodyMap.Method == "securePassthrough" {
		var params struct {
			Request string `mapstructure:"request"`
		}
		err = mapstructure.Decode(bodyMap.Params, &params)
		assert.NoError(s.t, err)
		clearText, err := s.decrypt(params.Request, aesCipher, iv)
		assert.NoError(s.t, err)
		var innerBodyMap struct {
			Method string `json:"method"`
			Params any    `json:"params"`
		}
		err = json.Unmarshal(clearText, &innerBodyMap)
		assert.NoError(s.t, err)
		s.t.Logf("Clear Text: %s", string(clearText))

		var response []byte
		if innerBodyMap.Method == "login_device" {
			response = s.handleLoginRequest(innerBodyMap.Params)
		} else if s.handler != nil {
			response, err = (s.handler)(s.t, innerBodyMap.Method, innerBodyMap.Params)
			assert.NoError(s.t, err)
		} else {
			s.t.Fatalf("Unexpected inner method: %s", innerBodyMap.Method)
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
		assert.NoError(s.t, err)
		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write(responseBytes)
		assert.NoError(s.t, err)
	} else {
		s.t.Fatalf("Unexpected method: %s", bodyMap.Method)
	}
}

func (s *oldServer) getKeyDataFromCookie(writer http.ResponseWriter, request *http.Request) ([]byte, cipher.Block, []byte) {
	// The real server doesn't do this (directly), but it's convenient for keeping the mock server stateless
	sessionCookie, err := request.Cookie("TP_SESSIONID")
	var innerKeyRand []byte
	if errors.Is(err, http.ErrNoCookie) {
		innerKeyRand = s.generateNewKey()
		http.SetCookie(writer, &http.Cookie{
			Name:    "TP_SESSIONID",
			Value:   base64.StdEncoding.EncodeToString(innerKeyRand),
			Expires: time.Now().Add(time.Hour),
		})
	} else {
		innerKeyRand, err = base64.StdEncoding.DecodeString(sessionCookie.Value)
		assert.NoError(s.t, err)
		assert.Len(s.t, innerKeyRand, 32)
	}
	aesCipher, err := aes.NewCipher(innerKeyRand[0:16])
	assert.NoError(s.t, err)
	iv := innerKeyRand[16:32]
	assert.Len(s.t, iv, aesCipher.BlockSize())
	return innerKeyRand, aesCipher, iv
}

func (s *oldServer) doHandshake(writer http.ResponseWriter, Params any, innerKeyRand []byte) {
	var params struct {
		Key string `mapstructure:"key"`
	}
	err := mapstructure.Decode(Params, &params)
	assert.NoError(s.t, err)

	clientKey := readClientPublicKey(params.Key, s.t, err)
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, clientKey, innerKeyRand)
	assert.NoError(s.t, err)
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
	assert.NoError(s.t, err)
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(responseBytes)
	assert.NoError(s.t, err)
}

func readClientPublicKey(key string, t *testing.T, err error) *rsa.PublicKey {
	block, _ := pem.Decode([]byte(key))
	assert.Equal(t, "PUBLIC KEY", block.Type)
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	assert.NoError(t, err)
	return publicKey.(*rsa.PublicKey)
}

func (s *oldServer) generateNewKey() []byte {
	innerKeyRand := make([]byte, 32)
	bytesGenerated, err := rand.Read(innerKeyRand)
	assert.NoError(s.t, err)
	assert.Equal(s.t, 32, bytesGenerated)
	return innerKeyRand
}

func (s *oldServer) decrypt(base64CipherText string, aesCipher cipher.Block, iv []byte) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(base64CipherText)
	assert.NoError(s.t, err)
	clearText := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(aesCipher, iv).CryptBlocks(clearText, cipherText)
	return pkcs7.Unpad(clearText, len(iv))
}

func (s *oldServer) encrypt(clearText []byte, aesCipher cipher.Block, iv []byte) []byte {
	padded, err := pkcs7.Pad(clearText, len(iv))
	assert.NoError(s.t, err)
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
	assert.NoError(s.t, err)
	s.t.Logf("Username: %s, Password: %s", params.Username, params.Password)

	hashedUsername, err := base64.StdEncoding.DecodeString(params.Username)
	assert.NoError(s.t, err)
	hashedExpectedUsername := sha1.Sum([]byte(s.username))
	clearPassword, err := base64.StdEncoding.DecodeString(params.Password)
	assert.NoError(s.t, err)
	assert.Equal(s.t, s.password, string(clearPassword))
	errorCode := 0
	token := "abc123"
	if !(hex.EncodeToString(hashedExpectedUsername[:]) == string(hashedUsername)) ||
		!(s.password == string(clearPassword)) {
		errorCode = 1003
		token = ""
	}

	response, err := json.Marshal(struct {
		Result    any `json:"result"`
		ErrorCode int `json:"error_code"`
	}{
		ErrorCode: errorCode,
		Result: struct {
			Token string `json:"token"`
		}{
			Token: token,
		},
	})
	assert.NoError(s.t, err)
	return response
}
