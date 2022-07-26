package util

import (
	"testing"

	uuid "github.com/satori/go.uuid"
)

func Test_Plus(t *testing.T) {
	v := NewSemVersion()
	t.Log(MinorPlus(v))
	t.Log(uuid.NewV4().String())
}
