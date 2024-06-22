package tapo

import (
	"bytes"
	"crypto/cipher"
	"crypto/rsa"
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
	Method          string      `json:"method"`
	Params          interface{} `json:"params,omitempty"`
	RequestTimeMils int64       `json:"requestTimeMils"`
}
type requestBodyMethodParams struct {
	Method string      `json:"method"`
	Params interface{} `json:"params,omitempty"`
}
type passthroughParams struct {
	Request string `json:"request"`
}

type deviceAddresses struct {
	ip          string // x.x.x.x
	baseUrl     string // http://x.x.x.x:80
	url         *url.URL
	appUrl      string // http://x.x.x.x:80/app
	appTokenUrl string // Empty string when not logged in, otherwise http://x.x.x.x:80/app?token=xxxx
}

type deviceConnection struct {
	hashedEmail string // Hashed email of the account that originally set up the device
	password    string // Isn't it weird how the email is hashed and the password isn't
	addresses   deviceAddresses
	client      *http.Client // A long-lived HTTP client that also retains the HTTP session state (e.g. cookies)

	privateKey   *rsa.PrivateKey // This app's private key, nil until created during key-exchange
	publicKeyPem string          // This app's public component, the empty string until created during key-exchange
	cbcIv        []byte          // The shared CBC init vector between this app and the device, nil until after key-exchange
	cbcCipher    *cipher.Block   // The shared cipher info between this app and the device, nil until after key-exchange
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
	parsedUrl, err := url.Parse(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as a URL object: %w", baseUrl, err)
	}
	return &deviceConnection{
		hashedEmail: hashUsername(email),
		password:    password,
		addresses: deviceAddresses{
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

func (dc *deviceConnection) initNewRsaKeypair() error {
	key, err := NewRsaKeypair()
	if err != nil {
		return fmt.Errorf("could not generate new RSA keypair: %w", err)
	}
	dc.privateKey = key
	pubKeyString, err := textualPublicKey(dc.privateKey)
	if err != nil {
		dc.privateKey = nil
		return fmt.Errorf("could not extract textual public key from priate key: %w", err)
	}
	dc.publicKeyPem = pubKeyString
	return nil
}

func (dc *deviceConnection) devicePostUrl() string {
	if dc.addresses.appTokenUrl == "" {
		return dc.addresses.appUrl
	} else {
		return dc.addresses.appTokenUrl
	}
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

func (dc *deviceConnection) exchange(body []byte) (map[string]interface{}, error) {
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

func (dc *deviceConnection) marshalPassthroughPayload(method string, params any) ([]byte, error) {
	clearTextPayload, err := json.Marshal(requestBodyMethodParamsTime{
		Method:          method,
		RequestTimeMils: time.Now().UnixMilli(),
		Params:          params,
	})
	if err != nil {
		return nil, fmt.Errorf("could not marshal passthrough payload: %w", err)
	}
	return json.Marshal(requestBodyMethodParams{
		Method: "securePassthrough",
		Params: passthroughParams{
			Request: encryptWithPkcs7Padding(dc.newEncrypter(), clearTextPayload),
		},
	})
}

func (dc *deviceConnection) unmarshalPassthroughResponse(passthroughResult map[string]interface{}) (map[string]interface{}, error) {
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

func (dc *deviceConnection) doKeyExchange() error {
	dc.logout()
	if err := dc.initNewRsaKeypair(); err != nil {
		return fmt.Errorf("could not initialise new RSA keypair: %w", err)
	}

	type handshakeParams struct {
		Key string `json:"key"`
	}
	handshakeBody, err := json.Marshal(requestBodyMethodParamsTime{
		Method:          "handshake",
		RequestTimeMils: 0,
		Params: handshakeParams{
			Key: dc.publicKeyPem,
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
	block, iv, err := cbcCipherAndIvFromHandshakeResponse(remoteKey, dc.privateKey)
	if err != nil {
		return fmt.Errorf("could not determine CBC parameters from key exchange response: %w", err)
	}
	dc.cbcIv = iv
	dc.cbcCipher = block
	return nil
}
func (dc *deviceConnection) hasExchangedKeys() bool {
	return dc.hasValidSessionCookie() && dc.privateKey != nil && dc.cbcCipher != nil && dc.cbcIv != nil
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

func (dc *deviceConnection) doLogin() error {
	if !dc.hasExchangedKeys() {
		if err := dc.doKeyExchange(); err != nil {
			return fmt.Errorf("could not do key exchange before logging in: %w", err)
		}
	}
	dc.logout()

	type loginParams struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	passthroughBody, err := dc.marshalPassthroughPayload("login_device", loginParams{
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
func (dc *deviceConnection) isLoggedIn() bool {
	return dc.hasExchangedKeys() && dc.addresses.appTokenUrl != "" && dc.client != nil
}
func (dc *deviceConnection) logout() {
	dc.addresses.appTokenUrl = ""
}

func (dc *deviceConnection) forgetKeysAndSession() {
	dc.logout()
	dc.client.CloseIdleConnections()
	dc.client.Jar.SetCookies(dc.addresses.url, []*http.Cookie{{
		Name:   "TP_SESSIONID",
		MaxAge: -1,
	}})
	dc.cbcCipher = nil
	dc.cbcIv = nil
	dc.privateKey = nil
	dc.publicKeyPem = ""
}

func (dc *deviceConnection) makeApiCall(method string, params any) (map[string]interface{}, error) {
	if !dc.isLoggedIn() {
		log.Println("Not logged in, will log in before making api request")
		if err := dc.doLogin(); err != nil {
			return nil, fmt.Errorf("could not log in before making API call: %w", err)
		}
	}

	passthroughBody, err := dc.marshalPassthroughPayload(method, params)
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
