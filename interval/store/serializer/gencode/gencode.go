package gencode

import (
	"time"

	"github.com/vitelabs/go-vite/interval/common"
)

func SerializeSnapshotBlock(block *common.SnapshotBlock) ([]byte, error) {
	dbBlock := &DBSnapshotBlock{
		Height:    block.Height(),
		PreHash:   block.PreHash(),
		Hash:      block.Hash(),
		Timestamp: block.Timestamp().UnixNano(),
		Accounts:  serializeAccounts(block.Accounts),
	}
	r, e := dbBlock.Marshal(nil)
	return r, e
}

func DeserializeSnapshotBlock(bs []byte) (*common.SnapshotBlock, error) {
	dbBlock := &DBSnapshotBlock{}
	_, e := dbBlock.Unmarshal(bs)
	if e != nil {
		return nil, e
	}

	rblock := common.NewSnapshotBlock(dbBlock.Height, dbBlock.Hash, dbBlock.PreHash, dbBlock.Signer, time.Unix(0, dbBlock.Timestamp), deserializeAccounts(dbBlock.Accounts))
	return rblock, nil
}

func SerializeAccountBlock(block *common.AccountStateBlock) ([]byte, error) {
	dbBlock := &DBAccountBlock{
		Height:         block.Height(),
		PreHash:        block.PreHash(),
		Hash:           block.Hash(),
		Signer:         block.Signer(),
		Timestamp:      block.Timestamp().UnixNano(),
		Amount:         int64(block.Amount),
		ModifiedAmount: int64(block.ModifiedAmount),
		BlockType:      uint8(block.BlockType),
		From:           block.From,
		To:             block.To,
		Source:         serializeHashHeight(block.Source),
	}
	r, e := dbBlock.Marshal(nil)
	return r, e
}

func DeserializeAccountBlock(bs []byte) (*common.AccountStateBlock, error) {
	dbBlock := &DBAccountBlock{}
	_, e := dbBlock.Unmarshal(bs)
	if e != nil {
		return nil, e
	}

	rblock := common.NewAccountBlock(dbBlock.Height, dbBlock.Hash, dbBlock.PreHash, dbBlock.Signer, time.Unix(0, dbBlock.Timestamp),
		int(dbBlock.Amount), int(dbBlock.ModifiedAmount), common.BlockType(dbBlock.BlockType), dbBlock.From, dbBlock.To, deserializeHashHeight(dbBlock.Source))
	return rblock, nil
}

func serializeHashHeight(hashHeight *common.HashHeight) *HashHeight {
	if hashHeight == nil {
		return nil
	}
	return &HashHeight{Hash: hashHeight.Hash, Height: hashHeight.Height}
}
func deserializeHashHeight(hashHeight *HashHeight) *common.HashHeight {
	if hashHeight == nil {
		return nil
	}
	return &common.HashHeight{Hash: hashHeight.Hash, Height: hashHeight.Height}
}

func SerializeHashHeight(hashHeight *common.HashHeight) ([]byte, error) {
	return serializeHashHeight(hashHeight).Marshal(nil)
}
func DeserializeHashHeight(byt []byte) (*common.HashHeight, error) {
	dbHashH := &HashHeight{}
	_, e := dbHashH.Unmarshal(byt)
	if e != nil {
		return nil, e
	}
	return deserializeHashHeight(dbHashH), nil
}

func SerializeAccountHashH(accountH *common.AccountHashH) ([]byte, error) {
	h := &AccountHashH{Height: accountH.Height, Hash: accountH.Hash, Addr: accountH.Addr}
	byt, e := h.Marshal(nil)
	return byt, e
}
func DeserializeAccountHashH(byt []byte) (*common.AccountHashH, error) {
	accountH := &AccountHashH{}
	_, e := accountH.Unmarshal(byt)
	if e != nil {
		return nil, e
	}
	h := common.NewAccountHashH(accountH.Addr, accountH.Hash, accountH.Height)
	return h, nil
}

func serializeAccounts(hs []*common.AccountHashH) []*AccountHashH {
	i := len(hs)
	if hs == nil || i == 0 {
		return nil
	}
	var accs []*AccountHashH
	for _, v := range hs {
		accs = append(accs, &AccountHashH{Height: v.Height, Hash: v.Hash, Addr: v.Addr})
	}
	return accs
}
func deserializeAccounts(hs []*AccountHashH) []*common.AccountHashH {
	if hs == nil || len(hs) == 0 {
		return nil
	}
	var accs []*common.AccountHashH
	for _, v := range hs {
		accs = append(accs, common.NewAccountHashH(v.Addr, v.Hash, v.Height))
	}
	return accs[:]
}
