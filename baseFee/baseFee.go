package baseFee

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	api2 "github.com/guowenshuai/lotus_tool/api"
	"github.com/guowenshuai/lotus_tool/util"
	"github.com/sirupsen/logrus"
)

func watchBaseFee(api api.FullNode, ctx context.Context, setter func(big.Int)) {
	timer := util.Ticker(ctx, time.Second*30)
	base := api2.GetBase(ctx, api)
	setter(base)
	for {
		select {
		case <-timer:
			base := api2.GetBase(ctx, api)
			setter(base)
		case <-ctx.Done():
			logrus.Info("exist watch deals")
			return
		}
	}
}
