package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
	"bytes"
	"sync"
	"github.com/pkg/errors"
	"encoding/hex"
	"fmt"
)

type blockWriteMutexBody struct {
	LatestBlock *ledger.AccountBlock
	Reference bool
}

type blockWriteMutex map[string]*blockWriteMutexBody

const bwmuBuffer = 10 * 10000
const bwmuReleaseCount = 2 * 10000

// The mutex for blockWriteMutex execute locking or unlocking
var bwMutexMutex sync.Mutex
func (bwm *blockWriteMutex) release () {
	count := 0
	for key, mutexBody := range *bwm {
		if count >= bwmuReleaseCount {
			return
		}
		if !mutexBody.Reference {
			delete(*bwm, key)
			count++
		}
	}
}

func (bwm *blockWriteMutex) Lock (block *ledger.AccountBlock, meta *ledger.AccountMeta) error {
	bwMutexMutex.Lock()
	defer bwMutexMutex.Unlock()

	accountAddress := block.AccountAddress
	mutexBody, ok:= (*bwm)[accountAddress.String()]

	if !ok || mutexBody == nil {
		if len(*bwm) >= bwmuBuffer {
			// Release memory
			bwm.release()
		}

		mutexBody = &blockWriteMutexBody {
			Reference: false,
		}

		if meta != nil {
			var err error
			mutexBody.LatestBlock, err = accountChainAccess.store.GetLatestBlockByAccountId(meta.AccountId)
			if err != nil {
				return err
			}
		}

		(*bwm)[accountAddress.String()] = mutexBody
	}

	if mutexBody.Reference {
		return errors.New("Lock failed")
	}

	if mutexBody.LatestBlock != nil &&
		!bytes.Equal(mutexBody.LatestBlock.Hash, block.PrevHash) {
		return errors.New("PrevHash of accountBlock which will be write is not the hash of the latest account block.")
	}

	mutexBody.Reference = true

	return nil
}

func (bwm *blockWriteMutex) UnLock (block *ledger.AccountBlock, writeErr error) {
	bwMutexMutex.Lock()
	defer bwMutexMutex.Unlock()

	accountAddress := block.AccountAddress

	mutexBody, ok:= (*bwm)[accountAddress.String()]
	if !ok {
		return
	}

	if writeErr == nil {
		mutexBody.LatestBlock = block
	}

	mutexBody.Reference = false
}


type AccountChainAccess struct {
	store         *vitedb.AccountChain
	accountStore  *vitedb.Account
	snapshotStore *vitedb.SnapshotChain
	tokenStore    *vitedb.Token
	bwMutex 	  *blockWriteMutex
}


var accountChainAccess = &AccountChainAccess{
	store:         vitedb.GetAccountChain(),
	accountStore:  vitedb.GetAccount(),
	snapshotStore: vitedb.GetSnapshotChain(),
	tokenStore:    vitedb.GetToken(),
	bwMutex:	   &blockWriteMutex{},
}

func GetAccountChainAccess() *AccountChainAccess {
	return accountChainAccess
}


func (aca *AccountChainAccess) WriteBlockList(blockList []*ledger.AccountBlock) error {
	for _, block := range blockList {
		err := aca.WriteBlock(block)
		if err != nil {
			return err
		}
	}
	return nil
}

func (aca *AccountChainAccess) WriteBlock(block *ledger.AccountBlock) error {
	err := aca.store.BatchWrite(nil, func(batch *leveldb.Batch) error {
		return aca.writeBlock(batch, block)
	})

	if err != nil {
		fmt.Println("Write block " + hex.EncodeToString(block.Hash) + " failed, block data is ")
		fmt.Printf("%+v\n", block)
	} else {
		fmt.Println("Write block " + hex.EncodeToString(block.Hash) + " succeed.")
	}

	return err
}

