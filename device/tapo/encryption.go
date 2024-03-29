package tapo

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
)

func NewRsaKeypair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 1024)
}

func textualPublicKey(key *rsa.PrivateKey) (string, error) {
	marshalled, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", fmt.Errorf("could not marshal public key as PKIX: %w", err)
	}
	var outBytes bytes.Buffer
	if err := pem.Encode(&outBytes, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: marshalled,
	}); err != nil {
		return "", fmt.Errorf("could not PEM-encode marshalled public key: %w", err)
	}
	return string(outBytes.Bytes()), nil
}

func cbcCipherAndIvFromHandshakeResponse(base64Ciphertext string, decryptionKey *rsa.PrivateKey) (*cipher.Block, []byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decode ciphertext as base64: %w", err)
	}
	cleartextPayload, err := rsa.DecryptPKCS1v15(rand.Reader, decryptionKey, cipherText)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decrypt ciphertext: %w", err)
	}
	if len(cleartextPayload) != 32 {
		return nil, nil, errors.New("Expected payload to be 32 bytes, but payload was actually " + strconv.Itoa(len(cleartextPayload)) + " bytes")
	}
	block, err := aes.NewCipher(cleartextPayload[0:16])
	if err != nil {
		return nil, nil, fmt.Errorf("could not construct CBC cipher from decrypted payload: %w", err)
	}
	return &block, cleartextPayload[16:32], nil
}

func encryptWithPkcs7Padding(encrypter cipher.BlockMode, clearText []byte) string {
	var padded = pkcs7Pad(clearText, encrypter.BlockSize())
	var cipherText = make([]byte, len(padded))
	encrypter.CryptBlocks(cipherText, padded)
	return base64.StdEncoding.EncodeToString(cipherText)
}

func decryptFromBase64(decrypter cipher.BlockMode, base64Ciphertext string) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("could not decode ciphertext as base64: %w", err)
	}
	var clearText = make([]byte, len(cipherText))
	decrypter.CryptBlocks(clearText, cipherText)
	return pkcs7UnPad(clearText, decrypter.BlockSize())
}

// from https://github.com/mergermarket/go-pkcs7/blob/master/pkcs7.go - so small it's probably immoral to add a dependency
func pkcs7Pad(buf []byte, size int) []byte {
	bufLen := len(buf)
	padLen := size - bufLen%size
	padded := make([]byte, bufLen+padLen)
	copy(padded, buf)
	for i := 0; i < padLen; i++ {
		padded[bufLen+i] = byte(padLen)
	}
	return padded
}
func pkcs7UnPad(padded []byte, size int) ([]byte, error) {
	if len(padded)%size != 0 {
		return nil, errors.New("block size incorrect for padded input")
	}
	bufLen := len(padded) - int(padded[len(padded)-1])
	buf := make([]byte, bufLen)
	copy(buf, padded[:bufLen])
	return buf, nil
}

func hashUsername(username string) string {
	input := []byte(username)
	hashed := sha1.Sum(input)
	return hex.EncodeToString(hashed[:])
}

func (dc *deviceConnection) newEncrypter() cipher.BlockMode {
	return cipher.NewCBCEncrypter(*dc.cbcCipher, dc.cbcIv)
}
func (dc *deviceConnection) newDecrypter() cipher.BlockMode {
	return cipher.NewCBCDecrypter(*dc.cbcCipher, dc.cbcIv)
}
