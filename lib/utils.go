package lib

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
)

// GobPress will gob encode and compress a struct
func GobPress(s interface{}, data io.Writer) error {

	j, err := json.Marshal(s)
	if err != nil {
		return err
	}

	// Encrypt the data
	enc, err := Encrypt(j)
	if err != nil {
		return err
	}

	data.Write(enc)

	return nil
}

// UngobUnpress will gob decode and decompress a struct
func UngobUnpress(s interface{}, data []byte) error {

	// Decrypt the data
	decryptData, err := Decrypt(data)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(decryptData, &s); err != nil {
		return err
	}

	return nil
}

// ByteSplit will split []byte into chunks of lim
func ByteSplit(buf []byte, lim int) [][]byte {
	var chunk []byte

	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}

	if len(buf) > 0 {
		chunks = append(chunks, buf[:])
	}

	return chunks
}

// RandomString just generates a crappy random string.
// This is not a crypto related function, so "how random" really doesnt matter.
func RandomString(strlen int) string {

	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := range result {
		result[i] = chars[mrand.Intn(len(chars))]
	}
	return string(result)
}

// Encrypt will encrypt a byte stream
// https://golang.org/pkg/crypto/cipher/#NewCFBEncrypter
func Encrypt(plaintext []byte) ([]byte, error) {
	key, _ := hex.DecodeString(cryptKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	plaintext, err = pkcs7pad(plaintext, aes.BlockSize) // BlockSize = 16
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCBCEncrypter(block, iv)
	stream.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// Decrypt will decrypt a byte stream
// https://golang.org/pkg/crypto/cipher/#example_NewCFBDecrypter
func Decrypt(ciphertext []byte) ([]byte, error) {
	key, _ := hex.DecodeString(cryptKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("Cipher text too short")
	}

	// Ensure we have the correct blocksize
	if (len(ciphertext) % aes.BlockSize) != 0 {
		return nil, errors.New("Cipher text is not the expected length")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCBCDecrypter(block, iv)
	stream.CryptBlocks(ciphertext, ciphertext)

	ciphertext, err = pkcs7strip(ciphertext, aes.BlockSize)
	if err != nil {
		return nil, err
	}

	return ciphertext, nil
}

// pkcs7strip remove pkcs7 padding
// https://gist.github.com/nanmu42/b838acc10d393bc51cb861128ce7f89c
func pkcs7strip(data []byte, blockSize int) ([]byte, error) {
	length := len(data)

	if length == 0 {
		return nil, errors.New("pkcs7: Data is empty")
	}

	if length%blockSize != 0 {
		return nil, errors.New("pkcs7: Data is not block-aligned")
	}

	padLen := int(data[length-1])
	ref := bytes.Repeat([]byte{byte(padLen)}, padLen)

	if padLen > blockSize || padLen == 0 || !bytes.HasSuffix(data, ref) {
		return nil, errors.New("pkcs7: Invalid padding")
	}

	return data[:length-padLen], nil
}

// pkcs7pad add pkcs7 padding
// https://gist.github.com/nanmu42/b838acc10d393bc51cb861128ce7f89c
func pkcs7pad(data []byte, blockSize int) ([]byte, error) {

	if blockSize < 0 || blockSize > 256 {
		return nil, fmt.Errorf("pkcs7: Invalid block size %d", blockSize)
	}

	padLen := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)

	return append(data, padding...), nil
}
