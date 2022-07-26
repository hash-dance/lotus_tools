package main

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	db "github.com/guowenshuai/lotus_tool/db/mongo"
	ltypes "github.com/guowenshuai/lotus_tool/types"
)

func InsertOneTipSet(ctx context.Context, ts *ltypes.TipSet) (*mongo.InsertOneResult, error) {
	return globalDB.Collection(db.BlockCollectionName).InsertOne(ctx, ts)
}

func DeleteOneTipSet(ctx context.Context, height abi.ChainEpoch) *mongo.SingleResult {
	return globalDB.Collection(db.BlockCollectionName).FindOneAndDelete(ctx, bson.D{{"height", height}})
}

func InsertMany(ctx context.Context, dat []interface{}) (*mongo.InsertManyResult, error) {
	ordered := false
	return globalDB.Collection(db.MessagesCollectionName).InsertMany(ctx, dat, &options.InsertManyOptions{
		Ordered: &ordered,
	})
}

// 同步块高度和爆块信息
func syncHeightBlocks(ctx context.Context, nodeapi api.FullNode, ts *types.TipSet) error {
	// 同步message信息
	if err := getEpochTrace(ctx, nodeapi, ts); err != nil {
		logrus.Warnf("getEpochTrace record %d", ts.Height())
		// DeleteOneTipSet(ctx, ts.Height())
	}
	if _, err := InsertOneTipSet(ctx, ltypes.LotusTipSetConvert(ctx, nodeapi, ts)); err != nil {
		logrus.Errorf("insert tipSet [%s] err %s", ts.Height().String(), err.Error())
		return err
	}

	return nil

}

// 获取该高度message信息
func getEpochTrace(ctx context.Context, nodeapi api.FullNode, ts *types.TipSet) error {
	var msgs []*types.Message

	start := time.Now()
	invocRes, err := nodeapi.StateCompute(ctx, ts.Height(), msgs, ts.Key())
	if err != nil {
		logrus.Errorf("error message StateCall %s", err.Error())
		return nil
	}
	baseFee := ts.Blocks()[0].ParentBaseFee

	logrus.Infof("stateCompute %d, use %s\n", ts.Height(), time.Now().Sub(start))

	start = time.Now()
	wg := &sync.WaitGroup{}
	allTrace := make([]interface{}, 0)
	lock := &sync.Mutex{}
	traceAppend := func(d interface{}) {
		lock.Lock()
		allTrace = append(allTrace, d)
		lock.Unlock()
	}

	logrus.Infof("len %d", len(invocRes.Trace))
	for idx, _ := range invocRes.Trace {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, im *api.InvocResult) {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					return
				}
			}()
			codeCache := map[address.Address]cid.Cid{}
			lock := &sync.Mutex{}
			getCode := func(addr address.Address) (cid.Cid, error) {
				lock.Lock()
				defer lock.Unlock()
				if c, found := codeCache[addr]; found {
					return c, nil
				}
				c, err := nodeapi.StateGetActor(ctx, addr, ts.Key())
				if err != nil {
					return cid.Cid{}, err
				}

				codeCache[addr] = c.Code
				return c.Code, nil
			}
			// todo im.Msg.From
			// todo im.Msg.To
			if im == nil {
				return
			}
			// logrus.Infof("parse msg %s\n", im.Msg.Cid())
			for {
				select {
				case <-ctx.Done():
					return // returning not to leak the goroutine
				default:
					res := ltypes.LotusInvocResultConvert(ctx, nodeapi, baseFee.String(), im, ts.Height(), ts.MinTimestamp(), getCode)
					// allTrace = append(allTrace, res)
					traceAppend(res)
					return
				}
			}
		}(ctx, wg, invocRes.Trace[idx])
	}
	wg.Wait()

	logrus.Infof("parseMsg %d, use %s", ts.Height(), time.Now().Sub(start))

	if _, err := InsertMany(ctx, allTrace); err != nil {
		logrus.Errorf("insertMany %s", err.Error())
		return err
	}
	return nil

}
