package baseFeeAuto

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	types2 "github.com/guowenshuai/lotus_tool/types"
	aType "github.com/guowenshuai/lotus_tool/types/baseFeeAuto"
	"github.com/guowenshuai/lotus_tool/util"
	"github.com/ipfs/go-cid"
	"github.com/sirupsen/logrus"
)

type MpoolQueue struct {
	nodeApi    api.FullNode
	storageApi api.StorageMiner
	ctx        context.Context
	conf       *aType.Config
	filter     map[address.Address]struct{}
	feecap     types.BigInt
	maxFeecap  types.BigInt
	feeQueue   FeeCapQueue // 队列中消息费
	codeCache  map[address.Address]cid.Cid
	getCode    func(addr address.Address) (cid.Cid, error)
	localpools map[abi.SectorNumber]aType.MessageLabels
}

var (
	mpoolQueue *MpoolQueue
)
var msgTimeout = time.Hour * 48

func NewLocalPool(ctx context.Context, nodeapi api.FullNode, storageApi api.StorageMiner, conf *aType.Config) (*MpoolQueue, error) {
	if mpoolQueue == nil {
		filter := map[address.Address]struct{}{}
		for _, a := range conf.Setting.Addresses {
			add, err := address.NewFromString(a)
			if err != nil {
				logrus.Errorf("err add: %s\n", err.Error())
				return nil, err
			}
			filter[add] = struct{}{}
		}
		feecap, err := types.BigFromString(conf.Setting.BaseFee)
		if err != nil {
			logrus.Errorf("parsing gas-feecap: %s", err.Error())
			return nil, err
		}
		maxFeecap, err := types.BigFromString(conf.Setting.BaseFeeMax)
		if err != nil {
			logrus.Errorf("parsing max-gas-feecap: %s", err.Error())
			return nil, err
		}
		codeCache := map[address.Address]cid.Cid{}
		ResetBaseFeeCap(feecap)
		return &MpoolQueue{
			nodeApi:    nodeapi,
			storageApi: storageApi,
			ctx:        ctx,
			conf:       conf,
			filter:     filter,
			feecap:     feecap,
			maxFeecap:  maxFeecap,
			codeCache:  codeCache,
			getCode: func(addr address.Address) (cid.Cid, error) {
				if c, found := codeCache[addr]; found {
					return c, nil
				}
				c, err := nodeapi.StateGetActor(ctx, addr, types.EmptyTSK)
				if err != nil {
					return cid.Cid{}, err
				}

				codeCache[addr] = c.Code
				return c.Code, nil
			},
			localpools: make(map[abi.SectorNumber]aType.MessageLabels),
		}, nil
	}

	return mpoolQueue, nil
}

// SyncMsg 获取消息池消息, 同步到redis
func (p *MpoolQueue) Start() {
	go p.DoClean()
	timer := util.Ticker(p.ctx, time.Second*time.Duration(p.conf.Setting.RefreshTime))
	p.syncMsg()
	for {
		select {
		case <-timer:
			p.syncMsg()
		case <-p.ctx.Done():
			logrus.Info("exist watch deals")
			return
		}
	}
}
func (p *MpoolQueue) syncMsg() {
	// 1. 获取消息池消息
	msgs, err := p.nodeApi.MpoolPending(p.ctx, types.EmptyTSK)
	if err != nil {
		logrus.Errorf("MpoolPending: %s", err.Error())
		return
	}

	ts, err := p.nodeApi.ChainHead(p.ctx)
	if err != nil {
		logrus.Infof("get current chain err %s", err.Error())
		return
	}

	// estimateLimit := make(map[abi.MethodNum]int64)
	// 2. 按照nonce排序
	// codeCache := map[address.Address]cid.Cid{}
	// getCode := func(addr address.Address) (cid.Cid, error) {
	// 	if c, found := codeCache[addr]; found {
	// 		return c, nil
	// 	}
	// 	c, err := p.nodeApi.StateGetActor(p.ctx, addr, types.EmptyTSK)
	// 	if err != nil {
	// 		return cid.Cid{}, err
	// 	}

	// 	codeCache[addr] = c.Code
	// 	return c.Code, nil
	// }
	resetMpoolGauge() // 清空指标
	thislocalpools := make(map[abi.SectorNumber]aType.MessageLabels)
	msgtotal := 0
	for _, msg := range msgs {
		if p.filter != nil {
			if _, has := p.filter[msg.Message.From]; !has {
				continue
			}
		}
		msgtotal++
		nonce := fmt.Sprintf("%d", msg.Message.Nonce)
		entry := logrus.WithFields(logrus.Fields{
			"nonce": nonce,
		})
		code, _ := p.getCode(msg.Message.To)

		last := aType.MessageLabels{}
		lmsg := types2.LotusMessageConvert(p.ctx, p.nodeApi, code, &msg.Message)
		if v, ok := p.localpools[lmsg.SectorNumber]; ok {
			last = v
			thislocalpools[lmsg.SectorNumber] = v
		} else {
			last.Message = lmsg
			sealEpoch, seedEpoch, expireEpoch, expireTimeStamp, _ := p.getTickerEpoch(&msg.Message, lmsg.SectorNumber)
			// if err != nil {
			// 	entry.Warnf("get ticker epoch err %s", err.Error())
			// 	continue
			// }
			last.SealRandEpoch = sealEpoch
			last.SeedEpoch = seedEpoch
			last.ExpireEpoch = expireEpoch
			last.ExpireTimeStamp = expireTimeStamp
			if last.ExpireEpoch > 0 {
				last.RemainEpoch = last.ExpireEpoch - ts.Height()
			}
			thislocalpools[lmsg.SectorNumber] = last
		}

		entry.Infof("sync message %+v", last)
		// 输出到指标
		setMpoolGauge(&last, &msg.Message)

		// if !hasLast {
		// 	entry.Infof("the message is new for %s", time.Now())
		// 	continue
		// }
		// entry.Infof("the message is old for %s, try to handle", last.CatchTime.String())

		// 剩余时间
		// if last.RemainEpoch < 0 {
		// 	currentLimit = p.conf.Setting.PreBreakLimit
		// 	logrus.Infof("sync: nonce [%d] method [%d] 任务超时, e1: %d, e2: %d, ex: %d, rm: %d 设置limit %d 意图爆掉消息", msg.Message.Nonce,
		// 		msg.Message.Method, sealEpoch, seedEpoch, expireEpoch, last.RemainEpoch, currentLimit)
		// }

		// todo 注销阶梯处理消息
		// p.handleMsg(msgs[idx], last.CatchTime, entry, currentLimit)
	}
	// 替换localpools
	p.localpools = thislocalpools
	logrus.Infof("localpools has %d messges", len(p.localpools))

	// 检测发现消息数量,
	logrus.Infof("msg pool count is %d", msgtotal)

	if p.checkMsgTooMore(msgtotal) {
		// 调高basefee
		p.ResetBaseFeeCap(p.getNextFeeCap(p.conf.Setting.BaseFeePercent, true))
	}
	if p.checkMsgTooLow(msgtotal) {
		// 按照百分比调整basefeecap
		p.ResetBaseFeeCap(p.getNextFeeCap(p.conf.Setting.BaseFeePercent, false))
	}
}

