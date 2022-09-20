package tapo

import (
	"crypto/cipher"
	"crypto/rsa"
	"homepower/types"
	"net/http"
	"net/http/cookiejar"
	"time"
)

type DeviceConnection struct {
	Device       *types.DeviceConfig
	privateKey   *rsa.PrivateKey
	publicKeyPem string
	cbcIv        []byte
	cbcCipher    *cipher.Block
	jar          *cookiejar.Jar
	client       *http.Client
	token        *string
}

func (dc *DeviceConnection) ensureHttpClient() error {
	if dc.client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return err
		}
		dc.jar = jar
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
		dc.client = &http.Client{
			Transport: tr,
			Jar:       dc.jar,
			Timeout:   10 * time.Second,
		}
	}
	return nil
}

func (dc *DeviceConnection) initNewRsaKeypair() error {
	key, err := NewRsaKeypair()
	if err != nil {
		return err
	}
	dc.privateKey = key
	pubKeyString, err := textualPublicKey(dc.privateKey)
	if err != nil {
		dc.privateKey = nil
		return err
	}
	dc.publicKeyPem = pubKeyString
	return nil
}

func (dc *DeviceConnection) logout() {
	dc.token = nil
}

func (dc *DeviceConnection) isLoggedIn() bool {
	if dc.token == nil || dc.jar == nil || dc.cbcIv == nil || dc.cbcCipher == nil || dc.client == nil {
		return false
	}
	// TODO: check for cookie expiry
	return true
}

func (dc *DeviceConnection) newEncrypter() cipher.BlockMode {
	return cipher.NewCBCEncrypter(*dc.cbcCipher, dc.cbcIv)
}
func (dc *DeviceConnection) newDecrypter() cipher.BlockMode {
	return cipher.NewCBCDecrypter(*dc.cbcCipher, dc.cbcIv)
}
