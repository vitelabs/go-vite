package face

import "strconv"

type FetchRequest struct {
	Chain   string
	Height  uint64
	Hash    string
	PrevCnt uint64
}

func (self *FetchRequest) String() string {
	return self.Chain + "," + strconv.FormatUint(self.Height, 10) + "," + self.Hash + "," + strconv.FormatUint(self.PrevCnt, 10)
}
