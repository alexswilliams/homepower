package tapo

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
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
	Params interface{} `json:"params"`
}
type passthroughParams struct {
	Request string `json:"request"`
}

//goland:noinspection HttpUrlsUsage
func (dc *DeviceConnection) devicePostUrl() string {
	if dc.token == nil {
		return "http://" + dc.Device.Ip + ":80/app"
	} else {
		return "http://" + dc.Device.Ip + ":80/app?token=" + *dc.token
	}
}

//goland:noinspection HttpUrlsUsage
func (dc *DeviceConnection) applyTapoHeadersTo(request *http.Request) {
	request.Header.Set("Referer", "http://"+dc.Device.Ip+":80")
	request.Header.Set("requestByApp", "true")
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Connection", "Keep-Alive")
	request.Header.Set("Host", dc.Device.Ip)
	request.Header.Set("User-Agent", "okhttp/3.12.13")
}

func (dc *DeviceConnection) exchange(body []byte) (map[string]interface{}, error) {
	request, err := http.NewRequest(http.MethodPost, dc.devicePostUrl(), bytes.NewReader(body))
	dc.applyTapoHeadersTo(request)

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
		return nil, err
	}

	errorCode := int(responseData["error_code"].(float64))
	if errorCode != 0 {
		return nil, errors.New("Expected error code to be 0, but got error code " + strconv.Itoa(errorCode))
	}

	result := responseData["result"].(map[string]interface{})
	return result, nil
}

func (dc *DeviceConnection) marshalPassthroughPayload(method string, v any) ([]byte, error) {
	clearTextPayload, err := json.Marshal(requestBodyMethodParamsTime{
		Method:          method,
		RequestTimeMils: time.Now().UnixMilli(),
		Params:          v,
	})
	if err != nil {
		return nil, err
	}
	return json.Marshal(requestBodyMethodParams{
		Method: "securePassthrough",
		Params: passthroughParams{
			Request: encryptWithPkcs7Padding(dc.newEncrypter(), clearTextPayload),
		},
	})
}

func (dc *DeviceConnection) unmarshalPassthroughResponse(passthroughResult map[string]interface{}) (map[string]interface{}, error) {
	decryptedResponse, err := decryptFromBase64(dc.newDecrypter(), passthroughResult["response"].(string))
	if err != nil {
		return nil, err
	}
	var responseData map[string]interface{}
	if err := json.Unmarshal(decryptedResponse, &responseData); err != nil {
		return nil, err
	}
	if errorCode, present := responseData["error_code"]; present && int(errorCode.(float64)) != 0 {
		return nil, errors.New("non-zero error code returned within encrypted payload")
	}

	responseResult := responseData["result"].(map[string]interface{})
	return responseResult, nil
}

func (dc *DeviceConnection) DoKeyExchange() error {
	dc.logout()
	if err := dc.initNewRsaKeypair(); err != nil {
		return err
	}
	if err := dc.ensureHttpClient(); err != nil {
		return err
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
		return err
	}

	result, err := dc.exchange(handshakeBody)
	if err != nil {
		return err
	}

	remoteKey := result["key"].(string)
	block, iv, err := cbcCipherAndIvFromHandshakeResponse(remoteKey, dc.privateKey)
	if err != nil {
		return err
	}

	dc.cbcIv = iv
	dc.cbcCipher = block
	return nil
}

func (dc *DeviceConnection) DoLogin(email string, password string) error {
	if dc.privateKey == nil || dc.cbcCipher == nil || dc.cbcIv == nil {
		return errors.New("must have performed key exchange before logging in")
	}
	dc.logout()

	type loginParams struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	passthroughBody, err := dc.marshalPassthroughPayload("login_device", loginParams{
		Username: base64.StdEncoding.EncodeToString([]byte(hashUsername(email))),
		Password: base64.StdEncoding.EncodeToString([]byte(password)),
	})
	if err != nil {
		return err
	}

	passthroughResult, err := dc.exchange(passthroughBody)
	if err != nil {
		return err
	}

	responseResult, err := dc.unmarshalPassthroughResponse(passthroughResult)
	if err != nil {
		return err
	}
	token := responseResult["token"].(string)
	dc.token = &token
	return nil
}

