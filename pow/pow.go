package pow


import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
)

func GetSupply(ctx context.Context, nodeapi api.FullNode, h abi.ChainEpoch) (api.CirculatingSupply, error) {
	ts, _ := nodeapi.ChainGetTipSetByHeight(ctx, h, types.EmptyTSK)
	return nodeapi.StateVMCirculatingSupplyInternal(ctx, ts.Key())
}
