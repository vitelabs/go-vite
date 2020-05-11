package face

import (
	"strconv"

	"github.com/vitelabs/go-vite/interval/common"
)

type FetchRequest struct {
	Chain   *common.Address
	Height  common.Height
	Hash    common.Hash
	PrevCnt uint64
}

func (req *FetchRequest) String() string {
	if req.Chain != nil {
		return req.Chain.String() + "," + req.Height.String() + "," + req.Hash.String() + "," + strconv.FormatUint(req.PrevCnt, 10)
	} else {
		return req.Height.String() + "," + req.Hash.String() + "," + strconv.FormatUint(req.PrevCnt, 10)
	}
}
