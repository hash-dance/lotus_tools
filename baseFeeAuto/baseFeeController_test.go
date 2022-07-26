package baseFeeAuto

import (
	"sort"
	"testing"

	"github.com/filecoin-project/go-state-types/big"
)

func Test_s(t *testing.T) {
	cap := FeeCapQueue{}
	cap = append(cap, big.NewInt(1111111))
	cap = append(cap, big.NewInt(111))
	cap = append(cap, big.NewInt(1223261))
	cap = append(cap, big.NewInt(121))
	cap = append(cap, big.NewInt(222))
	cap = append(cap, big.NewInt(1111111))
	cap = append(cap, big.NewInt(44444))
	sortcap := FeeCapQueue{}
	sortcap = append(sortcap, cap...)
	sort.Sort(sortcap)
	t.Logf("%+v", cap)
	t.Logf("%+v", sortcap)
NEXT:
	if len(cap) > 5 {
		cap = cap[1:]
		goto NEXT
	}
	cap = append(cap, big.NewInt(132313))
	t.Logf("%+v", cap)
	t.Logf("%+v", sortcap)

}