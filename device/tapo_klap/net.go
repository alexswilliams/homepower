package tapo_klap

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

type requestBodyMethodParamsTime struct {
	Method          string `json:"method"`
	Params          any    `json:"params,omitempty"`
	RequestTimeMils int64  `json:"requestTimeMils"`
}

type deviceAddresses struct {
	ip      string // x.x.x.x
	baseUrl string // http://x.x.x.x:80
	url     *url.URL
}

type deviceConnection struct {
	email     string // Hashed email of the account that originally set up the device
	password  string // Isn't it weird how the email is hashed and the password isn't
	addresses deviceAddresses
	client    *http.Client // A long-lived HTTP client that also retains the HTTP session state (e.g. cookies)

	localSeed  []byte
	remoteSeed []byte
	authHash   []byte
	encryption *encryptionContext
}

//goland:noinspection HttpUrlsUsage
func newDeviceConnection(email, password, deviceIp string, port uint16) (*deviceConnection, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("could not create new cookie jar whilst initialising %s: %w", deviceIp, err)
	}
	tr := &http.Transport{
		DisableKeepAlives:      false,
		DisableCompression:     false,
		MaxIdleConnsPerHost:    1,
		MaxConnsPerHost:        1,
		IdleConnTimeout:        5 * time.Minute,
		ResponseHeaderTimeout:  5 * time.Second,
		MaxResponseHeaderBytes: 4096,
		ForceAttemptHTTP2:      false,
	}
	baseUrl := "http://" + deviceIp + ":" + strconv.FormatUint(uint64(port), 10)
	parsedUrl, err := url.Parse(baseUrl + "/app/request")
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as a URL object: %w", baseUrl, err)
	}
	return &deviceConnection{
		email:    email,
		password: password,
		addresses: deviceAddresses{
			ip:      deviceIp,
			baseUrl: baseUrl,
			url:     parsedUrl,
		},
		client: &http.Client{
			Transport: tr,
			Jar:       jar,
			Timeout:   10 * time.Second,
		},
	}, nil
}

func (dc *deviceConnection) applyHeadersTo(request *http.Request) {
	request.Header.Set("Referer", dc.addresses.baseUrl)
	request.Header.Set("requestByApp", "true")
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("Host", dc.addresses.ip)
	request.Header.Set("User-Agent", "okhttp/3.12.13")
}

func (dc *deviceConnection) doKeyExchange() error {
	dc.localSeed = make([]byte, 16)
	if _, err := rand.Read(dc.localSeed); err != nil {
		return err
	}
	request1, err := http.NewRequest(http.MethodPost, dc.addresses.baseUrl+"/app/handshake1", bytes.NewReader(dc.localSeed))
	if err != nil {
		return err
	}
	dc.applyHeadersTo(request1)

	handshakeResponse, err := dc.exchangeExpect200(request1)
	if err != nil {
		return err
	}
	if len(handshakeResponse) != 48 {
		return fmt.Errorf("expected handshake 1 response to be 48 byte but got %d", len(handshakeResponse))
	}
	dc.remoteSeed = handshakeResponse[0:16]
	userHash := sha1.Sum([]byte(dc.email))
	passHash := sha1.Sum([]byte(dc.password))
	authHash := sha256.Sum256(append(userHash[:], passHash[:]...))
	dc.authHash = authHash[:]
	localRemoteAuthBuffer := append(append(bytes.Clone(dc.localSeed), dc.remoteSeed...), dc.authHash...)
	expectedHash := sha256.Sum256(localRemoteAuthBuffer)
	if !bytes.Equal(expectedHash[:], handshakeResponse[16:]) {
		return errors.New("handshake 1 response hash did not match expected credentials")
	}
	time.Sleep(250 * time.Millisecond)

	payload := sha256.Sum256(append(append(bytes.Clone(dc.remoteSeed), dc.localSeed...), dc.authHash...))
	request2, err := http.NewRequest(http.MethodPost, dc.addresses.baseUrl+"/app/handshake2", bytes.NewReader(payload[:]))
	if err != nil {
		return err
	}
	_, err = dc.exchangeExpect200(request2)
	if err != nil {
		return err
	}

	dc.encryption, err = setupEncryption(localRemoteAuthBuffer)
	if err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Printf("KLAP Handshake Complete for %s\n", dc.addresses.ip)
	return nil
}

func (dc *deviceConnection) hasExchangedKeys() bool {
	return dc.hasValidSessionCookie() && dc.localSeed != nil && len(dc.localSeed) > 0
}

func (dc *deviceConnection) exchangeExpect200(request *http.Request) ([]byte, error) {
	response, err := dc.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	if response.StatusCode != 200 {
		return nil, errors.New("expected status code 200, got " + strconv.Itoa(response.StatusCode))
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return responseBody, nil
}

func (dc *deviceConnection) hasValidSessionCookie() bool {
	for _, cookie := range dc.client.Jar.Cookies(dc.addresses.url) {
		if cookie.Name == "TP_SESSIONID" {
			if cookie.Expires.Year() < 1601 { // has no expiry
				return true
			}
			return cookie.Expires.After(time.Now())
		}
	}
	return false
}

func (dc *deviceConnection) forgetKeysAndSession() {
	dc.client.CloseIdleConnections()
	dc.client.Jar.SetCookies(dc.addresses.url, []*http.Cookie{{
		Name:   "TP_SESSIONID",
		MaxAge: -1,
	}})
	dc.localSeed = nil
	dc.remoteSeed = nil
	dc.authHash = nil
}

func (dc *deviceConnection) GetDeviceInfo() (map[string]interface{}, error) {
	return dc.makeApiCall("{\"method\": \"get_device_info\"}")
}
func (dc *deviceConnection) GetEnergyUsage() (map[string]interface{}, error) {
	return dc.makeApiCall("{\"method\": \"get_energy_usage\"}")
}
func (dc *deviceConnection) makeApiCall(payload string) (map[string]interface{}, error) {
	if !dc.hasExchangedKeys() {
		log.Println("Not logged in, will log in before making api request")
		if err := dc.doKeyExchange(); err != nil {
			dc.forgetKeysAndSession()
			return nil, fmt.Errorf("could not log in before making API call: %w", err)
		}
	}

	encryptedPayload := dc.encryption.Encrypt([]byte(payload))
	request, err := http.NewRequest(
		http.MethodPost,
		dc.addresses.baseUrl+"/app/request?seq="+strconv.Itoa(int(dc.encryption.sequenceNumber)),
		bytes.NewReader(encryptedPayload))
	dc.applyHeadersTo(request)
	if err != nil {
		return nil, err
	}
	response, err := dc.exchangeExpect200(request)
	if err != nil {
		dc.forgetKeysAndSession()
		return nil, err
	}
	clearText, err := dc.encryption.Decrypt(response)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("clearText:\n %v\n %s\n\n", clearText, string(clearText))
	return dc.unmarshalApiResponse(clearText)
}

func (dc *deviceConnection) unmarshalApiResponse(response []byte) (map[string]interface{}, error) {
	var responseData map[string]interface{}
	if err := json.Unmarshal(response, &responseData); err != nil {
		return nil, err
	}
	if errorCode, present := responseData["error_code"]; present && int(errorCode.(float64)) != 0 {
		return nil, errors.New("non-zero error code returned: " + strconv.Itoa(int(errorCode.(float64))))
	}
	responseResult := responseData["result"].(map[string]interface{})
	return responseResult, nil
}
