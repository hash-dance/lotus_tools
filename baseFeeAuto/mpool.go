package baseFeeAuto

import (
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/guowenshuai/dingrobot"
	dingmessage "github.com/guowenshuai/dingrobot/message"
	"github.com/guowenshuai/lotus_tool/api"
	types2 "github.com/guowenshuai/lotus_tool/types"
	"github.com/guowenshuai/lotus_tool/util"
	"github.com/sirupsen/logrus"
)

func (p *MpoolQueue) DoClean() {
	timer := util.Ticker(p.ctx, time.Second*time.Duration(p.conf.Setting.RefreshBaseFee))

	for {
		select {
		case <-timer:
			base := api.GetBase(p.ctx, p.nodeApi)
			// 收集baseFee数据
			p.addFee(base)
			if base.LessThan(p.feecap) {
				logrus.Infof("current basefee is %s, less than setting %s", base.String(), p.feecap)
				p.MpoolClean()
			} else {
				logrus.Infof("current basefee is %s, greater than setting %s", base.String(), p.feecap)
			}
		case <-p.ctx.Done():
			logrus.Info("exist watch deals")
			return
		}
	}
}

// 开始执行提价功能
func (p *MpoolQueue) MpoolClean() {
	var filter map[address.Address]struct{}
	filter = map[address.Address]struct{}{}

	for _, a := range p.conf.Setting.Addresses {
		add, err := address.NewFromString(a)
		if err != nil {
			logrus.Errorf("err add: %s\n", err.Error())
			return
		}
		filter[add] = struct{}{}
	}

	msgs, err := p.nodeApi.MpoolPending(p.ctx, types.EmptyTSK)
	if err != nil {
		logrus.Errorf("MpoolPending: %s", err.Error())
		return
	}

	count := 0
	estimateLimit := make(map[abi.MethodNum]int64)
	for idx, msg := range msgs {
		if filter != nil {
			if _, has := filter[msg.Message.From]; !has {
				continue
			}
		}
		count++
		if count > p.conf.Setting.OnceMax {
			return
		}
		// 估算limit费用
		var currentLimit int64
		// if v, ok := estimateLimit[msg.Message.Method]; ok {
		// 	currentLimit = v
		// } else {
		limit, err := p.nodeApi.GasEstimateGasLimit(p.ctx, &msg.Message, types.EmptyTSK)
		if err != nil { // 估算失败
			// no pre-committed
			logrus.Errorf("gasEstimateGasLimit limit err %s", err.Error())
			sub := "no pre-committed"
			if strings.Index(err.Error(), sub) != -1 {
				logrus.Errorf("hit no pre-committed err, set limit min: %s", err.Error())
				limit = p.conf.Setting.PreBreakLimit
			}
		}
		limit = limit * 105 / 100
		// 估算成功
		estimateLimit[msg.Message.Method] = limit
		currentLimit = limit
		// }
		// 获取详情, 查询超时时间, 超时爆掉消息

		// 获取code
		code, err := p.getCode(msg.Message.To)
		if err != nil {
			logrus.Errorf("getcode err %s", err.Error())
		}
		lmsg := types2.LotusMessageConvert(p.ctx, p.nodeApi, code, &msg.Message)

		logrus.Info("msg number: ", lmsg.SectorNumber)
		logrus.Infof("msg info: %+v ", lmsg)

		// 解析获取时间,是否超时
		sealEpoch, seedEpoch, expireEpoch, _, err := p.getTickerEpoch(&msg.Message, lmsg.SectorNumber)
		if err != nil {
			logrus.Warnf("get ticker epoch err %s", err.Error())

		} else {
			if expireEpoch > 0 {
				ts, err := p.nodeApi.ChainHead(p.ctx)
				if err == nil {
					remainEpoch := expireEpoch - ts.Height()
					if remainEpoch < 0 {
						currentLimit = p.conf.Setting.PreBreakLimit
						logrus.Infof("clean: nonce [%d] method [%d] 任务超时, e1: %d, e2: %d, ex: %d, rm: %d 设置limit %d 意图爆掉消息", msg.Message.Nonce,
							msg.Message.Method, sealEpoch, seedEpoch, expireEpoch, remainEpoch, currentLimit)
					}
				}
			}
		}

		go p.doImprove(msgs[idx], currentLimit, p.feecap)
	}
}

