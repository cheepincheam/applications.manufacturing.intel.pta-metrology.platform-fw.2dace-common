package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	//"encoding/base64"
	"encoding/hex"
	"io"
	"fmt"
)

var key = "atcloudmlcloudarethesameconfuse!"

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write(([]byte(key)))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Encrypt - encrypt a string
// func Encrypt(data string) (string, error) {
// 	c, err := aes.NewCipher([]byte(createHash(key)))
// 	if err != nil {
// 		return "", err
// 	}

// 	gcm, err := cipher.NewGCM(c)
// 	if err != nil {
// 		return "", err
// 	}

// 	nonce := make([]byte, gcm.NonceSize())
// 	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
// 		return "", err
// 	}
// 	encrypted := gcm.Seal(nonce, nonce, []byte(data), nil)
// 	// Encoding using raw std encoding is needed here to prevent extra padding issue if the encrypted string is
// 	// being marshaled and unmarshaled to and from json object or file.
// 	return base64.RawStdEncoding.EncodeToString(encrypted), nil
// }

// Decrypt - decrypt a string
// func Decrypt(code string) (string, error) {

// 	c, err := aes.NewCipher([]byte(createHash(key)))
// 	if err != nil {
// 		return "", err
// 	}

// 	gcm, err := cipher.NewGCM(c)
// 	if err != nil {
// 		return "", err
// 	}

// 	codeBytes, err := base64.RawStdEncoding.DecodeString(code)
// 	if err != nil {
// 		return "", err
// 	}
// 	nonceSize := gcm.NonceSize()
// 	if len(codeBytes) < nonceSize {
// 		return "", err
// 	}

// 	nonce, ciphertext := codeBytes[:nonceSize], codeBytes[nonceSize:]
// 	ans, err := gcm.Open(nil, nonce, ciphertext, nil)
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(ans), nil
// }

//-- code modified from https://www.melvinvivas.com/how-to-encrypt-and-decrypt-data-using-aes/
func Encrypt(stringToEncrypt string) (string, error) {
	plaintext := []byte(stringToEncrypt)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return fmt.Sprintf("%x", ciphertext), nil
}

func Decrypt(encryptedString string) (string, error) {
	enc, _ := hex.DecodeString(encryptedString)

	//Create a new Cipher Block from the key
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := enc[:nonceSize], enc[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
