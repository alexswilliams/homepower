package tapo

import (
	"bytes"
	"crypto/cipher"
	"encoding/base64"
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

type oldDeviceAddresses struct {
	ip          string // x.x.x.x
	baseUrl     string // http://x.x.x.x:80
	url         *url.URL
	appUrl      string // http://x.x.x.x:80/app
	appTokenUrl string // Empty string when not logged in, otherwise http://x.x.x.x:80/app?token=xxxx
}

type oldDeviceConnection struct {
	hashedEmail string // Hashed email of the account that originally set up the device
	password    string // Isn't it weird how the email is hashed and the password isn't
	addresses   oldDeviceAddresses
	client      *http.Client // A long-lived HTTP client that also retains the HTTP session state (e.g. cookies)

	cbcIv     []byte        // The shared CBC init vector between this app and the device, nil until after key-exchange
	cbcCipher *cipher.Block // The shared cipher info between this app and the device, nil until after key-exchange
}

//goland:noinspection HttpUrlsUsage
func newTapoOldDeviceConnection(email, password, deviceIp string, port uint16) (*oldDeviceConnection, error) {
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
	parsedUrl, err := url.Parse(baseUrl + "/app")
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as a URL object: %w", baseUrl, err)
	}
	return &oldDeviceConnection{
		hashedEmail: hashUsername(email),
		password:    password,
		addresses: oldDeviceAddresses{
			ip:          deviceIp,
			baseUrl:     baseUrl,
			url:         parsedUrl,
			appUrl:      baseUrl + "/app",
			appTokenUrl: "",
		},
		client: &http.Client{
			Transport: tr,
			Jar:       jar,
			Timeout:   10 * time.Second,
		},
	}, nil
}

func (dc *oldDeviceConnection) devicePostUrl() string {
	if dc.addresses.appTokenUrl == "" {
		return dc.addresses.appUrl
	} else {
		return dc.addresses.appTokenUrl
	}
}

func (dc *oldDeviceConnection) applyHeadersTo(request *http.Request) {
	request.Header.Set("Referer", dc.addresses.baseUrl)
	request.Header.Set("requestByApp", "true")
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("Host", dc.addresses.ip)
	request.Header.Set("User-Agent", "okhttp/3.12.13")
}

func (dc *oldDeviceConnection) exchange(body []byte) (map[string]interface{}, error) {
	request, err := http.NewRequest(http.MethodPost, dc.devicePostUrl(), bytes.NewReader(body))
	dc.applyHeadersTo(request)

	response, err := dc.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, errors.New("Expected status code 200, got " + strconv.Itoa(response.StatusCode))
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	responseBody, err := io.ReadAll(response.Body)
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		return nil, fmt.Errorf("could not unmarshal response as JSON: %w", err)
	}

	errorCode := int(responseData["error_code"].(float64))
	if errorCode != 0 {
		return nil, errors.New("Expected error code to be 0, but got error code " + strconv.Itoa(errorCode))
	}
	result := responseData["result"].(map[string]interface{})
	return result, nil
}

func (dc *oldDeviceConnection) marshalPassthroughPayload(method string, params any) ([]byte, error) {
	clearTextPayload, err := json.Marshal(requestBodyMethodParamsTime{
		Method:          method,
		RequestTimeMils: time.Now().UnixMilli(),
		Params:          params,
	})
	if err != nil {
		return nil, fmt.Errorf("could not marshal passthrough payload: %w", err)
	}
	return json.Marshal(struct {
		Method string `json:"method"`
		Params any    `json:"params,omitempty"`
	}{
		Method: "securePassthrough",
		Params: struct {
			Request string `json:"request"`
		}{
			Request: encryptWithPkcs7Padding(dc.newEncrypter(), clearTextPayload),
		},
	})
}

func (dc *oldDeviceConnection) unmarshalPassthroughResponse(passthroughResult map[string]interface{}) (map[string]interface{}, error) {
	decryptedResponse, err := decryptAndRemovePadding(dc.newDecrypter(), passthroughResult["response"].(string))
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal passthrough response: %w", err)
	}
	var responseData map[string]interface{}
	if err := json.Unmarshal(decryptedResponse, &responseData); err != nil {
		return nil, err
	}
	if errorCode, present := responseData["error_code"]; present && int(errorCode.(float64)) != 0 {
		return nil, errors.New("non-zero error code returned within encrypted payload: " + strconv.Itoa(int(errorCode.(float64))))
	}

	responseResult := responseData["result"].(map[string]interface{})
	return responseResult, nil
}

