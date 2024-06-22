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
	"github.com/mergermarket/go-pkcs7"
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
	if err = pem.Encode(&outBytes, &pem.Block{
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
	padded, _ := pkcs7.Pad(clearText, encrypter.BlockSize())
	cipherText := make([]byte, len(padded))
	encrypter.CryptBlocks(cipherText, padded)
	return base64.StdEncoding.EncodeToString(cipherText)
}

func decryptAndRemovePadding(decrypter cipher.BlockMode, base64Ciphertext string) ([]byte, error) {
	cipherText, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("could not decode ciphertext as base64: %w", err)
	}
	var clearText = make([]byte, len(cipherText))
	decrypter.CryptBlocks(clearText, cipherText)
	return pkcs7.Unpad(clearText, decrypter.BlockSize())
}

func hashUsername(username string) string {
	hashed := sha1.Sum([]byte(username))
	return hex.EncodeToString(hashed[:])
}

func (dc *deviceConnection) newEncrypter() cipher.BlockMode {
	return cipher.NewCBCEncrypter(*dc.cbcCipher, dc.cbcIv)
}
func (dc *deviceConnection) newDecrypter() cipher.BlockMode {
	return cipher.NewCBCDecrypter(*dc.cbcCipher, dc.cbcIv)
}
