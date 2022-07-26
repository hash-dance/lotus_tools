package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/guowenshuai/lotus_tool/client"
	db "github.com/guowenshuai/lotus_tool/db/mongo"
	"github.com/guowenshuai/lotus_tool/util"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	ltypes "github.com/guowenshuai/lotus_tool/types"
)

func getNode(conf *ltypes.Config) (api.FullNode, jsonrpc.ClientCloser) {

	// DemoAuthToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.Z1FqAXG9w0UANRI--vyI4rGGh6tX-J-qq6bdp7-A5_c"
	DemoAuthToken := conf.Lotus.Token
	// DemoAddr := "172.18.6.106:1234"
	DemoAddr := conf.Lotus.Address

	header := http.Header{}
	if DemoAuthToken != "" {
		header.Add("Authorization", "Bearer "+DemoAuthToken)
	}

	nodeApi, closer, err := client.NewFullNodeRPC(context.Background(), "ws://"+DemoAddr+"/rpc/v0", http.Header{})
	if err != nil {
		logrus.Fatalf("connecting with lotus failed: %s", err)
	}
	return nodeApi, closer
}

var (
	globalDB   *mongo.Database
	globalConf *ltypes.Config
)

// 错误信息
// vendor/github.com/filecoin-project/go-state-types/exitcode/reserved.go:7
func main() {
	ctx := util.SigTermCancelContext(context.Background())

	conf := initConfig()
	globalConf = conf
	database, err := db.Connect(ctx, conf)
	if err != nil {
		logrus.Fatalf("connecting mongo err: %s\n", err)
	}
	globalDB = database
	// filter := getFilter()

	ht := conf.SyncConfig.Height
	// 删除可能有误差的数据, 配置高度的前后20个块
	globalDB.Collection(db.BlockCollectionName).DeleteMany(ctx, bson.D{{"height", bson.D{{"$gt", ht - 20}, {"$lt", ht + 20}}}})

	ht -= 20
	setSyncHeight(ctx, ht) // 配置高度

	go DoSync(ctx)

	<-ctx.Done()
	db.Disconnect()

}

func DoSync(ctx context.Context) {
	nodeapi, closer := getNode(globalConf)
	defer closer()

	timer := util.Ticker(ctx, time.Second*time.Duration(globalConf.SyncConfig.TimeWait))
	for {
		select {
		case <-timer:
			// 获取当前数据库中的同步高度
			heightSynced, err := getSyncHeight(ctx)
			if err != nil {
				logrus.Errorf("get sync height %s", err.Error())
				continue
			}
			// 获取链的最新高度
			head, err := nodeapi.ChainHead(ctx)
			if err != nil {
				logrus.Errorf("get chain height %s", err.Error())
				// 获取高度失败, 通常是websocket断掉了.
				// todo 1. 重新连接服务器;
				nodeapi, closer = getNode(globalConf)
				// 2. 回退10个高度
				setSyncHeight(ctx, heightSynced-10)
				continue
			}
			heightHead := head.Height() - 2

			if globalConf.SyncConfig.StopHeight > 0 && heightSynced > globalConf.SyncConfig.StopHeight {
				logrus.Infof("stop at %d", heightSynced)
				return
			}
			// 获取等待同步的高度
			wg := &sync.WaitGroup{}
			syncTODO := 0
			lastHeight := heightSynced
			// for i := heightSynced + 1; i <= heightSynced+abi.ChainEpoch(globalConf.SyncConfig.MaxOnce) && i <= heightHead; i++ {
			for i := heightSynced + 1; i <= heightHead; i++ {
				if heightHasSynced(ctx, i) {
					logrus.Warnf("height %d has synced", i)
					setSyncHeight(ctx, i)
					lastHeight, heightSynced = i, i
					// lastHeight = i
					continue
				}
				if syncTODO > int(globalConf.SyncConfig.MaxOnce) {
					break
				}
				syncTODO += 1
				wg.Add(1)
				lastHeight = i
				go func(ctx context.Context, wg *sync.WaitGroup, next abi.ChainEpoch) {
					defer wg.Done()
					logrus.Infof("start sync %d\n", next)
					syncHeight(ctx, nodeapi, next)
				}(ctx, wg, i)

			}
			wg.Wait()
			setSyncHeight(ctx, lastHeight) // 配置最后更新高度
		case <-ctx.Done():
			logrus.Infof("exit done")
			return
		}
	}
}

func syncHeight(ctx context.Context, nodeapi api.FullNode, height abi.ChainEpoch) {
	// 获取该高度出块信息
	ts, err := nodeapi.ChainGetTipSetByHeight(ctx, height, types.EmptyTSK)
	if err != nil {
		logrus.Panic(err.Error())
	}
	syncHeightBlocks(ctx, nodeapi, ts) // 同步块信息

	return
}
