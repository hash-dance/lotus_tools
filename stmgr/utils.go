package stmgr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/rt"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	exported0 "github.com/filecoin-project/specs-actors/actors/builtin/exported"
	exported2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/exported"
	exported3 "github.com/filecoin-project/specs-actors/v3/actors/builtin/exported"
	exported4 "github.com/filecoin-project/specs-actors/v4/actors/builtin/exported"
	exported5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/exported"
	exported6 "github.com/filecoin-project/specs-actors/v6/actors/builtin/exported"
	exported7 "github.com/filecoin-project/specs-actors/v7/actors/builtin/exported"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/sirupsen/logrus"
	cbg "github.com/whyrusleeping/cbor-gen"
)

type MethodMeta struct {
	Name string

	Params reflect.Type
	Ret    reflect.Type
}

var MethodsMap = map[cid.Cid]map[abi.MethodNum]MethodMeta{}

func init() {
	// TODO: combine with the runtime actor registry.
	var actors []rt.VMActor
	actors = append(actors, exported0.BuiltinActors()...)
	actors = append(actors, exported2.BuiltinActors()...)
	actors = append(actors, exported3.BuiltinActors()...)
	actors = append(actors, exported4.BuiltinActors()...)
	actors = append(actors, exported5.BuiltinActors()...)
	actors = append(actors, exported6.BuiltinActors()...)
	actors = append(actors, exported7.BuiltinActors()...)

	for _, actor := range actors {
		exports := actor.Exports()
		methods := make(map[abi.MethodNum]MethodMeta, len(exports))

		// Explicitly add send, it's special.
		methods[builtin.MethodSend] = MethodMeta{
			Name:   "Send",
			Params: reflect.TypeOf(new(abi.EmptyValue)),
			Ret:    reflect.TypeOf(new(abi.EmptyValue)),
		}

		// Iterate over exported methods. Some of these _may_ be nil and
		// must be skipped.
		for number, export := range exports {
			if export == nil {
				continue
			}

			ev := reflect.ValueOf(export)
			et := ev.Type()

			// Extract the method names using reflection. These
			// method names always match the field names in the
			// `builtin.Method*` structs (tested in the specs-actors
			// tests).
			fnName := runtime.FuncForPC(ev.Pointer()).Name()
			fnName = strings.TrimSuffix(fnName[strings.LastIndexByte(fnName, '.')+1:], "-fm")

			switch abi.MethodNum(number) {
			case builtin.MethodSend:
				panic("method 0 is reserved for Send")
			case builtin.MethodConstructor:
				if fnName != "Constructor" {
					panic("method 1 is reserved for Constructor")
				}
			}

			methods[abi.MethodNum(number)] = MethodMeta{
				Name:   fnName,
				Params: et.In(1),
				Ret:    et.Out(0),
			}
		}
		MethodsMap[actor.Code()] = methods
	}
}

func CodeStr(c cid.Cid) string {
	cmh, err := multihash.Decode(c.Hash())
	if err != nil {
		logrus.Warnf("CodeStr %s", err.Error())
		return "NA"
	}
	return string(cmh.Digest)
}

// 根据角色和方法获取方法名
func GetMethod(code cid.Cid, method abi.MethodNum) string {
	return MethodsMap[code][method].Name
}

func GetParamType(actCode cid.Cid, method abi.MethodNum) (cbg.CBORUnmarshaler, error) {
	m, found := MethodsMap[actCode][method]
	if !found {
		return nil, fmt.Errorf("unknown method %d for actor %s", method, actCode)
	}
	return reflect.New(m.Params.Elem()).Interface().(cbg.CBORUnmarshaler), nil
}

func JsonParams(code cid.Cid, method abi.MethodNum, params []byte) (string, error) {
	p, err := GetParamType(code, method)
	if err != nil {
		return "", err
	}

	if err := p.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		return "", err
	}

	b, _ := json.Marshal(p)
	return string(b), nil
}

func JsonReturn(code cid.Cid, method abi.MethodNum, ret []byte) (string, error) {
	methodMeta, found := MethodsMap[code][method]
	if !found {
		return "", fmt.Errorf("method %d not found on actor %s", method, code)
	}
	re := reflect.New(methodMeta.Ret.Elem())
	p := re.Interface().(cbg.CBORUnmarshaler)
	if err := p.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
		return "", fmt.Errorf("unmarshalCBOR %s", err.Error())
	}

	b, err := json.MarshalIndent(p, "", "  ")
	return string(b), err
}

func ParseParam(ctx context.Context, nodeapi api.FullNode, code cid.Cid, message interface{}) (string, error) {
	var msg *types.Message
	switch message.(type) {
	case *types.Message:
		msg = message.(*types.Message)
	case *types.SignedMessage:
		msg = &(message.(*types.SignedMessage)).Message
	default:
		logrus.Error("error message type")
		return "", errors.New("error message type")
	}
	return JsonParams(code, msg.Method, msg.Params)

}