func (aca *AccountChainAccess) writeBlock(batch *leveldb.Batch, block *ledger.AccountBlock) (result error) {
	accountMeta, err := aca.accountStore.GetAccountMetaByAddress(block.AccountAddress)
	if err != nil && err != leveldb.ErrNotFound{
		return err
	}

	// Mutex for a accountAddress
	if err := aca.bwMutex.Lock(block, accountMeta); err != nil {
		// Not Lock
		return err
	}

	defer aca.bwMutex.UnLock(block, result)

	var currentAccountToken *ledger.AccountSimpleToken

	isGenesisBlock :=  block.AccountAddress.String() == ledger.GenesisAccount.String() && block.PrevHash == nil
	if block.SnapshotTimestamp == nil {
		return errors.New("Fail to write block, because block.SnapshotTimestamp is uncorrect.")
	}

	if accountMeta != nil {
		// Get token info of account
		for _, token := range accountMeta.TokenList {
			if token.TokenId.String() == block.TokenId.String() {
				currentAccountToken = token
				break
			}
		}
	} else if block.FromHash != nil || isGenesisBlock {
		// If account doesn't exist and the block is a response block, we must create account
		lastAccountID, err := aca.accountStore.GetLastAccountId()
		if err != nil {
			return  err
		}

		if lastAccountID == nil {
			lastAccountID = big.NewInt(0)
		}

		newAccountId := &big.Int{}
		newAccountId.Add(lastAccountID, big.NewInt(1))

		// Set currentAccountToken when create account
		if block.TokenId != nil {
			currentAccountToken = &ledger.AccountSimpleToken{
				TokenId: block.TokenId,
				LastAccountBlockHeight: big.NewInt(1),
			}
		}

		// Create account meta which will be write to database later
		accountMeta = &ledger.AccountMeta{
			AccountId: newAccountId,
			TokenList: []*ledger.AccountSimpleToken{},
		}

		if currentAccountToken != nil {
			accountMeta.TokenList = append(accountMeta.TokenList, currentAccountToken)
		}

		if err := aca.accountStore.WriteAccountIdIndex(batch, newAccountId, block.AccountAddress); err != nil {
			return err
		}
	}

	var prevAccountBlockInToken *ledger.AccountBlock

	if currentAccountToken != nil {
		prevAccountBlockInToken, err = aca.store.GetBlockByHeight(accountMeta.AccountId, currentAccountToken.LastAccountBlockHeight)

		if err != nil && err != leveldb.ErrNotFound {
			return err
		}
	}



	if block.FromHash == nil {
		// Send block

		if !isGenesisBlock {
			if accountMeta == nil  {
				return errors.New("Write send block failed, because the account does not exist.")
			}

			if currentAccountToken == nil {
				return errors.New("Write send block failed, because the account does not have this token")
			}


			if  prevAccountBlockInToken == nil || block.Amount.Cmp(prevAccountBlockInToken.Balance) > 0 {
				return errors.New("Write send block failed, because the balance is not enough")
			}
		}


		// Sub balance
		if block.Amount != nil {
			block.Balance = &big.Int{}
			block.Balance.Sub(prevAccountBlockInToken.Balance, block.Amount)
		}

		if bytes.Equal(block.To.Bytes(), ledger.MintageAddress.Bytes()) {
			mintage, err := ledger.NewMintage(block)
			if err != nil {
				return err
			}

			// Write Mintage
			if err := tokenAccess.WriteMintage(batch, mintage, block); err != nil{
				return err
			}
		}

	} else {
		// Response block
		fromBlock, err := aca.store.GetBlockByHash(block.FromHash)

		if err != nil {
			return err
		}
		if fromBlock == nil{
			return errors.New("Write receive block failed, because the from block is not exist")
		}

		amount :=  fromBlock.Amount
		if bytes.Equal(fromBlock.To.Bytes(), ledger.MintageAddress.Bytes()) {
			mintage, err := ledger.NewMintage(fromBlock)

			if err != nil {
				return err
			}

			if mintage.Owner.String() != block.AccountAddress.String() {
				return errors.New("You are not the owner of this token.")
			}

			if currentAccountToken == nil {
				currentAccountToken = &ledger.AccountSimpleToken{
					TokenId: mintage.Id,
				}

				accountMeta.TokenList = append(accountMeta.TokenList, currentAccountToken)
			}


			amount = mintage.TotalSupply
			for i :=0 ; i < mintage.Decimals; i++ {
				amount.Mul(amount, big.NewInt(10))
			}
		}

		// Add balance
		prevBalance := big.NewInt(0)
		if prevAccountBlockInToken != nil {
			prevBalance = prevAccountBlockInToken.Balance
		}


		if block.Balance == nil {
			block.Balance = big.NewInt(0)
		}

		block.Balance.Add(prevBalance, amount)
	}

	// Write account block
	latestBlockHeight, err := aca.store.GetLatestBlockHeightByAccountId(accountMeta.AccountId)
	if err != nil {
		return err
	}

	if latestBlockHeight == nil {
		latestBlockHeight = big.NewInt(0)
	}

	newBlockHeight := latestBlockHeight.Add(latestBlockHeight, big.NewInt(1))

	if err := aca.store.WriteBlock(batch, accountMeta.AccountId, newBlockHeight, block); err != nil {
		return err
	}

	// Write account meta
	if currentAccountToken != nil {
		currentAccountToken.LastAccountBlockHeight = newBlockHeight
	}

	if err := aca.accountStore.WriteMeta(batch, block.AccountAddress, accountMeta); err != nil {
		return err
	}


	// Write account block meta
	newBlockMeta := &ledger.AccountBlockMeta {
		Height: newBlockHeight,
		AccountId: accountMeta.AccountId,
	}

	if err := aca.writeBlockMeta(batch, block, newBlockMeta); err != nil {
		return err
	}

	if err := aca.writeTii(batch, block); err != nil {
		return err
	}

	if err:= aca.writeStIndex(batch, block); err != nil {
		return err
	}

	return nil
}

