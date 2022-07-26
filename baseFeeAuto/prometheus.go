package baseFeeAuto

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/filecoin-project/go-state-types/big"
	types3 "github.com/filecoin-project/lotus/chain/types"
	types "github.com/guowenshuai/lotus_tool/types/baseFeeAuto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	labels = []string{"nonce", "cid", "number", "method", "sealRandEpoch", "seedEpoch",
		"expireEpoch", "remainEpoch", "expireTimeStamp", "gasfeecap", "gaspremium", "gaslimit"}
	mpool_local = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace:   "",
			Subsystem:   "",
			Name:        "mpool_local",
			Help:        "local mpool information",
			ConstLabels: nil,
		}, labels)
	baseFeeCap = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   "",
			Subsystem:   "",
			Name:        "baseFeeCap",
			Help:        "baseFeeCap atto",
			ConstLabels: nil,
		})
)

func init() {
	prometheus.MustRegister(mpool_local)
	prometheus.MustRegister(baseFeeCap)
}

func resetMpoolGauge() {
	mpool_local.Reset()
}

func setMpoolGauge(msgLabel *types.MessageLabels, current *types3.Message) {
	mpool_local.With(map[string]string{
		"nonce":           fmt.Sprintf("%d", msgLabel.Message.Nonce),
		"cid":             msgLabel.Message.Cid,
		"number":          fmt.Sprintf("%d", msgLabel.Message.SectorNumber),
		"method":          msgLabel.Message.Method.String(),
		"sealRandEpoch":   msgLabel.SealRandEpoch.String(),
		"seedEpoch":       msgLabel.SeedEpoch.String(),
		"expireEpoch":     msgLabel.ExpireEpoch.String(),
		"remainEpoch":     msgLabel.RemainEpoch.String(),
		"expireTimeStamp": msgLabel.ExpireTimeStamp.String(),
		"gasfeecap":       current.GasFeeCap.String(),
		"gaspremium":      current.GasPremium.String(),
		"gaslimit":        fmt.Sprintf("%d", current.GasLimit),
	}).Set(float64(msgLabel.Message.Nonce))
}

func Monitor(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	logrus.Fatal(http.ListenAndServe(addr, nil))
}

func ResetBaseFeeCap(value big.Int) {
	int64, err := strconv.ParseInt(value.String(), 10, 64)
	if err != nil {
		logrus.Errorf("basefeecap err %s", err.Error())
		return
	}
	baseFeeCap.Set(float64(int64))
}
