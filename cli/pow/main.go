package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	address2 "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/build"
	types2 "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/guowenshuai/lotus_tool/client"
	"github.com/guowenshuai/lotus_tool/pow"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	// VERSION current version
	VERSION = "v0.1.0"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
		// 定义时间戳格式
		TimestampFormat: "2006-01-02 15:04:05",
	}
	logrus.SetFormatter(formatter)

	app := &cli.App{
		Name:    "base fee monitor",
		Version: VERSION,
		Usage:   "base fee watcher",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "api",
				Value:    "172.18.5.202:1234",
				Required: true,
				Usage:    "specify lotus api address",
			}, &cli.Int64Flag{
				Name:     "from",
				Usage:    "calculate from",
				Required: true,
			}, &cli.Int64Flag{
				Name:     "to",
				Usage:    "calculate to",
				Required: true,
			},
		},
	}

	app.Action = func(c *cli.Context) error {
		return run(c)
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) error {
	ctx := context.Background()
	address := c.String("api")
	nodeApi, closer, err := client.NewFullNodeRPC(ctx, "ws://"+address+"/rpc/v0", http.Header{})
	if err != nil {
		logrus.Fatalf("connecting with lotus failed: %s", err)
	}
	defer closer()
	fromHeight := abi.ChainEpoch(c.Int64("from"))
	toHeight := abi.ChainEpoch(c.Int64("to"))

	ts, err := nodeApi.ChainHead(ctx)
	if err != nil {
		return err
	}
	if toHeight <= fromHeight {
		return errors.New("from must more than to height")
	}
	currentHeight := ts.Height()
	if toHeight > currentHeight {
		logrus.Warnf("current height is %s\n", currentHeight)
		toHeight = currentHeight
	}
	logrus.Infof("%s ---> %s\n", fromHeight, toHeight)

	var maddr address2.Address
	lastMined, oneDayMined := big.NewInt(0), big.NewInt(0)
	powerPoints := make([]Point, 0)
	mined1DayPoints := make([]Point, 0)

	fmt.Println("高度\t时间\tPower\tFilCirculating\tFilMined\tFilVested\tFilBurnt\tFilLocked\t24hMined\toneTMined")
	for i := fromHeight; i < toHeight; i += abi.ChainEpoch(builtin.EpochsInDay) {
		ts, err := nodeApi.ChainGetTipSetByHeight(ctx, i, types2.EmptyTSK)
		if err != nil {
			logrus.Warnf("get height %d err: %s\n", i, err.Error())
			continue
		}
		power, err := nodeApi.StateMinerPower(ctx, maddr, ts.Key())
		if err != nil {
			logrus.Warnf("get height %d power err %s\n", i, err.Error())
			continue
		}
		v, e := pow.GetSupply(ctx, nodeApi, i)
		if e != nil {
			logrus.Warnf("get height %d pow err %s\n", i, e.Error())
			continue
		}

		if !lastMined.LessThanEqual(big.NewInt(0)) {
			oneDayMined = big.Sub(v.FilMined, lastMined)
		}
		currentPower := power.TotalPower.QualityAdjPower
		currentPowerT := big.Div(currentPower, big.NewInt(1024*1024*1024*1024))
		onePowerTMined := big.Div(oneDayMined, currentPowerT)
		fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ts.Height(), time.Unix(int64(ts.MinTimestamp()), 0).Format(time.RFC3339), currentPowerT.String(),
			toFile(v.FilCirculating), toFile(v.FilMined), toFile(v.FilVested), toFile(v.FilBurnt), toFile(v.FilLocked),
			toFile(oneDayMined), toFile(onePowerTMined))
		lastMined = v.FilMined

		powerPoints = append(powerPoints, Point{
			X: float64(ts.Height()),
			Y: float64(currentPowerT.Int64()),
		})
		oneDayMinedFloat := float64(big.Div(oneDayMined, big.NewInt(int64(build.FilecoinPrecision))).Int64())
		logrus.Printf("%f", oneDayMinedFloat)
		mined1DayPoints = append(mined1DayPoints, Point{
			X: float64(ts.Height()),
			Y: oneDayMinedFloat,
		})
	}
	// 计算power线性
	a, b := Points2Func(powerPoints[1], powerPoints[len(powerPoints)-2])
	logrus.Infof("power func p = %f * h + (%f)\n", a, b)
	// 计算单日产币线性
	a, b = Points2Func(mined1DayPoints[1], mined1DayPoints[len(mined1DayPoints)-2])
	logrus.Infof("minedOneDay func m = %f * h + (%f)\n", a, b)
	return nil
}

func toFile(amount abi.TokenAmount) string {
	return strings.TrimSpace(strings.TrimRight(types2.FIL(amount).String(), "FIL"))
}
