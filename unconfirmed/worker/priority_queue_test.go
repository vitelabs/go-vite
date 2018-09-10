package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

const TO_COUNT = 5

func MakeDate() []*ledger.AccountBlock {
	var blockList []*ledger.AccountBlock
	var addrList []*types.Address
	for i := 0; i < TO_COUNT; i++ {
		addr, _, _ := types.CreateAddress()
		addrList = append(addrList, &addr)
	}
	for _, v := range addrList {
		height := big.NewInt(int64(time.Now().UnixNano()))
		h := height
		for i := 0; i < TO_COUNT; i++ {
			bal := make(map[types.TokenTypeId]*big.Int)
			bal[Vite_TokenId] = big.NewInt(1)

			block := &ledger.AccountBlock{
				Meta:              nil,
				BlockType:         0,
				Hash:              nil,
				Height:            h,
				PrevHash:          nil,
				AccountAddress:    v,
				PublicKey:         nil,
				ToAddress:         addrList[rand.Intn(TO_COUNT)],
				FromBlockHash:     nil,
				Amount:            nil,
				TokenId:           nil,
				QuotaFee:          nil,
				ContractFee:       nil,
				SnapshotHash:      nil,
				Data:              "",
				Timestamp:         0,
				StateHash:         nil,
				LogHash:           nil,
				Nonce:             nil,
				SendBlockHashList: nil,
				Signature:         nil,
			}
			blockList = append(blockList, block)
			h = height.Add(h, big.NewInt(1))
		}
	}
	return blockList
}

func Example_priorityQueue() {

	// heap.Init(&pq)
	// heap.Push(&pq, item)
	// heap.Pop(&pq, item)

}

func TestPriorityFromQueue_InsertNew(t *testing.T) {
	blockList := MakeDate()
	t.Log(blockList)
	var priorityFromQueue *PriorityFromQueue
	t.Log("priorityFromQueue start to insert blocks")
	for _, v := range blockList {
		priorityFromQueue.InsertNew(v)
	}
	t.Log("priorityFromQueue start to insert blocks")
	t.Log(priorityFromQueue)
}
