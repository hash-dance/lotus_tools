package util

import (
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
)

func Test_t(t *testing.T) {
	tt, _ := time.Parse("2006-01-02 15:04:05", "2021-01-06 00:00:00")
	t.Log(Height2time(abi.ChainEpoch(300000)).Local())
	t.Log(Time2height(tt))
}

