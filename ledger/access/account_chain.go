package access

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitedb"
	"log"
	"math/big"
	"sync"
)

type blockWriteMutexBody struct {
	LatestBlock *ledger.AccountBlock
	WriteLock   sync.Mutex
}

type blockWriteMutex map[string]*blockWriteMutexBody

// The mutex for blockWriteMutex execute locking or unlocking
var bwMutexMutex sync.Mutex

func (bwm *blockWriteMutex) Lock(block *ledger.AccountBlock, meta *ledger.AccountMeta) (returnError *AcWriteError) {
	bwMutexMutex.Lock()
	accountAddress := block.AccountAddress
	mutexBody, ok := (*bwm)[accountAddress.String()]

	if !ok || mutexBody == nil {
		mutexBody = &blockWriteMutexBody{}

		if meta != nil {
			var err error
			mutexBody.LatestBlock, err = accountChainAccess.store.GetLatestBlockByAccountId(meta.AccountId)
			if err != nil {
				bwMutexMutex.Unlock()
				return &AcWriteError{
					Code: WacDefaultErr,
					Err:  err,
				}

			}
		}

		(*bwm)[accountAddress.String()] = mutexBody
	}
	bwMutexMutex.Unlock()

	mutexBody.WriteLock.Lock()
	defer func() {
		if returnError != nil {
			mutexBody.WriteLock.Unlock()
		}
	}()
	if mutexBody.LatestBlock != nil {
		if block.PrevHash == nil {
			return &AcWriteError{
				Code: WacPrevHashUncorrectErr,
				Err:  errors.New("PrevHash of accountBlock is nil. Can't write."),
				Data: mutexBody.LatestBlock,
			}
		}
		if !bytes.Equal(mutexBody.LatestBlock.Hash.Bytes(), block.PrevHash.Bytes()) {
			if block.Meta == nil || block.Meta.Height == nil {
				return &AcWriteError{
					Code: WacPrevHashUncorrectErr,
					Err:  errors.New("PrevHash of accountBlock which will be write is not the hash of the latest account block. Can't write."),
					Data: mutexBody.LatestBlock,
				}
			} else {
				cmpResult := block.Meta.Height.Cmp(mutexBody.LatestBlock.Meta.Height)
				err := errors.New("PrevHash of accountBlock which will be write is not the hash of the latest account block. Current Latest block hash is " +
					mutexBody.LatestBlock.Hash.String() + " and Writing block hash is " + block.PrevHash.String() +
					", Current latest block height is " + mutexBody.LatestBlock.Meta.Height.String() +
					" and Writing block height is " + block.Meta.Height.String())

				if cmpResult == 0 {
					return &AcWriteError{
						Code: WacSameHeightErr,
						Err:  err,
						Data: mutexBody.LatestBlock,
					}
				} else if cmpResult < 0 {
					return &AcWriteError{
						Code: WacLowerErr,
						Err:  err,
						Data: mutexBody.LatestBlock,
					}
				} else {
					return &AcWriteError{
						Code: WacHigherErr,
						Err:  err,
						Data: mutexBody.LatestBlock,
					}
				}
			}
		}

		if block.Meta != nil && block.Meta.Height != nil {
			gap := big.NewInt(0)
			gap.Sub(block.Meta.Height, mutexBody.LatestBlock.Meta.Height)
			if gap.Cmp(big.NewInt(1)) != 0 {
				return &AcWriteError{
					Code: WacUncorrectHeightErr,
					Err:  errors.New("Height is error."),
					Data: mutexBody.LatestBlock,
				}
			}
		}
	}

	return nil
}

func (bwm *blockWriteMutex) UnLock(block *ledger.AccountBlock, writeErr *AcWriteError) {
	bwMutexMutex.Lock()
	defer bwMutexMutex.Unlock()

	accountAddress := block.AccountAddress

	mutexBody, ok := (*bwm)[accountAddress.String()]

	if !ok {
		log.Println("blockWriteMutex Unlock: unlock failed. because mutexBody doesn't exists.")
		return
	}

	if writeErr == nil {
		mutexBody.LatestBlock = block
	} else {
		log.Println("blockWriteMutex Unlock: writeErr is " + writeErr.Error())
	}

	mutexBody.WriteLock.Unlock()
}

type AccountChainAccess struct {
	store            *vitedb.AccountChain
	accountStore     *vitedb.Account
	snapshotStore    *vitedb.SnapshotChain
	tokenStore       *vitedb.Token
	unconfirmedStore *vitedb.Unconfirmed
	bwMutex          *blockWriteMutex
}

