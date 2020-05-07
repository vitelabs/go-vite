package syncer

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/p2p"
	"github.com/vitelabs/go-vite/interval/test"
)

type senderTest struct {
	store   map[string]common.Block
	fetcher *fetcher
	times   int32
}

func (self *senderTest) BroadcastAccountBlocks(address string, blocks []*common.AccountStateBlock) error {
	panic("implement BroadcastAccountBlocks")
}

func (self *senderTest) BroadcastSnapshotBlocks(blocks []*common.SnapshotBlock) error {
	panic("implement BroadcastSnapshotBlocks")
}

func (self *senderTest) SendAccountBlocks(address string, blocks []*common.AccountStateBlock, peer p2p.Peer) error {
	panic("implement SendAccountBlocks")
}

func (self *senderTest) SendSnapshotBlocks(blocks []*common.SnapshotBlock, peer p2p.Peer) error {
	panic("implement SendSnapshotBlocks")
}

func (self *senderTest) SendAccountHashes(address string, hashes []common.HashHeight, peer p2p.Peer) error {
	panic("implement SendAccountHashes")
}

func (self *senderTest) SendSnapshotHashes(hashes []common.HashHeight, peer p2p.Peer) error {
	panic("implement SendSnapshotHashes")
}

func (self *senderTest) RequestAccountHash(address string, height common.HashHeight, prevCnt int) error {
	go func() {
		var hashes []common.HashHeight
		height := height.Height
		for i := 1; i < prevCnt+1; i++ {
			tmpH := height - i
			hashes = append(hashes, common.HashHeight{Height: tmpH, Hash: genHashByHeight(tmpH)})
		}
		if len(hashes) > 0 {
			self.fetcher.fetchAccountBlockByHash(address, hashes)
		}
	}()
	return nil
}

func (self *senderTest) RequestSnapshotHash(height common.HashHeight, prevCnt int) error {
	panic("implement RequestSnapshotHash")
}

func (self *senderTest) RequestAccountBlocks(address string, hashes []common.HashHeight) error {
	go func() {
		for _, v := range hashes {
			block, ok := self.store[v.Hash]
			if ok {
				fmt.Printf("receive block: %v\n", block)
				atomic.CompareAndSwapInt32(&self.times, self.times, self.times+1)
			}
		}
	}()

	return nil
}
func (self *senderTest) RequestSnapshotBlocks(hashes []common.HashHeight) error {
	panic("implement RequestSnapshotBlocks")
}

func (self *senderTest) handle(blocks []common.Block) {
	for _, block := range blocks {
		fmt.Printf("receive block: %v\n", block)
		atomic.CompareAndSwapInt32(&self.times, self.times, self.times+1)
	}
}

func TestFetcher(t *testing.T) {
	//address := "viteshan"
	//N := 10
	//sender := &senderTest{store: genBlockStore(N)}
	//
	//fetcher := &fetcher{sender: sender, retryPolicy: &defaultRetryPolicy{fetchedHashs: make(map[string]*RetryStatus)}}
	//sender.fetcher = fetcher
	//
	//fetcher.fetchAccountBlockByHash(address, genFetchHash(N))
	//fetcher.fetchAccountBlockByHash(address, genFetchHash(N+10))
	//fetcher.fetchAccountBlockByHash(address, genFetchHash(N+5))
	//fetcher.fetchAccountBlockByHash(address, genFetchHash(N*2))
	//fetcher.FetchAccount(address, common.HashHeight{Height: N, Hash: genHashByHeight(N)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N + 5, Hash: genHashByHeight(N + 5)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 2, Hash: genHashByHeight(N * 2)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 3, Hash: genHashByHeight(N * 3)}, N)
	//
	//fetcher.FetchAccount(address, common.HashHeight{Height: N, Hash: genHashByHeight(N)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N + 5, Hash: genHashByHeight(N + 5)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 2, Hash: genHashByHeight(N * 2)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 3, Hash: genHashByHeight(N * 3)}, N)
	//
	//fetcher.FetchAccount(address, common.HashHeight{Height: N, Hash: genHashByHeight(N)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N + 5, Hash: genHashByHeight(N + 5)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 2, Hash: genHashByHeight(N * 2)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 3, Hash: genHashByHeight(N * 3)}, N)
	//
	//fetcher.FetchAccount(address, common.HashHeight{Height: N, Hash: genHashByHeight(N)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N + 5, Hash: genHashByHeight(N + 5)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 2, Hash: genHashByHeight(N * 2)}, N)
	//fetcher.FetchAccount(address, common.HashHeight{Height: N * 3, Hash: genHashByHeight(N * 3)}, N)
	//
	//time.Sleep(2 * time.Second)
	//if N != int(sender.times) {
	//	t.Errorf("error result. expect:%d, actual:%d", N, sender.times)
	//}
}
func genFetchHash(N int) []common.HashHeight {
	var hashes []common.HashHeight
	for i := 0; i < N; i++ {
		hashes = append(hashes, common.HashHeight{Height: N, Hash: genHashByHeight(N)})
	}
	return hashes
}

func genBlockStore(N int) map[string]common.Block {
	hashes := make(map[string]common.Block)
	for i := 0; i < N; i++ {
		height := N + i
		hashes[genHashByHeight(height)] = &test.TestBlock{Theight: height, Thash: genHashByHeight(height), TpreHash: genPrevHashByHeight(height)}
	}
	return hashes
}

func genHashByHeight(height int) string {
	return strconv.Itoa(height - 10)
}

func genPrevHashByHeight(height int) string {
	return strconv.Itoa(height - 11)
}
