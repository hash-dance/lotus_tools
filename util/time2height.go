package util

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin"
)

var (
	height     = abi.ChainEpoch(12000)
	heightTime = time.Time{}
)

func init() {
	tt, _ := time.Parse("2006-01-02 15:04:05", "2020-08-29 02:00:00")
	heightTime = tt
}

func Time2height(nextTime time.Time) abi.ChainEpoch {
	subSeconds :=  nextTime.Sub(heightTime).Seconds()
	nextHeight := height + abi.ChainEpoch(int64(subSeconds/float64(builtin.EpochDurationSeconds)))
	if nextHeight < 0 {
		nextHeight = 0
	}
	return nextHeight
}

func Height2time(nextHeight abi.ChainEpoch) time.Time {
	if nextHeight < 0 {
		return time.Now().Add(-time.Duration(time.Now().UnixNano()))
	}
	subDuration := (nextHeight - height) * builtin.EpochDurationSeconds
	return heightTime.Add(time.Duration(subDuration*1e9))
}
