package types

import (
	"context"
	"encoding/json"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

type TipSet struct {
	Height        abi.ChainEpoch `json:"height"`
	Time          time.Time      `json:"time"`
	ParentBaseFee string         `json:"parent_base_fee"`
	Blocks        []BlockHeader  `json:"blocks"`
}

type BlockHeader struct {
	Miner     string   `json:"miner"`
	Timestamp uint64   `json:"timestamp"`
	Source    string   `json:"source"`
	Cids      []string `json:"cids"`
}

func LotusTipSetConvert(ctx context.Context, nodeapi api.FullNode, ts *types.TipSet) *TipSet {
	blocks := make([]BlockHeader, 0)
	for _, bcid := range ts.Blocks() {
		source, _ := json.Marshal(bcid)
		cids := make([]string, 0)
		if msgs, err := nodeapi.ChainGetBlockMessages(ctx, bcid.Cid()); err == nil {
			for _, c := range msgs.Cids {
				cids = append(cids, c.String())
			}
		}
		blocks = append(blocks, BlockHeader{
			Miner:     bcid.Miner.String(),
			Timestamp: bcid.Timestamp,
			Source:    string(source),
			Cids:      cids,
		})
	}
	return &TipSet{
		Height:        ts.Height(),
		Time:          time.Unix(int64(ts.MinTimestamp()), 0),
		ParentBaseFee: ts.Blocks()[0].ParentBaseFee.String(),
		Blocks:        blocks,
	}
}
