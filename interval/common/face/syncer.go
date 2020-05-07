package face

import "strconv"

type FetchRequest struct {
	Chain   string
	Height  uint64
	Hash    string
	PrevCnt uint64
}

func (req *FetchRequest) String() string {
	return req.Chain + "," + strconv.FormatUint(req.Height, 10) + "," + req.Hash + "," + strconv.FormatUint(req.PrevCnt, 10)
}
