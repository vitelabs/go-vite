package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/viteshan/naive-vite/syncer"
)

type commonFetcher interface {
	fetch(hashHeight commonHashHeight, prevCnt uint64)
}

type accountFetcher struct {
	address types.Address
	fetcher syncer.Fetcher
}

func (self *accountFetcher) fetch(hashHeight commonHashHeight, prevCnt uint64) {
	//self.fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: prevCnt, Chain: self.address})
}

type snapshotFetcher struct {
	fetcher syncer.Fetcher
}

func (self *snapshotFetcher) fetch(hashHeight commonHashHeight, prevCnt uint64) {
	//self.fetcher.Fetch(face.FetchRequest{Hash: hashHeight.Hash, Height: hashHeight.Height, PrevCnt: prevCnt, Chain: self.address})
}
