package util

import "math/rand"

func EnsureBetweenRandom(val, min, max int) int {
	if val > max || val < min {
		return rand.Intn(max-min) + min
	}

	return val

}
