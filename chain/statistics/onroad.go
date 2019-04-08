package statistics

import (
	"github.com/golang/protobuf/proto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
)

type onroadMeta struct {
	TotalAmount big.Int
	Number      uint64
}

func (om *onroadMeta) proto() *vitepb.OnroadMeta {
	pb := &vitepb.OnroadMeta{}
	pb.Num = om.Number
	pb.Amount = om.TotalAmount.Bytes()
	return pb
}

func (om *onroadMeta) deProto(pb *vitepb.OnroadMeta) {
	om.Number = pb.Num
	totalAmount := big.NewInt(0)
	if len(pb.Amount) > 0 {
		totalAmount.SetBytes(pb.Amount)
	}
	om.TotalAmount = *totalAmount
}

func (om *onroadMeta) serialize() ([]byte, error) {
	return proto.Marshal(om.proto())
}

func (om *onroadMeta) deserialize(buf []byte) error {
	pb := &vitepb.OnroadMeta{}
	if err := proto.Unmarshal(buf, pb); err != nil {
		return err
	}
	om.deProto(pb)
	return nil
}

func (sDB *StatisticsDB) getOnroadInfo(addr types.Address) (map[types.TokenTypeId]*onroadMeta, error) {
	omMap := make(map[types.TokenTypeId]*onroadMeta)
	iter := sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateBalanceKeyPrefix(addr)), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		tokenTypeIdBytes := key[1+types.AddressSize : 1+types.AddressSize+types.TokenTypeIdSize]
		tokenTypeId, err := types.BytesToTokenTypeId(tokenTypeIdBytes)
		if err != nil {
			return nil, err
		}
		om := &onroadMeta{}
		om.deserialize(iter.Value())
		omMap[tokenTypeId] = om
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return omMap, nil
}

func (sDB *StatisticsDB) getOnRoadMeta(addr *types.Address, tId *types.TokenTypeId) (*onroadMeta, error) {
	value, err := sDB.db.Get(chain_utils.CreateOnRoadInfoKey(addr, tId), nil)
	if err != nil {
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}
	om := &onroadMeta{}
	om.deserialize(value)
	return om, nil
}

func (sDB *StatisticsDB) writeOnroadInfo(batch *leveldb.Batch, addr *types.Address, tId *types.TokenTypeId, amount *big.Int) error {
	om, err := sDB.getOnRoadMeta(addr, tId)
	if err != nil {
		return err
	}
	var dataSlice []byte
	var sErr error
	if om != nil {
		om.Number += 1
		om.TotalAmount.Add(&om.TotalAmount, amount)
		dataSlice, sErr = om.serialize()
		if sErr != nil {
			return sErr
		}
	} else {
		om_new := &onroadMeta{
			TotalAmount: *amount,
			Number:      1,
		}
		dataSlice, sErr = om_new.serialize()
		if sErr != nil {
			return sErr
		}
	}
	key := chain_utils.CreateOnRoadInfoKey(addr, tId)
	batch.Put(key, dataSlice)
	return nil
}

func (sDB *StatisticsDB) deleteOnroadInfo(batch *leveldb.Batch, addr *types.Address, tId *types.TokenTypeId) {
	key := chain_utils.CreateOnRoadInfoKey(addr, tId)
	batch.Delete(key)
}
