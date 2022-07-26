package api

import (
	"context"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/sirupsen/logrus"
)

func GetBase(ctx context.Context, api api.FullNode) big.Int {
	ts, err := api.ChainHead(ctx)
	if err != nil {
		logrus.Errorf("chain head err %s", err.Error())
		return big.NewInt(0)
	}

	baseFee := ts.Blocks()[0].ParentBaseFee
	logrus.Info(baseFee)
	return baseFee
}