// Tii is TokenIdIndex
func (aca *AccountChainAccess) writeBlockMeta (batch *leveldb.Batch, block *ledger.AccountBlock, meta *ledger.AccountBlockMeta) error {
	if block.FromHash == nil {
		meta.Status = 1 // open
	} else {
		meta.Status = 2 // closed
		fromBlockMeta, err := aca.store.GetBlockMeta(block.FromHash)
		if fromBlockMeta == nil {
			return errors.New("Write receive block failed, because the from block is not exist")
		}

		if err != nil {
			return err
		}

		fromBlockMeta.Status = 2 // closed

		if err := aca.store.WriteBlockMeta(batch, block.FromHash, fromBlockMeta); err != nil {
			return err
		}
	}


	if err := aca.store.WriteBlockMeta(batch, block.Hash, meta); err != nil {
		return err
	}
	return nil
}

// Temporary code
type tokenIdCacheBody struct {
	LastTokenBlockHeight *big.Int
}

var tokenIdCache = make(map[string]*tokenIdCacheBody)
var tokenIdMutex sync.Mutex

// Tii is TokenIdIndex
func (aca *AccountChainAccess) writeTii (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.TokenId == nil {
		return nil
	}

	tokenIdMutex.Lock()
	defer tokenIdMutex.Unlock()

	cacheBody, ok := tokenIdCache[block.TokenId.String()]

	newBlockHeightInToken := &big.Int{}
	// Write TokenId index
	if !ok {
		latestBlockHeightInToken, err := aca.tokenStore.GetLatestBlockHeightByTokenId(block.TokenId)

		if err == leveldb.ErrNotFound {
			latestBlockHeightInToken = big.NewInt(-1)
		} else if err != nil {
			return err
		}

		cacheBody = &tokenIdCacheBody {
			LastTokenBlockHeight: latestBlockHeightInToken,
		}
	}

	newBlockHeightInToken.Add(cacheBody.LastTokenBlockHeight, big.NewInt(1))


	if err := aca.tokenStore.WriteTokenIdIndex(batch, block.TokenId, newBlockHeightInToken, block.Hash); err != nil {
		return err
	}

	cacheBody.LastTokenBlockHeight = newBlockHeightInToken
	return nil
}

// Temporary code
type stIdCacheBody struct {
	LastStId *big.Int
}

var stIdCache = make(map[string]*stIdCacheBody)
var stIdMutex sync.Mutex

func (aca *AccountChainAccess) getNewLastStId (block *ledger.AccountBlock) (*big.Int, error) {
	stIdMutex.Lock()
	defer stIdMutex.Unlock()

	cacheBody, ok := stIdCache[string(block.SnapshotTimestamp)]

	if !ok {
		var stHeight *big.Int
		if bytes.Equal(block.SnapshotTimestamp, ledger.GenesisSnapshotBlockHash) {
			stHeight = big.NewInt(0)
		}  else  {
			var err error
			stHeight, err = aca.snapshotStore.GetHeightByHash(block.SnapshotTimestamp)
			if err != nil {
				return nil, err
			}
		}


		lastStId, err := aca.store.GetLastIdByStHeight(stHeight)
		if err != nil {
			return nil, err
		}

		if lastStId == nil {
			lastStId = big.NewInt(0)
		}

		cacheBody = &stIdCacheBody{
			LastStId:  lastStId,
		}
		stIdCache[string(block.SnapshotTimestamp)] = cacheBody
	}

	// Write st index
	cacheBody.LastStId.Add(cacheBody.LastStId, big.NewInt(1))

	return cacheBody.LastStId, nil

}

func (aca *AccountChainAccess) writeStIndex (batch *leveldb.Batch, block *ledger.AccountBlock) error {
	// Write st index
	newStId, err := aca.getNewLastStId(block)
	if err != nil {
		return err
	}

	if err := aca.store.WriteStIndex(batch, block.SnapshotTimestamp, newStId, block.Hash); err != nil {
		return err
	}
	return nil
}

func (aca *AccountChainAccess) GetBlockByHash(blockHash []byte) (*ledger.AccountBlock, error) {
	accountBlock, err := aca.store.GetBlockByHash(blockHash)

	if err != nil {
		return nil, err
	}

	return aca.processBlock(accountBlock, nil)
}

