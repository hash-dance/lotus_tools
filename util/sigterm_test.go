package util

import (
	"testing"
)

func Test_aes256cfb(t *testing.T) {
	t.Log(CFBEncrypter("password"))
	t.Log(CFBDecrypter("a1a64541a8518f2f2a51fabf3e2c2046eabe5202f9358fdd"))
}
