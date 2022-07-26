package baseFeeAuto

import (
	"sort"
	"time"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/sirupsen/logrus"
)

type FeeCapQueue []big.Int

func (f FeeCapQueue) Len() int {
	return len(f)
}

func (f FeeCapQueue) Less(i, j int) bool {
	return f[i].LessThan(f[j])
}

func (f FeeCapQueue) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (p *MpoolQueue) ResetBaseFeeCap(cap big.Int) {
	logrus.Infof("set basefeecap %s", cap.String())
	feecapMin, _ := types.BigFromString(p.conf.Setting.BaseFee)

	if cap.LessThan(feecapMin) {
		cap = feecapMin
	}
	if p.maxFeecap.LessThan(cap) {
		logrus.Warnf("curren fee cap %s more than max-fee-cap", cap.String())
		return
	}
	ResetBaseFeeCap(cap)
	p.feecap = cap
}

func (p *MpoolQueue) queueTop() int {
	// 如果要收集一个小时的数据, 用1小时的秒数除以basefee查询的间隔
	harborTime := time.Hour * 1
	count := int(harborTime / (time.Second * time.Duration(p.conf.Setting.RefreshBaseFee)))
	logrus.Infof("queueTop value is %d", count)
	return count
}

// 添加监控到的basefee数据, 用于后续使用
func (p *MpoolQueue) addFee(fee big.Int) {
	logrus.Infof("add basefee %s to queue", fee.String())
NEXT:
	if len(p.feeQueue) > p.queueTop() {
		p.feeQueue = p.feeQueue[1:]
		goto NEXT
	}
	p.feeQueue = append(p.feeQueue, fee)
}

// 获取下一次的basefee, 根据百分比的配置计算获取当前合适的basefee
func (p *MpoolQueue) getNextFeeCap(percent int, needmax bool) big.Int {
	if len(p.feeQueue) < 10 {
		return p.feecap
	}
	sortcap := FeeCapQueue{}
	sortcap = append(sortcap, p.feeQueue...)
	sort.Sort(sortcap)

	if percent > 100 {
		percent = 100
	}
	if needmax {
		return sortcap[len(sortcap)-1]
	}
	return sortcap[len(sortcap)*percent/100-1]
}

func (p *MpoolQueue) checkMsgTooMore(mpoolTotal int) bool {
	if mpoolTotal < p.conf.Setting.MpoolThresholdLow {
		return false
	}
	// 估算每个小时最大能走多少消息
	maxFree := p.conf.Setting.OnceMax * 120
	// 估算实际每小时能走多少消息
	actualFree := maxFree * p.conf.Setting.BaseFeePercent / 100
	// 最大每小时能多走多少消息
	extraFree := maxFree - actualFree

	if p.conf.Setting.MpoolThresholdHigh > 0 && mpoolTotal > p.conf.Setting.MpoolThresholdHigh {
		return true
	}
	// 消息池总数大于需要疏通的最大时间配额, 就认为消息过多
	if mpoolTotal > extraFree*int(p.conf.Setting.TimeKeep) {
		return true
	}
	return false
}

func (p *MpoolQueue) checkMsgTooLow(mpoolTotal int) bool {
	if mpoolTotal < p.conf.Setting.MpoolThresholdLow {
		return true
	}
	return false
}