func (aca *AccountChainAccess) processBlock (accountBlock *ledger.AccountBlock, accountAddress *types.Address) (*ledger.AccountBlock, error) {
	if accountBlock.Meta == nil {
		var err error
		accountBlock.Meta, err = aca.store.GetBlockMeta(accountBlock.Hash)
		if err != nil{
			return nil, err
		}
	}
	if accountBlock.FromHash != nil &&  accountBlock.From == nil{
		fromAccountBlockMeta, err := aca.store.GetBlockMeta(accountBlock.FromHash)

		if err != nil {
			return nil, err
		}

		fromAddress, err := aca.accountStore.GetAddressById(fromAccountBlockMeta.AccountId)
		if err != nil {
			return nil, errors.New("GetAddressById func error ")
		}

		accountBlock.From = fromAddress
	}

	if accountBlock.AccountAddress == nil {
		if accountAddress != nil {
			accountBlock.AccountAddress = accountAddress
		} else {
			accountId := accountBlock.Meta.AccountId
			var err error
			accountBlock.AccountAddress, err = aca.accountStore.GetAddressById(accountId)
			if err != nil {
				return nil, errors.Wrap(err, "[AccountChainAccess.GetBlockByHash]")
			}

			if err != nil {
				return nil, errors.Wrap(err, "[AccountChainAccess.GetBlockByHash]")
			}
		}

	}

	return accountBlock, nil
}



func (aca *AccountChainAccess) GetBlockListByAccountAddress(index int, num int, count int, accountAddress *types.Address) ([]*ledger.AccountBlock, error) {
	accountMeta, err := aca.accountStore.GetAccountMetaByAddress(accountAddress)
	if err != nil {
		return nil, err
	}

	blockList, err := aca.store.GetBlockListByAccountMeta(index, num, count, accountMeta)
	if err != nil {
		return nil, err
	}

	var processedBlockList = make([]*ledger.AccountBlock, len(blockList))
	for index, block := range blockList {
		processedBlockList[index], err = aca.processBlock(block, accountAddress)
		if err != nil {
			return nil, err
		}
	}
	return processedBlockList, nil
}

func (aca *AccountChainAccess) GetBlockListByTokenId(index int, num int, count int, tokenId *types.TokenTypeId) ([]*ledger.AccountBlock, error) {
	blockHashList, err := aca.tokenStore.GetAccountBlockHashListByTokenId(index, num, count, tokenId)
	if err != nil {
		return nil, err
	}
	var accountBlockList []*ledger.AccountBlock
	for _, blockHash := range blockHashList {
		block, err := aca.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		accountBlockList = append(accountBlockList, block)
	}

	return accountBlockList, nil
}

func (aca *AccountChainAccess) GetBlockList(index, num, count int) ([]*ledger.AccountBlock, error) {
	blockHashList, err := aca.store.GetBlockHashList(index, num, count)
	if err != nil {
		return nil, err
	}

	var blockList []*ledger.AccountBlock
	for _, blockHash := range blockHashList {
		block, err := aca.GetBlockByHash(blockHash)
		if err != nil {
			return nil, err
		}
		blockList = append(blockList, block)
	}

	return blockList, nil
}

func (aca *AccountChainAccess) GetConfirmBlock(accountBlock *ledger.AccountBlock) (*ledger.SnapshotBlock, error) {
	var err error
	var confirmSnapshotBlock *ledger.SnapshotBlock

	aca.snapshotStore.Iterate(func(snapshotBlock *ledger.SnapshotBlock) bool {
		if itemAccountBlockHash, ok := snapshotBlock.Snapshot[accountBlock.AccountAddress.String()]; ok {
			var itemAccountBlockMeta *ledger.AccountBlockMeta
			itemAccountBlockMeta, err = aca.store.GetBlockMeta(itemAccountBlockHash)
			if itemAccountBlockMeta.Height.Cmp(accountBlock.Meta.Height) > 0 {
				confirmSnapshotBlock = snapshotBlock
				return false
			}
		}
		return false
	}, accountBlock.SnapshotTimestamp)

	return confirmSnapshotBlock, err

}

func (aca *AccountChainAccess) GetConfirmTimes(confirmSnapshotBlock *ledger.SnapshotBlock) (*big.Int, error) {
	if confirmSnapshotBlock == nil {
		return nil, nil
	}

	latestBlock, err := aca.snapshotStore.GetLatestBlock()
	if err != nil {
		return nil, err
	}

	result := &big.Int{}
	result = result.Sub(latestBlock.Height, confirmSnapshotBlock.Height)
	return result, nil
}

func (aca *AccountChainAccess) GetAccountBalance (accountId *big.Int, blockHeight *big.Int) (*big.Int, error) {
	accountBLock, err := aca.store.GetBlockByHeight(accountId, blockHeight)
	if err != nil {
		return nil, err
	}
	return accountBLock.Balance, nil
}

func (aca *AccountChainAccess) GetLatestBlockHeightByAccountId (accountId *big.Int) (* big.Int, error){
	return aca.store.GetLatestBlockHeightByAccountId(accountId)
}

