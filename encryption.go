package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

// func xor(str string, key int) []byte {
// 	k1 := key & 0xff
// 	k2 := (key >> 8) & 0xff
// 	k3 := (key >> 16) & 0xff
// 	k4 := (key >> 24) & 0xff

// 	res := make([]byte, 0)
// 	for _, c := range str {
// 		res = append(res, byte((((int(c)^k1)^k2)^k3)^k4))
// 	}
// 	return res
// }

func xor(str []byte, key int) []byte {
	k1 := key & 0xff
	k2 := (key >> 8) & 0xff
	k3 := (key >> 16) & 0xff
	k4 := (key >> 24) & 0xff

	res := make([]byte, 0)
	for _, c := range str {
		res = append(res, byte((((int(c)^k1)^k2)^k3)^k4))
	}
	return res
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func HashPassword(password string, salt []byte) (string, error) {
	salted := append([]byte(password), salt...)
	hash := sha256.Sum256(salted)

	return base64.StdEncoding.EncodeToString(append(hash[:], salt...)), nil
}

func VerifyPassword(hashed_password string, password string, salt []byte) (bool, error) {
	hash, err := HashPassword(password, salt)
	if err != nil {
		return false, err
	}

	return hash == hashed_password, nil
}
