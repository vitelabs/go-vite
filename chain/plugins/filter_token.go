package chain_plugins

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type FilterToken struct {
	store *chain_db.Store
	chain Chain
}

func newFilterToken(store *chain_db.Store, chain Chain) Plugin {
	return &FilterToken{
		store: store,
		chain: chain,
	}
}

func (ft *FilterToken) InsertAccountBlock(batch *leveldb.Batch, accountBlock *ledger.AccountBlock) error {
	key := createDiffTokenKey(accountBlock.AccountAddress, accountBlock.TokenId, accountBlock.Height)
	batch.Put(key, accountBlock.Hash.Bytes())
	return nil
}

func (ft *FilterToken) InsertSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error {
	return nil
}

func (ft *FilterToken) DeleteAccountBlocks(batch *leveldb.Batch, accountBlocks []*ledger.AccountBlock) error {
	for _, accountBlock := range accountBlocks {
		key := createDiffTokenKey(accountBlock.AccountAddress, accountBlock.TokenId, accountBlock.Height)
		batch.Delete(key)
	}
	return nil
}

func (ft *FilterToken) DeleteSnapshotBlocks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	for _, chunk := range chunks {
		for _, accountBlock := range chunk.AccountBlocks {
			key := createDiffTokenKey(accountBlock.AccountAddress, accountBlock.TokenId, accountBlock.Height)
			batch.Delete(key)
		}
	}
	return nil
}

func (ft *FilterToken) GetBlocks(addr types.Address, tokenId types.TokenTypeId, blockHash *types.Hash, count uint64) ([]*ledger.AccountBlock, error) {
	maxHeight := helper.MaxUint64
	if blockHash != nil {
		block, err := ft.chain.GetAccountBlockByHash(*blockHash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, errors.New(fmt.Sprintf("block %s is not exited", blockHash))
		}

		maxHeight = block.Height + 1
	}

	iter := ft.store.NewIterator(&util.Range{Start: createDiffTokenKey(addr, tokenId, 0), Limit: createDiffTokenKey(addr, tokenId, maxHeight)})
	defer iter.Release()

	iterOk := iter.Last()
	index := uint64(0)
	blocks := make([]*ledger.AccountBlock, 0, count)
	if iterOk && index < count {
		value := iter.Value()
		hash, err := types.BytesToHash(value)
		if err != nil {
			return nil, err
		}
		block, err := ft.chain.GetAccountBlockByHash(hash)
		if err != nil {
			return nil, err
		}
		if block != nil {
			blocks = append(blocks, block)
		}
		index++
		iterOk = iter.Prev()
	}
	return blocks, nil
}

func createDiffTokenKey(addr types.Address, tokenId types.TokenTypeId, height uint64) []byte {
	key := make([]byte, 0, 1+types.AddressSize+types.TokenTypeIdSize+8)
	key = append(key, DiffTokenHash)
	key = append(key, addr.Bytes()...)
	key = append(key, tokenId.Bytes()...)
	key = append(key, chain_utils.Uint64ToBytes(height)...)
	return key
}
