package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
)

func TestAccountBlock_ComputeHash(t *testing.T) {
	address, _ := types.HexToAddress("vite_4827fbc6827797ac4d9e814affb34b4c5fa85d39bf96d105e7")
	fromHash, _ := types.HexToHash("23a3ea450176ea92e4d7c0943db364bffd305b2befea4ee2d36725f9831f5b4d")
	prevHash, _ := types.HexToHash("2961c93f11bb9b315326f34dc28c5a73b15d142a566e160ca84b57769f350ee8")
	snapshotTimestamp, _ := types.HexToHash("ce36d4b39fcf713880e0299927e7b343a93508de2030b2085171444fa5c543b3")

	block := &AccountBlock{
		Meta: &AccountBlockMeta{
			Height: big.NewInt(2),
		},
		AccountAddress:    &address,
		FromHash:          &fromHash,
		PrevHash:          &prevHash,
		Timestamp:         uint64(1537271245),
		TokenId:           &MockViteTokenId,
		Data:              "",
		SnapshotTimestamp: &snapshotTimestamp,
		Nounce:            []byte{0, 0, 0, 0, 0},
		Difficulty:        []byte{0, 0, 0, 0, 0},
		FAmount:           big.NewInt(0),
	}

	hash, _ := block.ComputeHash()
	if hash.String() != "2455cc5e710f3ad2ee1dda716ad78e44443d8bcfb12fa456f1cc09e1b2e5c56f" {
		t.Log(hash.String())
		t.Fatal("ComputeHash error !!!")
	}
}

//func TestUint64ToBytes(t *testing.T) {
//	j := 0
//	year := 0
//	for i := uint64(969017386); i <= 32525926186; i++ {
//		if j%(60*60*24*365) == 0 {
//			year++
//			fmt.Println("第" + strconv.Itoa(year) + "年")
//		}
//		if !bytes.Equal([]byte(string(i)), []byte{239, 191, 189}) {
//			t.Fatal(i)
//		}
//		j++
//	}
//}
