package util

import (
	"encoding/base32"
	"math/rand"
)

func EnsureBetweenRandom(val, min, max int) int {
	if val > max || val < min {
		return rand.Intn(max-min) + min
	}

	return val

}

func RandStringBytes(length int) string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return base32.StdEncoding.EncodeToString(randomBytes)[:length]
}
