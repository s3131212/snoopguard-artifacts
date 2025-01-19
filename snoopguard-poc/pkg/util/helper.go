package util

import "math/rand"

// RandomString create a random string with the given length.
func RandomString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// ContainString checks if the given string is in the given string slice.
func ContainString(value string, slice []string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// RandomBytes create a random byte slice with the given length.
func RandomBytes(length int) []byte {
	b := make([]byte, length)
	rand.Read(b)
	return b
}
