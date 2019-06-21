package ledger

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"testing"
)

func TestContractMeta_Serialize(t *testing.T) {

	hashtmp, err := types.HexToHash("db7a03be03372c499ed6a211f5b5d8ba5fd6469869f6e4415b2e58e5dd321636")

	if err != nil {
		t.Fatal(err)
	}

	cm := &ContractMeta{
		Gid:                types.DELEGATE_GID,
		SendConfirmedTimes: 1,
		CreateBlockHash:    hashtmp, // hash of send create block for creating the contract
		QuotaRatio:         10,      // the ratio of quota cost for the send block
		SeedConfirmedTimes: 2,
	}

	//[0 0 0 0 0 0 0 0 0 2 1 219 122 3 190 3 55 44 73 158 214 162 17 245 181 216 186 95 214 70 152 105 246 228 65 91 46 88 229 221 50 22 54 10 1]
	byteBuf := cm.Serialize()

	if len(byteBuf) != LengthBeforeSeedFork+1 {
		t.Fatal(fmt.Sprintf("must be equal %d, but is  %d", LengthBeforeSeedFork+1, len(byteBuf)))
	}

	// case 1
	cmNew1 := *cm
	if err := checkSame(cmNew1, byteBuf); err != nil {
		t.Fatal(err)
	}

	// case 2
	cmNew2 := *cm
	cmNew2.SeedConfirmedTimes = cmNew2.SendConfirmedTimes

	if err := checkSame(cmNew2, cm.Serialize()[:LengthBeforeSeedFork]); err != nil {
		t.Fatal(err)
	}

	// case 3
	cm2 := &ContractMeta{
		Gid:                types.DELEGATE_GID,
		SendConfirmedTimes: 1,
		CreateBlockHash:    hashtmp, // hash of send create block for creating the contract
		QuotaRatio:         10,      // the ratio of quota cost for the send block
		SeedConfirmedTimes: 0,
	}

	// [0 0 0 0 0 0 0 0 0 2 1 219 122 3 190 3 55 44 73 158 214 162 17 245 181 216 186 95 214 70 152 105 246 228 65 91 46 88 229 221 50 22 54 10 0]
	byteBuf2 := cm2.Serialize()
	if len(byteBuf2) != LengthBeforeSeedFork+1 {
		t.Fatal(fmt.Sprintf("must be equal %d, but is  %d", LengthBeforeSeedFork+1, len(byteBuf2)))
	}

	if err := checkSame(*cm2, byteBuf2); err != nil {
		t.Fatal(err)
	}

}

func checkSame(cm ContractMeta, byteBuf []byte) error {
	//byteNew := make([]byte, len(byteBuf)-1)

	//copy(byteNew, byteBuf[:len(byteBuf)-1])

	cmNew := &ContractMeta{}
	if err := cmNew.Deserialize(byteBuf); err != nil {
		return err
	}

	if cm.Gid != cmNew.Gid ||
		cm.CreateBlockHash != cmNew.CreateBlockHash ||
		cm.QuotaRatio != cmNew.QuotaRatio ||
		cm.SendConfirmedTimes != cmNew.SendConfirmedTimes ||
		cm.SeedConfirmedTimes != cmNew.SeedConfirmedTimes {
		return errors.New(fmt.Sprintf("cm is %+v, cmnew is %+v", cm, cmNew))
	}

	return nil

}