// WatchMsg 检查消息和时间, 对达到阶梯限制的消息进行单独疏通
func (p *MpoolQueue) handleMsg(msg *types.SignedMessage, last time.Time, entry *logrus.Entry, limit int64) {
	// 判断阶梯时间
	hours := time2now(last)
	entry.Infof("%f hours ago", hours)
	currentBaseFee := "0"
	for _, v := range p.conf.Setting.StepFee {
		if hours > v.Hour {
			currentBaseFee = v.Fee
		}
	}
	if currentBaseFee == "0" {
		return
	}

	feecap, err := types.BigFromString(currentBaseFee)
	if err != nil {
		entry.Errorf("parsing gas-feecap: %s", err.Error())
		return
	}
	entry.Infof("last time %f hours to now, go to improve to baseFee %s", hours, feecap)
	go p.doImprove(msg, limit, feecap)
}

func (p *MpoolQueue) getTickerEpoch(msg *types.Message, sectorNumber abi.SectorNumber) (abi.ChainEpoch,
	abi.ChainEpoch, abi.ChainEpoch, time.Time, error) {
	var sealEpoch, seedEpoch, expireEpoch abi.ChainEpoch
	var expireTimeStamp time.Time
	// 获取ticker
	if msg.Method == builtin.MethodsMiner.PreCommitSector || msg.Method == builtin.MethodsMiner.ProveCommitSector {
		if info, err := p.storageApi.SectorsStatus(p.ctx, sectorNumber, true); err != nil {
			return sealEpoch, seedEpoch, expireEpoch, expireTimeStamp, err
		} else {
			// logrus.Infof("sectorNumber %s: %+v", sectorNumber.String(), info)
			sealEpoch = info.Ticket.Epoch
			seedEpoch = info.Seed.Epoch
		}

		switch msg.Method {
		case builtin.MethodsMiner.PreCommitSector:
			expireEpoch = sealEpoch + miner.MaxPreCommitRandomnessLookback
			oldts, err := p.nodeApi.ChainGetTipSetByHeight(p.ctx, sealEpoch, types.EmptyTSK)
			if err == nil {
				expireTimeStamp = time.Unix(int64(oldts.MinTimestamp())+int64(miner.MaxPreCommitRandomnessLookback*30), 0)
			}
		case builtin.MethodsMiner.ProveCommitSector:
			expireEpoch = seedEpoch + builtin.EpochsInDay
			oldts, err := p.nodeApi.ChainGetTipSetByHeight(p.ctx, seedEpoch, types.EmptyTSK)
			if err == nil {
				expireTimeStamp = time.Unix(int64(oldts.MinTimestamp())+int64(builtin.EpochsInDay*30), 0)
			}
		}
	}
	return sealEpoch, seedEpoch, expireEpoch, expireTimeStamp, nil
}

func time2now(last time.Time) float64 {
	return time.Now().Sub(last).Hours()
}
