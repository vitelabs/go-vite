package onroad_pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type TestChainDB struct {
}

func (db *TestChainDB) LoadOnRoad(gid types.Gid) (map[types.Address]map[types.Address][]ledger.HashHeight, error) {
	return nil, nil
}

func (db *TestChainDB) GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (db *TestChainDB) GetCompleteBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error) {
	return nil, nil
}

/*
func (p *contractOnRoadPool) InsertAccountBlock(block *ledger.AccountBlock) error {
	mlog := p.log.New("method", "WriteAccountBlock")
	isWrite := true

	if block.IsSendBlock() {
		orAddress := block.ToAddress
		caller := block.AccountAddress

		cc, exist := p.cache.Load(orAddress)
		if !exist || cc == nil {
			cc, _ = p.cache.LoadOrStore(orAddress, NewCallerCache())
		}

		isCallerContract := types.IsContractAddr(caller)
		if isCallerContract {
			return ErrBlockTypeErr
		}

		or := &ledger.hashHeight{
			Hash:   block.Hash,
			Height: block.Height,
		}
		if err := cc.(*callerCache).addTx(&caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("write block-s: %v -> %v %v %v", block.AccountAddress, block.ToAddress, block.Height, block.Hash),
				"err", err)
			panic(ErrAddTxFailed)
		}

	} else {
		// insert receive
		fromBlock, err := p.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if fromBlock == nil {
			return errors.New("failed to find send")
		}

		orAddress := fromBlock.ToAddress
		caller := fromBlock.AccountAddress

		cc, exist := p.cache.Load(orAddress)
		if !exist {
			mlog.Error(fmt.Sprintf("write block-r: %v %v %v fromHash=%v", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash),
				"err", ErrLoadCallerCacheFailed)
			panic(ErrLoadCallerCacheFailed)
		}

		isCallerContract := types.IsContractAddr(caller)
		or := &ledger.hashHeight{
			Hash:   fromBlock.Hash,
			Height: fromBlock.Height,
		}
		if isCallerContract {
			completeBlock, err := p.chain.GetCompleteBlockByHash(fromBlock.Hash)
			if err != nil {
				return err
			}
			if completeBlock == nil {
				return errors.New("failed to find complete send's parent receive")
			}
			or.Height = completeBlock.Height // refer to its parent receive's height
		}
		if err := cc.(*callerCache).rmTx(&caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("write block-r: %v %v %v fromHash=%v", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash),
				"err", err)
			panic(ErrRmTxFailed)
		}

		// insert sendBlockList
		for _, subSend := range block.SendBlockList {
			orAddress := subSend.ToAddress
			caller := subSend.AccountAddress

			isToAddrContract := types.IsContractAddr(orAddress)
			if !isToAddrContract {
				continue
			}
			cc, exist := p.cache.Load(orAddress)
			if !exist || cc == nil {
				cc, _ = p.cache.LoadOrStore(orAddress, NewCallerCache())
			}
			or := &ledger.hashHeight{
				Hash:   subSend.Hash,
				Height: block.Height, // refer to its parent receive's height
			}
			if err := cc.(*callerCache).addTx(&caller, true, or, isWrite); err != nil {
				mlog.Error(fmt.Sprintf("write block-s: %v -> %v %v %v", subSend.AccountAddress, subSend.ToAddress, block.Height, subSend.Hash),
					"err", err)
				panic(ErrAddTxFailed)
			}
		}
	}
	return nil
}

func (p *contractOnRoadPool) DeleteAccountBlock(block *ledger.AccountBlock) error {
	mlog := p.log.New("method", "DeleteAccountBlock")
	isWrite := false

	if block.IsSendBlock() {
		orAddress := block.ToAddress
		caller := block.AccountAddress

		cc, exist := p.cache.Load(orAddress)
		if !exist || cc == nil {
			mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", block.AccountAddress, block.ToAddress, block.Height, block.Hash),
				"err", ErrLoadCallerCacheFailed)
			panic(ErrLoadCallerCacheFailed)
		}

		isCallerContract := types.IsContractAddr(caller)
		if isCallerContract {
			return ErrBlockTypeErr
		}

		or := &ledger.hashHeight{
			Height: block.Height,
			Hash:   block.Hash,
		}
		if err := cc.(*callerCache).rmTx(&caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", block.AccountAddress, block.ToAddress, block.Height, block.Hash),
				"err", err)
			panic(ErrRmTxFailed)
		}

	} else {
		// revert receive
		fromBlock, err := p.chain.GetAccountBlockByHash(block.FromBlockHash)
		if err != nil {
			return err
		}
		if fromBlock == nil {
			return errors.New("failed to find send")
		}

		orAddress := fromBlock.ToAddress
		caller := fromBlock.AccountAddress

		cc, exist := p.cache.Load(orAddress)
		if !exist || cc == nil {
			cc, _ = p.cache.LoadOrStore(orAddress, NewCallerCache())
		}

		isCallerContract := types.IsContractAddr(caller)
		or := &ledger.hashHeight{
			Hash:   fromBlock.Hash,
			Height: fromBlock.Height,
		}
		if isCallerContract {
			completeBlock, err := p.chain.GetCompleteBlockByHash(fromBlock.Hash)
			if err != nil {
				return err
			}
			if completeBlock == nil {
				return errors.New("failed to find complete send's parent receive")
			}
			or.Height = completeBlock.Height // refer to its parent receive's height
		}
		if err := cc.(*callerCache).addTx(&caller, isCallerContract, or, isWrite); err != nil {
			mlog.Error(fmt.Sprintf("delete block-r: %v %v %v fromHash=%v", block.AccountAddress, block.Height, block.Hash, block.FromBlockHash),
				"err", err)
			panic(ErrAddTxFailed)
		}

		// revert sendBlockList
		for _, subSend := range block.SendBlockList {

			orAddress := subSend.ToAddress
			caller := subSend.AccountAddress

			isOrAddrContract := types.IsContractAddr(orAddress)
			if !isOrAddrContract {
				continue
			}
			cc, exist := p.cache.Load(orAddress)
			if !exist || cc == nil {
				mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", subSend.AccountAddress, subSend.ToAddress, block.Height, subSend.Hash),
					"err", ErrLoadCallerCacheFailed)
				panic(ErrLoadCallerCacheFailed)
			}
			or := &ledger.hashHeight{
				Hash:   subSend.Hash,
				Height: block.Height, // refer to its parent receive's height
			}
			if err := cc.(*callerCache).rmTx(&caller, true, or, isWrite); err != nil {
				mlog.Error(fmt.Sprintf("delete block-s: %v -> %v %v %v", subSend.AccountAddress, subSend.ToAddress, block.Height, subSend.Hash),
					"err", err)
				panic(ErrRmTxFailed)
			}
		}
	}
	return nil
}*/
