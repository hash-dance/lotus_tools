package util

import (
	"math/rand"
	"time"

	uuid "github.com/satori/go.uuid"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

// StringWithCharset make string
func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RandString random string by len
func RandString(length int) string {
	return StringWithCharset(length, charset)
}

// RandIID return uuid
func RandIID() string {
	uid, _ := uuid.NewV4()
	return uid.String()
}
