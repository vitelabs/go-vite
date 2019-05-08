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

func (ft *FilterToken) SetStore(store *chain_db.Store) {
	ft.store = store
}

func (ft *FilterToken) InsertAccountBlock(batch *leveldb.Batch, accountBlock *ledger.AccountBlock) error {
	if accountBlock.BlockType == ledger.BlockTypeGenesisReceive {
		batch.Put(createDiffTokenKey(accountBlock.AccountAddress, ledger.ViteTokenId, accountBlock.Height), accountBlock.Hash.Bytes())
		batch.Put(createDiffTokenKey(accountBlock.AccountAddress, ledger.VCPTokenId, accountBlock.Height), accountBlock.Hash.Bytes())
		return nil
	}

	var tokenTypeId types.TokenTypeId

	if accountBlock.IsReceiveBlock() {
		sendBlock, err := ft.chain.GetAccountBlockByHash(accountBlock.FromBlockHash)

		if err != nil {
			return errors.New(fmt.Sprintf("ft.chain.GetAccountBlockByHash failed. Error: %s", err))
		}

		if sendBlock == nil {
			return errors.New(fmt.Sprintf("send block is nil"))
		}

		tokenTypeId = sendBlock.TokenId
	} else {

		tokenTypeId = accountBlock.TokenId
	}

	batch.Put(createDiffTokenKey(accountBlock.AccountAddress, tokenTypeId, accountBlock.Height), accountBlock.Hash.Bytes())

	for _, sendBlock := range accountBlock.SendBlockList {
		batch.Put(createDiffTokenKey(accountBlock.AccountAddress, sendBlock.TokenId, accountBlock.Height), accountBlock.Hash.Bytes())
	}
	return nil
}

func (ft *FilterToken) InsertSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error {
	return nil
}

func (ft *FilterToken) DeleteAccountBlocks(batch *leveldb.Batch, accountBlocks []*ledger.AccountBlock) error {
	sendBlocksMap := make(map[types.Hash]*ledger.AccountBlock)

	return ft.deleteAccountBlocks(batch, accountBlocks, sendBlocksMap)

}

func (ft *FilterToken) DeleteSnapshotBlocks(batch *leveldb.Batch, chunks []*ledger.SnapshotChunk) error {
	sendBlocksMap := make(map[types.Hash]*ledger.AccountBlock)

	for _, chunk := range chunks {
		if err := ft.deleteAccountBlocks(batch, chunk.AccountBlocks, sendBlocksMap); err != nil {
			return err
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
	for iterOk && index < count {

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

func (ft *FilterToken) deleteAccountBlocks(batch *leveldb.Batch, accountBlocks []*ledger.AccountBlock, sendBlocksMap map[types.Hash]*ledger.AccountBlock) error {
	for _, accountBlock := range accountBlocks {
		// add send blocks
		for _, sendBlock := range accountBlock.SendBlockList {
			sendBlocksMap[sendBlock.Hash] = sendBlock

			key := createDiffTokenKey(accountBlock.AccountAddress, sendBlock.TokenId, accountBlock.Height)
			batch.Delete(key)
		}
		var tokenTypeId types.TokenTypeId
		if accountBlock.IsSendBlock() {

			sendBlocksMap[accountBlock.Hash] = accountBlock

			tokenTypeId = accountBlock.TokenId

		} else if accountBlock.BlockType == ledger.BlockTypeGenesisReceive {
			tokenTypeId = accountBlock.TokenId
		} else {
			sendBlock, ok := sendBlocksMap[accountBlock.FromBlockHash]
			if !ok {
				var err error
				sendBlock, err = ft.chain.GetAccountBlockByHash(accountBlock.FromBlockHash)
				if err != nil {
					return err
				}
			} else {
				delete(sendBlocksMap, accountBlock.FromBlockHash)
			}

			if sendBlock == nil {
				return errors.New(fmt.Sprintf("send block is nil, send block: %+v\n", sendBlock))
			}
			tokenTypeId = sendBlock.TokenId

		}

		key := createDiffTokenKey(accountBlock.AccountAddress, tokenTypeId, accountBlock.Height)
		batch.Delete(key)
	}
	return nil
}

func createDiffTokenKey(addr types.Address, tokenId types.TokenTypeId, height uint64) []byte {
	key := make([]byte, 0, 1+types.AddressSize+types.TokenTypeIdSize+8)
	key = append(key, DiffTokenHash)
	key = append(key, addr.Bytes()...)
	key = append(key, tokenId.Bytes()...)
	key = append(key, chain_utils.Uint64ToBytes(height)...)
	return key
}
