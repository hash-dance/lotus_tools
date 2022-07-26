package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/lotus/api"
	"github.com/guowenshuai/lotus_tool/baseFeeAuto"
	"github.com/guowenshuai/lotus_tool/client"
	"github.com/sirupsen/logrus"

	types "github.com/guowenshuai/lotus_tool/types/baseFeeAuto"
)

var config *types.Config

func init() {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
		// 定义时间戳格式
		TimestampFormat: "2006-01-02 15:04:05",
	}
	logrus.SetFormatter(formatter)
	// 初始化启动配置
	config = &types.Config{
		Demo: &types.Lotus{
			Token:   "",
			Address: "",
		},
		Storage: &types.Lotus{
			Token:   "",
			Address: "",
		},
		Setting: &types.Setting{
			RefreshTime:        30,
			RefreshBaseFee:     20,
			BaseFee:            "2000000000",
			BaseFeeMax:         "9000000000",
			BaseFeePercent:     80,
			TimeKeep:           1,
			MpoolThresholdHigh: 0,
			MpoolThresholdLow:  20,
			ProLimit:           68978435,
			PreLimit:           41078435,
			OnceMax:            2,
			LimitAdjustSeed:    30,
			LimitEstimateSeed:  130,
			LimitMaxPremium:    1900000,
			PreBreakLimit:      700000,
			PremiumSeed:        126,
		},
		Prometheus: &types.Prometheus{
			Port: "8881",
		},
		RedisAddress:  "localhost:6379",
		RedisPassword: "password",
		RedisDBNumber: 0,
	}
	if err := types.LoadYaml("conf.yaml", config); err != nil {
		panic(err)
	}

}

func main() {
	// 106
	DemoAuthToken := config.Demo.Token
	DemoAddr := config.Demo.Address

	header := http.Header{}
	if DemoAuthToken != "" {
		header.Add("Authorization", "Bearer "+DemoAuthToken)
	}

	ctx := context.Background()
	// fmt.Printf("%+v",ctx.Value(""))
	d, _ := json.Marshal(config)
	logrus.Printf("config is: %+v", string(d))
	nodeApi, closer, err := client.NewFullNodeRPC(ctx, "ws://"+DemoAddr+"/rpc/v0", header)
	if err != nil {
		logrus.Fatalf("connecting with lotus demo failed: %s", err)
	}
	defer closer()

	storageToken := config.Storage.Token
	storageAddr := config.Storage.Address

	var storageApi api.StorageMiner
	if storageAddr != "" && storageToken != "" {

		header2 := http.Header{}
		if storageToken != "" {
			header.Add("Authorization", "Bearer "+storageToken)
		}
		c, closer2, err := client.NewStorageMinerRPC(ctx, "ws://"+storageAddr+"/rpc/v0", header2)
		if err != nil {
			logrus.Fatalf("connecting with lotus storage failed: %s", err)
		}
		defer closer2()
		storageApi = c
	}

	// todo
	// redis.RedisConnect(config.RedisAddress, config.RedisPassword, config.RedisDBNumber)

	go func() {
		baseFeeAuto.Monitor(fmt.Sprintf(":%s", config.Prometheus.Port))
	}()
	go func() {
		tmer, err := baseFeeAuto.NewLocalPool(ctx, nodeApi, storageApi, config)
		if err != nil {
			panic("start msgSync err: " + err.Error())
		}
		tmer.Start()
	}()

	<-ctx.Done()
}
