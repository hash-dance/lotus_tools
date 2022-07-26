package types

import (
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/guowenshuai/lotus_tool/types"
)

type MessageLabels struct {
	Message         *types.Message `json:"message"`
	SealRandEpoch   abi.ChainEpoch `json:"seal_rand_epoch"` // epoch高度
	SeedEpoch       abi.ChainEpoch `json:"seed_epoch"`
	ExpireEpoch     abi.ChainEpoch `json:"expire_epoch"` // 过期epoch
	RemainEpoch     abi.ChainEpoch `json:"remain_epoch"` // 剩余epoch
	ExpireTimeStamp time.Time      `json:"expire_time_stamp"` // 过期epoch换算的过期时间
	CatchTime       time.Time      `json:"catch_time"`        // 从消息池抓取的时间
}
