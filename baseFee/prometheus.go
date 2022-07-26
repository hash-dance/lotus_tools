package baseFee

import (
	"context"
	"net/http"
	"strconv"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)


var (
	// Create a summary to track fictional interservice RPC latencies for three
	// distinct services with different latency distributions. These services are
	// differentiated via a "service" label.
	baseFee = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace:   "",
			Subsystem:   "",
			Name:        "baseFee",
			Help:        "baseFee atto",
			ConstLabels: nil,
		})
	start = float64(build.MinimumBaseFee)
	filWidth = float64(500_000_000)

	baseFeeHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "baseFee_histogram",
		Help:    "baseFee histogram distributions.",
		Buckets: prometheus.LinearBuckets(start, filWidth, 16),
	})
)

func init() {
	prometheus.MustRegister(baseFee)
	prometheus.MustRegister(baseFeeHistogram)
}

func Monitor(ctx context.Context, api api.FullNode, addr string)  {
	go func() {
		watchBaseFee(api, ctx, func(b big.Int) {
			int64, err := strconv.ParseInt(b.String(), 10, 64)
			if err != nil {
				logrus.Errorf("basefee err %s", err.Error())
				return
			}
			baseFee.Set(float64(int64))
			baseFeeHistogram.Observe(float64(int64))
		})
	}()
	http.Handle("/metrics", promhttp.Handler())
	logrus.Fatal(http.ListenAndServe(addr, nil))
}

