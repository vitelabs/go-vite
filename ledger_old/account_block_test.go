package ledger

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"testing"
)

func TestAccountBlock_ComputeHash(t *testing.T) {
	address, _ := types.HexToAddress("vite_85ffbcd9fcd341838811fd96aae5f0b02e0ae141b4e70fcecb")
	fromHash, _ := types.HexToHash("2415413b0c4c73d2ff7c23be16a74cbc89a94f34b97bdf7db79de841a21d8033")
	prevHash, _ := types.HexToHash("6afb1e1bbe26a5ddc7aa658784f1eb58a278fbd31b06c3e6974a8cbf08e79073")
	snapshotTimestamp, _ := types.HexToHash("52893cb69d6aa6fd1938692cfb91c93a8602ffc41f518fc08444380ae51f6a56")

	block := &AccountBlock{
		Meta: &AccountBlockMeta{
			Height: big.NewInt(100),
		},
		AccountAddress:    &address,
		FromHash:          &fromHash,
		PrevHash:          &prevHash,
		Timestamp:         uint64(1537009137),
		TokenId:           &MockViteTokenId,
		Data:              "12345",
		SnapshotTimestamp: &snapshotTimestamp,
		Nounce:            []byte{0, 0, 0, 0, 0},
		Difficulty:        []byte{0, 0, 0, 0, 0},
		FAmount:           big.NewInt(0),
	}

	hash, _ := block.ComputeHash()
	if hash.String() != "2455cc5e710f3ad2ee1dda716ad78e44443d8bcfb12fa456f1cc09e1b2e5c56f" {
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
