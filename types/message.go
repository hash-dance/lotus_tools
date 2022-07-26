package types

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/exitcode"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	actorsv5 "github.com/filecoin-project/specs-actors/v5/actors/builtin"
	"github.com/guowenshuai/lotus_tool/stmgr"
	"github.com/ipfs/go-cid"
	"github.com/sirupsen/logrus"
)

type InvocResult struct {
	MsgCid   string `json:"msg_cid,omitempty"`
	Message  `bson:",inline"`
	MsgRct   *MessageReceipt `json:"msg_rct,omitempty"`
	Error    string          `json:"error,omitempty"`
	Duration time.Duration   `json:"duration,omitempty"`

	BaseFee    string `json:"base_fee,omitempty"`
	MsgGasCost `bson:",inline"`

	Height         abi.ChainEpoch `json:"height"`
	Time           time.Time      `json:"time"`
	ExecutionTrace ExecutionTrace `json:"executionTrace,omitempty"`
}

type MsgGasCost struct {
	GasUsed            string `json:"gas_used,omitempty"`
	BaseFeeBurn        string `json:"base_fee_burn,omitempty"`
	OverEstimationBurn string `json:"over_estimation_burn,omitempty"` // 燃烧惩罚
	MinerTip           string `json:"miner_tip,omitempty"`            // 矿工手续费收益
	TotalCost          string `json:"total_cost,omitempty"`           // 消息总花费
	Refund             string `json:"refund,omitempty"`               // 返还费用
	MinerPenalty       string `json:"miner_penalty,omitempty"`        // 矿工惩罚
}

type Message struct {
	Version uint64 `json:"version,omitempty"`

	To   string `json:"to,omitempty"`
	From string `json:"from,omitempty"`

	Nonce uint64 `json:"nonce,omitempty"`

	Value string `json:"value,omitempty"`

	GasLimit   int64  `json:"gas_limit,omitempty"`
	GasFeeCap  string `json:"gas_fee_cap,omitempty"`
	GasPremium string `json:"gas_premium,omitempty"`

	Method        abi.MethodNum    `json:"method,omitempty"`
	Params        string           `json:"params,omitempty"`
	SectorNumber  abi.SectorNumber `json:"sector_number,omitempty"` // 扇区号
	SealRandEpoch abi.ChainEpoch   `json:"seal_rand_epoch,omitempty"`
	// v1.10.0 对打包消息的处理, 一个打包消息里面有多少扇区
	SectorLength int `json:"sector_length,omitempty"`

	Cid       string `json:"cid,omitempty"`
	MethodStr string `json:"method_str"`
	CodeStr   string `json:"code_str"`
}

type MessageReceipt struct {
	ExitCode exitcode.ExitCode
	Return   string
	GasUsed  int64
}

type ExecutionTrace struct {
	Msg      *Message         `json:"msg,omitempty"`
	MsgRct   *MessageReceipt  `json:"msg_rct,omitempty"`
	Error    string           `json:"error,omitempty"`
	Duration time.Duration    `json:"duration,omitempty"`
	Subcalls []ExecutionTrace `json:"subcalls,omitempty"`
}

func LotusMessageReceiptConvert(ctx context.Context, nodeapi api.FullNode, code cid.Cid, method abi.MethodNum, msg *types.MessageReceipt) (msgRct *MessageReceipt) {
	if msg == nil {
		return nil
	}
	defer func() {
		if err := recover(); err != nil {
			msgRct = nil
		}
	}()
	var ret string

	if retParam, err := stmgr.JsonReturn(code, method, msg.Return); err == nil {
		ret = retParam
	}
	msgRct.ExitCode = msg.ExitCode
	msgRct.Return = ret
	msgRct.GasUsed = msg.GasUsed
	return
}

func LotusMessageConvert(ctx context.Context, nodeapi api.FullNode, code cid.Cid, msg *types.Message) *Message {
	if msg == nil {
		logrus.Warn("msg null")
		return nil
	}

	var pa string
	if param, err := stmgr.ParseParam(ctx, nodeapi, code, msg); err == nil {
		pa = param
	} else {
		logrus.Errorf("ParseParam %s", err.Error())
	}
	logrus.Debugf("LotusMessageConvert param %s", pa)
	sectorNumber := getSectorNumberFromParam(msg.Method, pa)
	sealRandEpoch := getSealRandEpochFromParam(msg.Method, pa)
	sectorLength := getSectorLength(msg.Method, pa)
	// sectorLength := 100

	return &Message{
		Version:       msg.Version,
		To:            msg.To.String(),
		From:          msg.From.String(),
		Nonce:         msg.Nonce,
		Value:         msg.Value.String(),
		GasLimit:      msg.GasLimit,
		GasFeeCap:     msg.GasFeeCap.String(),
		GasPremium:    msg.GasPremium.String(),
		Method:        msg.Method,
		Params:        pa,
		SectorNumber:  sectorNumber,
		SectorLength:  sectorLength,
		SealRandEpoch: sealRandEpoch,
		Cid:           msg.Cid().String(),
		MethodStr:     stmgr.GetMethod(code, msg.Method),
		CodeStr:       stmgr.CodeStr(code),
	}
}