func (dc *oldDeviceConnection) doKeyExchange() error {
	dc.logout()
	privateKey, err := NewRsaKeypair()
	if err != nil {
		return fmt.Errorf("could not generate new RSA keypair: %w", err)
	}
	publicKeyPem, err := textualPublicKey(privateKey)
	if err != nil {
		return fmt.Errorf("could not extract textual public key from priate key: %w", err)
	}

	type handshakeParams struct {
		Key string `json:"key"`
	}
	handshakeBody, err := json.Marshal(requestBodyMethodParamsTime{
		Method:          "handshake",
		RequestTimeMils: 0,
		Params: handshakeParams{
			Key: publicKeyPem,
		},
	})
	if err != nil {
		return fmt.Errorf("could not marshal key exchange request body: %w", err)
	}
	result, err := dc.exchange(handshakeBody)
	if err != nil {
		return fmt.Errorf("could not perform key exchange POST request: %w", err)
	}

	remoteKey := result["key"].(string)
	block, iv, err := cbcCipherAndIvFromHandshakeResponse(remoteKey, privateKey)
	if err != nil {
		return fmt.Errorf("could not determine CBC parameters from key exchange response: %w", err)
	}
	dc.cbcIv = iv
	dc.cbcCipher = block
	return nil
}
func (dc *oldDeviceConnection) hasExchangedKeys() bool {
	return dc.hasValidSessionCookie() && dc.cbcCipher != nil && dc.cbcIv != nil
}

func (dc *oldDeviceConnection) hasValidSessionCookie() bool {
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

func (dc *oldDeviceConnection) doLogin() error {
	if !dc.hasExchangedKeys() {
		if err := dc.doKeyExchange(); err != nil {
			return fmt.Errorf("could not do key exchange before logging in: %w", err)
		}
	}
	dc.logout()

	passthroughBody, err := dc.marshalPassthroughPayload("login_device", struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: base64.StdEncoding.EncodeToString([]byte(dc.hashedEmail)),
		Password: base64.StdEncoding.EncodeToString([]byte(dc.password)),
	})
	if err != nil {
		return fmt.Errorf("could not marshal login_device payload: %w", err)
	}

	passthroughResult, err := dc.exchange(passthroughBody)
	if err != nil {
		return fmt.Errorf("could not perform login POST request: %w", err)
	}

	responseResult, err := dc.unmarshalPassthroughResponse(passthroughResult)
	if err != nil {
		return fmt.Errorf("could not unmarshal login_device response: %w", err)
	}
	token := responseResult["token"].(string)
	dc.addresses.appTokenUrl = dc.addresses.appUrl + "?token=" + token
	return nil
}
func (dc *oldDeviceConnection) isLoggedIn() bool {
	return dc.hasExchangedKeys() && dc.addresses.appTokenUrl != "" && dc.client != nil
}
func (dc *oldDeviceConnection) logout() {
	dc.addresses.appTokenUrl = ""
}

func (dc *oldDeviceConnection) forgetKeysAndSession() {
	dc.logout()
	dc.client.CloseIdleConnections()
	dc.client.Jar.SetCookies(dc.addresses.url, []*http.Cookie{{
		Name:   "TP_SESSIONID",
		MaxAge: -1,
	}})
	dc.cbcCipher = nil
	dc.cbcIv = nil
}

func (dc *oldDeviceConnection) GetDeviceInfo() (map[string]interface{}, error) {
	return dc.makeApiCall("get_device_info")
}
func (dc *oldDeviceConnection) GetEnergyUsage() (map[string]interface{}, error) {
	return dc.makeApiCall("get_energy_usage")
}
func (dc *oldDeviceConnection) makeApiCall(method string) (map[string]interface{}, error) {
	if !dc.isLoggedIn() {
		log.Println("Not logged in, will log in before making api request")
		if err := dc.doLogin(); err != nil {
			return nil, fmt.Errorf("could not log in before making API call: %w", err)
		}
	}

	passthroughBody, err := dc.marshalPassthroughPayload(method, nil)
	if err != nil {
		return nil, fmt.Errorf("could not marshal passthrough payload for %s: %w", method, err)
	}
	passthroughResult, err := dc.exchange(passthroughBody)
	if err != nil {
		return nil, fmt.Errorf("could not perform %s POST request: %w", method, err)
	}
	responseResult, err := dc.unmarshalPassthroughResponse(passthroughResult)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal passthrough respone for %s: %w", method, err)
	}
	return responseResult, nil
}