var accountChainAccess *AccountChainAccess

func GetAccountChainAccess() *AccountChainAccess {
	if accountChainAccess == nil {
		accountChainAccess = &AccountChainAccess{
			store:            vitedb.GetAccountChain(),
			accountStore:     vitedb.GetAccount(),
			snapshotStore:    vitedb.GetSnapshotChain(),
			tokenStore:       vitedb.GetToken(),
			unconfirmedStore: vitedb.GetUnconfirmed(),
			bwMutex:          &blockWriteMutex{},
		}
	}
	return accountChainAccess
}

func (aca *AccountChainAccess) WriteBlockList(blockList []*ledger.AccountBlock) error {
	for _, block := range blockList {
		err := aca.WriteBlock(block, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

type signAccountBlockFuncType func(*ledger.AccountBlock) (*ledger.AccountBlock, error)

// [Fixme]
var WriteBlockMutex sync.Mutex

func (aca *AccountChainAccess) WriteBlock(block *ledger.AccountBlock, signFunc signAccountBlockFuncType) error {
	err := aca.store.BatchWrite(nil, func(batch *leveldb.Batch) error {
		// [Fixme]
		WriteBlockMutex.Lock()
		defer WriteBlockMutex.Unlock()

		// When *AcWriteError data type convert to error interface, nil become non-nil. So need return nil manually
		if err := aca.writeBlock(batch, block, signFunc); err != nil {
			return err
		}

		// [Fixme]
		if counterAddError := aca.store.CounterAdd(batch); counterAddError != nil {
			return counterAddError
		}
		return nil
	})

	// Send signal to listener if exists.
	if err == nil && block.IsSendBlock() {
		// [Fixme]
		GetUnconfirmedAccess().SendSignalToListener(*block.To)
	}
	return err
}

func (aca *AccountChainAccess) WriteGenesisBlock() {
	if err := aca.WriteBlock(ledger.AccountGenesisBlockFirst, nil); err != nil {
		log.Fatal(errors.Wrap(err, "accountChain.WriteGenesisBlock"))
	}

	log.Println("accountChain.WriteGenesisBlock success.")
}

func (aca *AccountChainAccess) WriteGenesisSecondBlock() {
	if err := accountChainAccess.WriteBlock(ledger.AccountGenesisBlockSecond, nil); err != nil {
		log.Fatal(errors.Wrap(err, "accountChain.WriteGenesisSecondBlock"))
	}

	log.Println("accountChain.WriteGenesisSecondBlock success.")
}

func (aca *AccountChainAccess) writeSendBlock(batch *leveldb.Batch, block *ledger.AccountBlock, accountMeta *ledger.AccountMeta) error {
	if accountMeta == nil && !block.IsGenesisBlock() {
		return errors.New("Write the send block failed, because the account does not exist.")
	}

	if block.IsMintageBlock() {
		return aca.writeMintageBlock(batch, block)
	}

	if block.TokenId == nil {
		return errors.New("Write the send block failed, because the token id of block is nil.")
	}

	accountTokenInfo := accountMeta.GetTokenInfoByTokenId(block.TokenId)

	if accountTokenInfo == nil {
		return errors.New("Write the send block failed, because the account does not have this token.")
	}

	prevAccountBlockInToken, prevAbErr := aca.store.GetBlockByHeight(accountMeta.AccountId, accountTokenInfo.LastAccountBlockHeight)
	if prevAbErr != nil || prevAccountBlockInToken == nil {
		return errors.New("Write the send block failed, because the balance is not enough.")
	}

	if block.Amount == nil {
		return errors.New("Write the send block failed, because the block.Amount does not exist.")
	}

	if block.Amount.Cmp(prevAccountBlockInToken.Balance) >= 0 {
		return errors.New("Write the send block failed, because the balance is not enough.")
	}

	block.Balance = &big.Int{}
	block.Balance.Sub(prevAccountBlockInToken.Balance, block.Amount)

	return nil
}

func (aca *AccountChainAccess) writeReceiveBlock(batch *leveldb.Batch, block *ledger.AccountBlock, accountMeta *ledger.AccountMeta) error {
	// Get from block
	if block.FromHash == nil {
		return errors.New("Write the receive block failed, because the fromHash does not exist.")
	}

	fromBlock, err := aca.store.GetBlockByHash(block.FromHash)
	if err != nil {
		return errors.New("Write receive block failed, because getting the from block failed. Error is " + err.Error())
	}
	if fromBlock == nil {
		return errors.New("Write receive block failed, because the from block is not exist")
	}

	var amount = fromBlock.Amount

	if fromBlock.IsMintageBlock() {
		// Receive is mintageBlock
		mintage, err := ledger.NewMintage(fromBlock)

		if err != nil {
			return err
		}

		if mintage.Owner.String() != block.AccountAddress.String() {

			return errors.New("You are not the owner of this token.")
		}

		amount = mintage.TotalSupply
		for i := 0; i < mintage.Decimals; i++ {
			amount.Mul(amount, big.NewInt(10))
		}

		block.Balance = amount
		block.Amount = amount
		block.TokenId = mintage.Id

		log.Println("Accountchain WriteReceiveBlock: mintage receive. amount is " + amount.String() + ", balance is " + block.Balance.String())
	} else {

		// Add balance
		prevBalance := big.NewInt(0)

		accountTokenInfo := accountMeta.GetTokenInfoByTokenId(fromBlock.TokenId)
		if accountTokenInfo == nil {
			accountTokenInfo = &ledger.AccountSimpleToken{
				TokenId: block.TokenId,
			}
		}

		if accountTokenInfo.LastAccountBlockHeight != nil {
			prevAccountBlockInToken, prevAbErr := aca.store.GetBlockByHeight(accountMeta.AccountId, accountTokenInfo.LastAccountBlockHeight)
			if prevAbErr != nil || prevAccountBlockInToken == nil {
				return errors.New("Write receive block failed, Error is " + prevAbErr.Error())
			}
			prevBalance = prevAccountBlockInToken.Balance
		}

		block.Balance = big.NewInt(0)
		block.Balance.Add(prevBalance, amount)
		block.Amount = amount
		block.TokenId = fromBlock.TokenId

		log.Println("Accountchain WriteReceiveBlock: prevBalance is " + prevBalance.String() + " , amount is " + amount.String() + ", balance is " + block.Balance.String())
	}

	// Write from block meta
	fromBlock.Meta.Status = 2

	if err := aca.writeBlockMeta(batch, fromBlock); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	return nil
}

func (aca *AccountChainAccess) writeMintageBlock(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	mintage, err := ledger.NewMintage(block)
	if err != nil {
		return err
	}

	// Write Mintage
	if err := tokenAccess.WriteMintage(batch, mintage, block); err != nil {
		return err
	}
	return nil
}

func (aca *AccountChainAccess) writeBlock(batch *leveldb.Batch, block *ledger.AccountBlock, signFunc signAccountBlockFuncType) (acWriteError *AcWriteError) {

	// AccountBlock must have the snapshotTimestamp
	if block.SnapshotTimestamp == nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  errors.New("Fail to write block, because block.SnapshotTimestamp is uncorrect."),
		}
	}

	accountMeta, err := aca.accountStore.GetAccountMetaByAddress(block.AccountAddress)
	if err != nil && err != leveldb.ErrNotFound {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	// Mutex for an accountAddress
	if err := aca.bwMutex.Lock(block, accountMeta); err != nil {
		// Not Lock
		return err
	}

	// Unlock mutex
	defer func() {
		aca.bwMutex.UnLock(block, acWriteError)
	}()

	var latestBlockHeight *big.Int

	needCreateNewAccount := false
	if block.IsSendBlock() {
		err = aca.writeSendBlock(batch, block, accountMeta)
		// Write account block
	} else if block.IsReceiveBlock() {
		if accountMeta == nil {

			writeNewAccountMutex.Lock()
			defer writeNewAccountMutex.Unlock()
			needCreateNewAccount = true
			accountMeta, err = GetAccountAccess().CreateNewAccountMeta(batch, block.AccountAddress, block.PublicKey)
			latestBlockHeight = big.NewInt(0)
		}

		err = aca.writeReceiveBlock(batch, block, accountMeta)
	}

	if err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}

	// Set new block height
	newBlockHeight := big.NewInt(0)

	if block.Meta != nil && block.Meta.Height != nil {
		newBlockHeight = block.Meta.Height
	} else {
		if latestBlockHeight == nil {
			latestBlockHeight, err = aca.store.GetLatestBlockHeightByAccountId(accountMeta.AccountId)
			if err != nil || latestBlockHeight == nil {
				return &AcWriteError{
					Code: WacDefaultErr,
					Err:  errors.New("Write the block failed, because the latestBlockHeight is error."),
				}
			}
		}
		newBlockHeight.Add(latestBlockHeight, big.NewInt(1))
	}

	// Set account block meta
	newBlockMeta := &ledger.AccountBlockMeta{
		Height:    newBlockHeight,
		AccountId: accountMeta.AccountId,
	}

	if block.IsSendBlock() {
		newBlockMeta.Status = 1 // Open
	} else {
		newBlockMeta.Status = 2 // Closed
	}

	block.Meta = newBlockMeta

	// Set account meta
	if block.TokenId != nil {
		accountTokenInfo := accountMeta.GetTokenInfoByTokenId(block.TokenId)
		if accountTokenInfo != nil {
			accountTokenInfo.LastAccountBlockHeight = newBlockHeight
		} else {
			accountTokenInfo = &ledger.AccountSimpleToken{
				TokenId:                block.TokenId,
				LastAccountBlockHeight: block.Meta.Height,
			}
		}

		accountMeta.SetTokenInfo(accountTokenInfo)
	}

	// Set hash
	if block.Hash == nil {
		hash, err := block.ComputeHash()
		if err != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  err,
			}
		}

		block.Hash = hash
	}
	log.Printf("AccountChainAccess writeblock: set hash success.")

	// Sign
	if signFunc != nil && block.Signature == nil {
		var signErr error

		block, signErr = signFunc(block)

		if signErr != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  signErr,
			}
		}
	}
	log.Printf("AccountChainAccess writeblock: sign success.")

	// Write account meta
	if err := aca.accountStore.WriteMeta(batch, block.AccountAddress, accountMeta); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	log.Printf("AccountChainAccess writeblock: writeMeta success.")

	// Write account id index
	if needCreateNewAccount {
		if err := aca.accountStore.WriteAccountIdIndex(batch, accountMeta.AccountId, block.AccountAddress); err != nil {
			return &AcWriteError{
				Code: WacDefaultErr,
				Err:  err,
			}
		}
	}
	log.Printf("AccountChainAccess writeblock: Write account id index success.")

	// Write account block meta
	if wrbErr := aca.writeBlockMeta(batch, block); wrbErr != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  wrbErr,
		}
	}
	log.Printf("AccountChainAccess writeblock: Write block meta success.")

	// Write account block
	if err := aca.store.WriteBlock(batch, accountMeta.AccountId, block); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  errors.New("Write the block failed, error is " + err.Error()),
		}
	}
	log.Printf("AccountChainAccess writeblock: Write block success.")

	// Write tii
	if err := aca.writeTii(batch, block); err != nil {
		return &AcWriteError{
			Code: WacDefaultErr,
			Err:  err,
		}
	}
	log.Printf("AccountChainAccess writeblock: Write tii success.")

	// Write st index
	if err := aca.writeStIndex(batch, block); err != nil {
		log.Println("AccountChainAccess writeBlock warning: writeStIndex failed.")
	}

	return nil
}