func LotusExecutionTraceConvert(ctx context.Context, nodeapi api.FullNode,
	im types.ExecutionTrace, getCode func(addr address.Address) (cid.Cid, error)) ExecutionTrace {
	lock := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	subCalls := make([]ExecutionTrace, 0)
	subCallsAppend := func(d ExecutionTrace) {
		lock.Lock()
		subCalls = append(subCalls, d)
		lock.Unlock()
	}

	for _, tc := range im.Subcalls {
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, nextTc types.ExecutionTrace) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return // returning not to leak the goroutine
				default:
					subCallsAppend(LotusExecutionTraceConvert(ctx, nodeapi, nextTc, getCode))
					return
				}
			}
		}(ctx, wg, tc)
	}

	wg.Wait()
	code, _ := getCode(im.Msg.To)
	// if err != nil {
	// 	logrus.Infof("%s %s %s", im.Msg.Cid(), im.Msg.To, err.Error())
	// 	panic("x")
	// }

	exTc := ExecutionTrace{
		Msg:      LotusMessageConvert(ctx, nodeapi, code, im.Msg),
		MsgRct:   LotusMessageReceiptConvert(ctx, nodeapi, code, im.Msg.Method, im.MsgRct),
		Error:    im.Error,
		Duration: im.Duration,
		Subcalls: subCalls,
	}
	return exTc
}

func LotusInvocResultConvert(ctx context.Context, nodeapi api.FullNode, baseFee string,
	invoc *api.InvocResult, height abi.ChainEpoch, timestamp uint64, getCode func(addr address.Address) (cid.Cid, error)) *InvocResult {
	code, _ := getCode(invoc.Msg.To)
	// if err != nil {
	// 	return nil
	// }

	return &InvocResult{
		MsgCid:   invoc.MsgCid.String(),
		Message:  *LotusMessageConvert(ctx, nodeapi, code, invoc.Msg),
		MsgRct:   LotusMessageReceiptConvert(ctx, nodeapi, code, invoc.Msg.Method, invoc.MsgRct),
		Error:    invoc.Error,
		Duration: invoc.Duration,
		BaseFee:  baseFee,
		Height:   height,
		Time:     time.Unix(int64(timestamp), 0),
		MsgGasCost: MsgGasCost{
			GasUsed:            invoc.GasCost.GasUsed.String(),
			BaseFeeBurn:        invoc.GasCost.BaseFeeBurn.String(),
			OverEstimationBurn: invoc.GasCost.OverEstimationBurn.String(),
			MinerTip:           invoc.GasCost.MinerTip.String(),
			TotalCost:          invoc.GasCost.TotalCost.String(),
			Refund:             invoc.GasCost.Refund.String(),
			MinerPenalty:       invoc.GasCost.MinerPenalty.String(),
		},
		ExecutionTrace: LotusExecutionTraceConvert(ctx, nodeapi, invoc.ExecutionTrace, getCode),
	}
}

func getSectorNumberFromParam(method abi.MethodNum, param string) abi.SectorNumber {
	number := abi.SectorNumber(0)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(param), &result); err != nil {
		logrus.Debugf("getSectorNumberFromParam %s ", err.Error())
		return number
	}
	if method == builtin.MethodsMiner.PreCommitSector || method == builtin.MethodsMiner.ProveCommitSector {
		if num, ok := result["SectorNumber"]; ok {
			number = abi.SectorNumber(num.(float64))
		}
	}
	return number
}

func getSealRandEpochFromParam(method abi.MethodNum, param string) abi.ChainEpoch {
	epoch := abi.ChainEpoch(0)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(param), &result); err != nil {
		return epoch
	}
	if method == builtin.MethodsMiner.PreCommitSector {
		if num, ok := result["SealRandEpoch"]; ok {
			epoch = abi.ChainEpoch(num.(float64))
		}
	}
	return epoch
}

func getSectorLength(method abi.MethodNum, param string) int {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(param), &result); err != nil {
		return 0
	}
	if method == actorsv5.MethodsMiner.PreCommitSectorBatch {
		if arr, ok := result["Sectors"]; ok {
			if vl, ok2 := arr.([]interface{}); ok2 {
				return len(vl)
			}
		}
	}
	if method == actorsv5.MethodsMiner.ProveCommitAggregate {
		sum := float64(0)
		if arr, ok := result["SectorNumbers"]; ok {
			if vl, ok2 := arr.([]interface{}); ok2 {
				for i, v := range vl {
					if i%2 == 1 {
						sum += v.(float64)
					}
				}
				return int(sum)
			}
		}
	}
	return 0

}

// func GetCode(ctx context.Context, addr address.Address, nodeapi api.FullNode, tsk types.TipSetKey) (cid.Cid, error) {
// 	cacheLock.Lock()
// 	defer cacheLock.Unlock()
// 	if c, found := codeCache[addr]; found {
// 		return c, nil
// 	}
// 	act, err := nodeapi.StateGetActor(ctx, addr, tsk)
// 	if err != nil {
// 		return cid.Cid{}, err
// 	}
//
// 	codeCache[addr] = act.Code
// 	return act.Code, nil
// }