func (p *MpoolQueue) doImprove(localmsg *types.SignedMessage, limit int64, nextFee big.Int) {
	entry := logrus.WithFields(logrus.Fields{
		"nonce": localmsg.Message.Nonce,
	})
	// 获取消息
	entry.Infof("nonce:%d method:%d limit:%d cap:%d prem:%d\n", localmsg.Message.Nonce, localmsg.Message.Method,
		localmsg.Message.GasLimit, localmsg.Message.GasFeeCap, localmsg.Message.GasPremium)

	message := localmsg.Message
	entry.Infof("估算limit-1: %d\n", limit)

	if limit == 0 {
		limit = message.GasLimit
	}

	seedEstimate := p.conf.Setting.LimitEstimateSeed

	if message.Method == builtin.MethodsMiner.ProveCommitSector {
		if limit != p.conf.Setting.PreBreakLimit {
			limit = limit * seedEstimate / 100
		}
		if limit > p.conf.Setting.ProLimit {
			limit = p.conf.Setting.ProLimit
		}
	} else if message.Method == builtin.MethodsMiner.PreCommitSector {
		if limit != p.conf.Setting.PreBreakLimit {
			limit = limit * seedEstimate / 100
		}
		if limit > p.conf.Setting.PreLimit {
			limit = p.conf.Setting.PreLimit
		}
	}

	entry.Infof("估算limit-2: %d\n", limit)

	if nextFee.GreaterThanEqual(big.NewInt(10000000000)) { // 10 nano
		logrus.Warn("too large baseFee, more than 10nano")
		return
	}
	if limit > 1000_000_000 { // 10 nano
		logrus.Warn("too large limit, more than 1000_000_000")
		return
	}
	// 防止多次调整费用,导致小费过高
	// 如果是等价的basefee, 做检测, 如果不是等价的basefee, 直接修改
	seed := p.conf.Setting.LimitAdjustSeed
	if nextFee.Equals(message.GasFeeCap) {
		// 超时的任务的limit,
		if limit != p.conf.Setting.PreBreakLimit {
			//  0.7*message.GasLimit =< limit >=  1.3*message.GasLimit
			if limit >= (message.GasLimit*(100-seed)/100) && limit <= (message.GasLimit*(100+seed)/100) {
				entry.Warnf("feecap equal, nextlimit [%d] near by current [%d],  skip", limit, message.GasLimit)
				return
			}

			// if limit <= message.GasLimit {
			// 	entry.Warnf("feecap equal, nextlimit [%d] less than current [%d],  skip", limit, message.GasLimit)
			// 	return
			// }
		} else {
			if limit == message.GasLimit {
				entry.Warnf("feecap equal, nextlimit [%d] eq current [%d],  skip", limit, message.GasLimit)
				return
			}
		}
	}

	message.GasLimit = limit
	message.GasPremium = big.Div(message.GasPremium, big.NewInt(100))
	message.GasPremium = big.Mul(message.GasPremium, big.NewInt(p.conf.Setting.PremiumSeed))
	message.GasFeeCap = nextFee
	entry.Infof("提费 nonce:%d method:%d limit:%d cap:%d prem:%d\n", message.Nonce, message.Method,
		message.GasLimit, message.GasFeeCap, message.GasPremium)

	if message.GasPremium.GreaterThan(big.NewInt(p.conf.Setting.LimitMaxPremium)) { // 100,000 * 1.25^13
		// if message.GasPremium.GreaterThan(big.NewInt(97)) { // 100,000 * 1.25^13
		entry.Warnf("GasPremium %s too large, stop improve msg", message.GasPremium)
		// todo 小费过高, 某些地方有问题, 报警
		robot := dingrobot.NewRobot(p.conf.Alert.DingDing.URL)
		robot.Send(dingmessage.MarkdownMessage{
			MarkdownContent: dingmessage.MarkdownContent{
				Title: fmt.Sprintf("消息调费失败 %d\n", message.Nonce),
				Text: fmt.Sprintf("# GasPremium过高, 查看是否消息堵塞\n  # nonce:%d method:%d limit:%d cap:%d prem:%d\n >钱包: %s", message.Nonce, message.Method,
					message.GasLimit, message.GasFeeCap, message.GasPremium, p.conf.Setting.Addresses),
			},
			At: dingmessage.At{
				IsAtAll: true,
			},
		})
		return
	}
	{
		smsg, err := p.nodeApi.WalletSignMessage(p.ctx, message.From, &message)
		if err != nil {
			entry.Errorf("failed to sign message: %s", err.Error())
			return
		}

		cid, err := p.nodeApi.MpoolPush(p.ctx, smsg)
		if err != nil {
			entry.Errorf("failed to push new message to mempool: %s", err.Error())
			return
		}
		entry.Infof("new message cid: %s, method:%d limit:%d cap:%d prem:%d",
			cid, message.Method, message.GasLimit, message.GasFeeCap, message.GasPremium)
	}
}
