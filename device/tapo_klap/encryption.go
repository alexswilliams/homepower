package tapo_klap

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/mergermarket/go-pkcs7"
)

type encryptionContext struct {
	block          cipher.Block
	iv             []byte
	signature      []byte
	sequenceNumber int32
}

func setupEncryption(localRemoteAuthBuffer []byte) (*encryptionContext, error) {
	keyHash := sha256.Sum256(append([]byte("lsk"), localRemoteAuthBuffer...))
	ivHash := sha256.Sum256(append([]byte("iv"), localRemoteAuthBuffer...))
	sequence := int32(binary.BigEndian.Uint32(ivHash[sha256.Size-4 : sha256.Size]))
	sigHash := sha256.Sum256(append([]byte("ldk"), localRemoteAuthBuffer...))
	aesCipher, err := aes.NewCipher(keyHash[:16])
	if err != nil {
		return nil, err
	}
	return &encryptionContext{
		block:          aesCipher,
		iv:             ivHash[:12],
		signature:      sigHash[:28],
		sequenceNumber: sequence,
	}, nil
}

func (ec *encryptionContext) getIv() []byte {
	return binary.BigEndian.AppendUint32(bytes.Clone(ec.iv), uint32(ec.sequenceNumber))
}

func (ec *encryptionContext) sign(cipherText []byte) []byte {
	hash := sha256.Sum256(
		append(binary.BigEndian.AppendUint32(bytes.Clone(ec.signature), uint32(ec.sequenceNumber)), cipherText...))
	return hash[:]
}

func (ec *encryptionContext) Encrypt(data []byte) []byte {
	ec.sequenceNumber++
	padded, _ := pkcs7.Pad(data, aes.BlockSize)
	cipherText := make([]byte, len(padded))
	cipher.NewCBCEncrypter(ec.block, ec.getIv()).CryptBlocks(cipherText, padded)
	return append(ec.sign(cipherText), cipherText...)
}

func (ec *encryptionContext) Decrypt(data []byte) ([]byte, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("data must be at least 32 bytes")
	}
	plainText := make([]byte, len(data))
	cipher.NewCBCDecrypter(ec.block, ec.getIv()).CryptBlocks(plainText, data[32:])
	return pkcs7.Unpad(plainText[:len(plainText)-32], aes.BlockSize)
}
