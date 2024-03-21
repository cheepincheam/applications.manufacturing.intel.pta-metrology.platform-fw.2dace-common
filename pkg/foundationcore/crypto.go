//
//                  INTEL CORPORATION PROPRIETARY INFORMATION
//     This software is supplied under the terms of a license agreement or
//     nondisclosure agreement with Intel Corporation and may not be copied
//     or disclosed except in accordance with the terms of that agreement.
//          Copyright(c) 2009-2019 Intel Corporation. All Rights Reserved.
//
//

package foundationcore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

// our crypto key
var cryptoKey = []byte("intelcloudfoundationcorekeyset32")

// Encrypt - encrypt the string
func Encrypt(data string) (string, error) {
	// generate a new AES cipher using our 32 bytes long key
	c, err := aes.NewCipher(cryptoKey)
	if err != nil {
		return "", err
	}
	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return string(gcm.Seal(nonce, nonce, []byte(data), nil)), nil
}

// Decrypt - decrypt the string
func Decrypt(data string) (string, error) {

	ciphertext := []byte(data)

	c, err := aes.NewCipher(cryptoKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
