package dex

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

//periodId is finish period
func DoFinishVxUnlock(db vm_db.VmDb, periodId uint64) error {
	if !IsEarthFork(db) {
		return nil
	}
	iterator, err := db.NewStorageIterator(vxUnlocksKeyPrefix)
	if err != nil {
		panic(err)
	}
	defer iterator.Release()
	for {
		var vxUnlocksKey, vxUnlocksValue []byte
		if !iterator.Next() {
			if iterator.Error() != nil {
				panic(iterator.Error())
			}
			break
		}
		vxUnlocksKey = iterator.Key() //5+20
		vxUnlocksValue = iterator.Value()
		if len(vxUnlocksValue) == 0 {
			continue
		}
		if len(vxUnlocksKey) != len(vxUnlocksKeyPrefix) + types.AddressSize {
			panic(fmt.Errorf("invalid vx unlocks key"))
		}
		address, _ := types.BytesToAddress(vxUnlocksKey[len(vxUnlocksKeyPrefix):])
		unlocks := &VxUnlocks{}
		if err := unlocks.DeSerialize(vxUnlocksValue); err != nil {
			panic(err)
		}
		var i = 0
		var amount = new(big.Int)
		for _, ul := range unlocks.Unlocks {
			if ul.PeriodId+uint64(VxUnlockScheduleDays) <= periodId {
				amount.Add(amount, new(big.Int).SetBytes(ul.Amount))
				i++
			} else {
				break
			}
		}
		if i > 0 {
			unlocks.Unlocks = unlocks.Unlocks[i:]
			UpdateVxUnlocks(db, address, unlocks)
			if _, err = FinishVxUnlockForDividend(db, address, amount); err != nil {
				return err
			}
		}
	}
	return nil
}