func (dc *DeviceConnection) GetDeviceInfo() error {
	if !dc.isLoggedIn() {
		return errors.New("must be logged in before getting energy info")
	}

	passthroughBody, err := dc.marshalPassthroughPayload("get_device_info", nil)
	if err != nil {
		return err
	}
	passthroughResult, err := dc.exchange(passthroughBody)
	if err != nil {
		return err
	}
	responseResult, err := dc.unmarshalPassthroughResponse(passthroughResult)
	if err != nil {
		return err
	}
	log.Println(responseResult) // TODO: just for debugging
	//map[
	// avatar:fan
	// default_states:map[
	//  state:map[on:true]
	//  type:custom
	// ]
	// device_id:8022108E94DD9F0F5CD7CAA59D0F71901FE5D070
	// device_on:true
	// fw_id:00000000000000000000000000000000
	// fw_ver:1.0.7 Build 210629 Rel.174901
	// has_set_location_info:true
	// hw_id:56DD079101D61D400A11C4A3D41C51DA
	// hw_ver:1.0
	// ip:192.168.1.67
	// lang:en_US
	// latitude:501234 // (degrees * 1000 - smudged to protect my location...)
	// longitude:-11234 // (degrees * 1000 - smudged as above)
	// mac:28-87-BA-C8-DF-77
	// model:P110
	// nickname:RnJpZGdlIEZyZWV6ZXIg // base64 for "Fridge Freezer " with the trailing space
	// oem_id:AE7B616A7168B34151ABBCF86C88DF34
	// on_time:2386 // not hours, would be too long; not second, because that's only 39 minutes and what?
	// overheated:false
	// region:Europe/London
	// rssi:-56
	// signal_level:2
	// specs:
	// ssid:QWxleElvVA== // base64 for "AlexIoT"
	// time_diff:0
	// type:SMART.TAPOPLUG
	//]
	return nil
}

func (dc *DeviceConnection) GetEnergyInfo() error {
	if !dc.isLoggedIn() {
		return errors.New("must be logged in before getting energy info")
	}

	passthroughBody, err := dc.marshalPassthroughPayload("get_energy_usage", nil)
	if err != nil {
		return err
	}
	passthroughResult, err := dc.exchange(passthroughBody)
	if err != nil {
		return err
	}
	responseResult, err := dc.unmarshalPassthroughResponse(passthroughResult)
	if err != nil {
		return err
	}
	log.Println(responseResult) // TODO: just for debugging
	//map[
	// current_power:2529
	// local_time:2022-09-20 03:05:19
	// month_energy:5203
	// month_runtime:17644
	// past1y:[0 0 0 0 0 0 0 0 0 0 0 5203]
	// past24h:[14 17 14 15 14 20 13 15 14 15 17 26 16 21 17 13 14 15 15 14 23 23 21 0]
	// past30d:[0 0 0 0 0 0 0 0 0 0 0 0 0 5 0 0 0 212 473 459 484 489 475 453 417 457 424 398 390 67]
	// past7d:[
	//  [15 26 17 23 12 27 17 13 16 26 14 14 17 28 15 15 19 18 29 17 16 13 20 26]
	//  [20 12 14 14 21 22 14 14 21 19 13 20 20 20 16 14 15 15 19 21 22 18 20 13]
	//  [23 18 21 26 14 14 23 22 13 17 22 21 22 19 12 25 13 17 15 25 23 18 14 20]
	//  [12 14 14 24 18 17 20 20 16 13 14 20 14 26 18 13 14 14 21 20 22 21 20 19]
	//  [13 20 20 12 19 13 19 13 24 17 13 18 13 15 25 18 14 14 17 17 18 16 13 17]
	//  [21 18 15 17 14 17 14 15 14 20 13 15 14 15 17 26 16 21 17 13 14 15 15 14]
	//  [23 23 21 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]]
	// today_energy:67
	// today_runtime:181
	//]

	return nil
}