// Tii is TokenIdIndex
func (aca *AccountChainAccess) writeBlockMeta(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	if block.IsSendBlock() && !block.IsGenesisBlock() && !block.IsGenesisSecondBlock() {
		if block.Meta.Status == 1 {
			if err := GetUnconfirmedAccess().WriteBlock(batch, block); err != nil {
				return &AcWriteError{
					Code: WacDefaultErr,
					Err:  err,
				}
			}

		} else if block.Meta.Status == 2 {
			if err := GetUnconfirmedAccess().DeleteBlock(batch, block); err != nil {
				return &AcWriteError{
					Code: WacDefaultErr,
					Err:  err,
				}
			}
		}
	}

	if err := aca.store.WriteBlockMeta(batch, block.Hash, block.Meta); err != nil {
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
func (aca *AccountChainAccess) writeTii(batch *leveldb.Batch, block *ledger.AccountBlock) error {
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

		cacheBody = &tokenIdCacheBody{
			LastTokenBlockHeight: latestBlockHeightInToken,
		}
	}

	log.Println("AccountChainAccess writeTii: last block height in token is " + cacheBody.LastTokenBlockHeight.String())
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

func (aca *AccountChainAccess) getNewLastStId(block *ledger.AccountBlock) (*big.Int, error) {
	stIdMutex.Lock()
	defer stIdMutex.Unlock()

	cacheBody, ok := stIdCache[block.SnapshotTimestamp.String()]

	if !ok {
		var stHeight *big.Int
		if block.SnapshotTimestamp.String() == ledger.SnapshotGenesisBlock.Hash.String() {
			stHeight = big.NewInt(1)
		} else {
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
			LastStId: lastStId,
		}
		stIdCache[block.SnapshotTimestamp.String()] = cacheBody
	}

	// Write st index
	cacheBody.LastStId.Add(cacheBody.LastStId, big.NewInt(1))

	return cacheBody.LastStId, nil

}

func (aca *AccountChainAccess) writeStIndex(batch *leveldb.Batch, block *ledger.AccountBlock) error {
	// Write st index
	newStId, err := aca.getNewLastStId(block)
	if err != nil {
		return err
	}

	if err := aca.store.WriteStIndex(batch, block.SnapshotTimestamp.Bytes(), newStId, block.Hash); err != nil {
		return err
	}
	return nil
}

func (aca *AccountChainAccess) GetBlocksFromOrigin(originBlockHash *types.Hash, count uint64, forward bool) (ledger.AccountBlockList, error) {
	return aca.store.GetBlocksFromOrigin(originBlockHash, count, forward)
}

func (aca *AccountChainAccess) GetBlockMetaByHash(blockHash *types.Hash) (*ledger.AccountBlockMeta, error) {
	accountBlockMeta, err := aca.store.GetBlockMeta(blockHash)

	if err != nil {
		return nil, err
	}

	return accountBlockMeta, nil
}

func (aca *AccountChainAccess) GetBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	accountBlock, err := aca.store.GetBlockByHash(blockHash)

	if err != nil {
		return nil, err
	}

	return aca.processBlock(accountBlock, nil)
}

func (aca *AccountChainAccess) processBlock(accountBlock *ledger.AccountBlock, accountAddress *types.Address) (*ledger.AccountBlock, error) {
	if accountBlock.Meta == nil {
		var err error
		accountBlock.Meta, err = aca.store.GetBlockMeta(accountBlock.Hash)
		if err != nil {
			return nil, err
		}
	}
	if accountBlock.FromHash != nil && accountBlock.From == nil {
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

func (aca *AccountChainAccess) GetLatestBlockByAccountAddress(addr *types.Address) (*ledger.AccountBlock, error) {
	accountMeta, err := aca.accountStore.GetAccountMetaByAddress(addr)
	if err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	if accountMeta == nil {
		return nil, nil
	}

	return aca.store.GetLatestBlockByAccountId(accountMeta.AccountId)
}

func (aca *AccountChainAccess) GetBlockListByAccountAddress(index int, num int, count int, accountAddress *types.Address) ([]*ledger.AccountBlock, *types.GetError) {
	accountMeta, err := aca.accountStore.GetAccountMetaByAddress(accountAddress)
	if err != nil || accountMeta == nil {
		return nil, &types.GetError{
			Code: 1,
			Err:  errors.New("Query account meta failed."),
		}
	}

	blockList, err := aca.store.GetBlockListByAccountMeta(index, num, count, accountMeta)
	if err != nil {
		return nil, &types.GetError{
			Code: 2,
			Err:  errors.New("Query block list failed."),
		}
	}

	var processedBlockList = make([]*ledger.AccountBlock, len(blockList))
	for index, block := range blockList {
		processedBlockList[index], err = aca.processBlock(block, accountAddress)
		if err != nil {
			return nil, &types.GetError{
				Code: 3,
				Err:  errors.New("Process block list failed."),
			}
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
		if snapshotItem, ok := snapshotBlock.Snapshot[accountBlock.AccountAddress.String()]; ok {
			var itemAccountBlockMeta *ledger.AccountBlockMeta
			itemAccountBlockMeta, err = aca.store.GetBlockMeta(snapshotItem.AccountBlockHash)
			if itemAccountBlockMeta.Height.Cmp(accountBlock.Meta.Height) >= 0 {
				confirmSnapshotBlock = snapshotBlock
				return false
			}
		}
		return true
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

func (aca *AccountChainAccess) GetAccountBalance(accountId *big.Int, blockHeight *big.Int) (*big.Int, error) {
	accountBLock, err := aca.store.GetBlockByHeight(accountId, blockHeight)
	if err != nil {
		return nil, err
	}
	return accountBLock.Balance, nil
}

func (aca *AccountChainAccess) GetLatestBlockHeightByAccountId(accountId *big.Int) (*big.Int, error) {
	return aca.store.GetLatestBlockHeightByAccountId(accountId)
}

func (aca *AccountChainAccess) GetTotalNumber() (*big.Int, error) {
	return aca.store.CounterGet()
}

func (aca *AccountChainAccess) isBlockExist(blockHash *types.Hash) bool {
	return aca.store.IsBlockExist(blockHash)
}
