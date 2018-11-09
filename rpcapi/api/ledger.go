package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"strconv"
)

// !!! Block = Transaction = TX

func NewLedgerApi(vite *vite.Vite) *LedgerApi {
	return &LedgerApi{
		chain: vite.Chain(),
		//signer:        vite.Signer(),
		log: log15.New("module", "rpc_api/ledger_api"),
	}
}

type LedgerApi struct {
	chain chain.Chain
	log   log15.Logger
}

func (l LedgerApi) String() string {
	return "LedgerApi"
}

func (l *LedgerApi) ledgerBlockToRpcBlock(block *ledger.AccountBlock) (*AccountBlock, error) {
	return ledgerToRpcBlock(block, l.chain)
}

func (l *LedgerApi) ledgerBlocksToRpcBlocks(list []*ledger.AccountBlock) ([]*AccountBlock, error) {
	var blocks []*AccountBlock
	for _, item := range list {
		rpcBlock, err := l.ledgerBlockToRpcBlock(item)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, rpcBlock)
	}
	return blocks, nil
}

func (l *LedgerApi) GetBlockByHash(blockHash *types.Hash) (*AccountBlock, error) {
	block, getError := l.chain.GetAccountBlockByHash(blockHash)

	if getError != nil {
		l.log.Error("GetAccountBlockByHash failed, error is "+getError.Error(), "method", "GetBlockByHash")

		return nil, getError
	}
	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

func (l *LedgerApi) GetBlocksByHash(addr types.Address, originBlockHash *types.Hash, count uint64) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByHash")

	list, getError := l.chain.GetAccountBlocksByHash(addr, originBlockHash, count, false)
	if getError != nil {
		return nil, getError
	}

	if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
		l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetBlocksByHash")
		return nil, err
	} else {
		return blocks, nil
	}

}

type Statistics struct {
	SnapshotBlockCount uint64 `json:"snapshotBlockCount"`
	AccountBlockCount  uint64 `json:"accountBlockCount"`
}

func (l *LedgerApi) GetStatistics() (*Statistics, error) {
	latestSnapshotBlock := l.chain.GetLatestSnapshotBlock()
	allLatestAccountBlock, err := l.chain.GetAllLatestAccountBlock()

	if err != nil {
		return nil, err
	}
	var accountBlockCount uint64
	for _, block := range allLatestAccountBlock {
		accountBlockCount += block.Height
	}

	return &Statistics{
		SnapshotBlockCount: latestSnapshotBlock.Height,
		AccountBlockCount:  accountBlockCount,
	}, nil
}

func (l *LedgerApi) GetBlocksByAccAddr(addr types.Address, index int, count int) ([]*AccountBlock, error) {
	l.log.Info("GetBlocksByAccAddr")

	list, getErr := l.chain.GetAccountBlocksByAddress(&addr, index, 1, count)

	if getErr != nil {
		l.log.Info("GetBlocksByAccAddr", "err", getErr)
		return nil, getErr
	}

	if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
		l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetBlocksByAccAddr")
		return nil, err
	} else {
		return blocks, nil
	}
}

func (l *LedgerApi) GetAccountByAccAddr(addr types.Address) (*RpcAccountInfo, error) {
	l.log.Info("GetAccountByAccAddr")

	account, err := l.chain.GetAccount(&addr)
	if err != nil {
		l.log.Error("GetAccount failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	if account == nil {
		return nil, nil
	}

	latestAccountBlock, err := l.chain.GetLatestAccountBlock(&addr)
	if err != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	totalNum := uint64(0)
	if latestAccountBlock != nil {
		totalNum = latestAccountBlock.Height
	}

	balanceMap, err := l.chain.GetAccountBalance(&addr)
	if err != nil {
		l.log.Error("GetAccountBalance failed, error is "+err.Error(), "method", "GetAccountByAccAddr")
		return nil, err
	}

	tokenBalanceInfoMap := make(map[types.TokenTypeId]*RpcTokenBalanceInfo)
	for tokenId, amount := range balanceMap {
		token, _ := l.chain.GetTokenInfoById(&tokenId)
		tokenBalanceInfoMap[tokenId] = &RpcTokenBalanceInfo{
			TokenInfo:   RawTokenInfoToRpc(token, tokenId),
			TotalAmount: amount.String(),
			Number:      nil,
		}
	}

	rpcAccount := &RpcAccountInfo{
		AccountAddress:      account.AccountAddress,
		TotalNumber:         strconv.FormatUint(totalNum, 10),
		TokenBalanceInfoMap: tokenBalanceInfoMap,
	}

	return rpcAccount, nil
}

func (l *LedgerApi) GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error) {
	block, err := l.chain.GetSnapshotBlockByHash(&hash)
	if err != nil {
		l.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "GetSnapshotBlockByHash")
	}
	return block, err
}

func (l *LedgerApi) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	block, err := l.chain.GetSnapshotBlockByHeight(height)
	if err != nil {
		l.log.Error("GetSnapshotBlockByHash failed, error is "+err.Error(), "method", "GetSnapshotBlockByHeight")
	}
	return block, err
}

func (l *LedgerApi) GetSnapshotChainHeight() string {
	l.log.Info("GetLatestSnapshotChainHeight")
	return strconv.FormatUint(l.chain.GetLatestSnapshotBlock().Height, 10)
}

func (l *LedgerApi) GetLatestSnapshotChainHash() *types.Hash {
	l.log.Info("GetLatestSnapshotChainHash")
	return &l.chain.GetLatestSnapshotBlock().Hash
}

func (l *LedgerApi) GetLatestBlock(addr types.Address) (*AccountBlock, error) {
	l.log.Info("GetLatestBlock")
	block, getError := l.chain.GetLatestAccountBlock(&addr)
	if getError != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+getError.Error(), "method", "GetLatestBlock")
		return nil, getError
	}

	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

func (l *LedgerApi) GetTokenMintage(tti types.TokenTypeId) (*RpcTokenInfo, error) {
	l.log.Info("GetTokenMintage")
	if t, err := l.chain.GetTokenInfoById(&tti); err != nil {
		return nil, err
	} else {
		return RawTokenInfoToRpc(t, tti), nil
	}
}

func (l *LedgerApi) GetSenderInfo() (*KafkaSendInfo, error) {
	l.log.Info("GetSenderInfo")
	if l.chain.KafkaSender() == nil {
		return nil, nil
	}
	senderInfo := &KafkaSendInfo{}

	var totalErr error
	senderInfo.TotalEvent, totalErr = l.chain.GetLatestBlockEventId()
	if totalErr != nil {
		l.log.Error("GetLatestBlockEventId failed, error is "+totalErr.Error(), "method", "GetKafkaSenderInfo")

		return nil, totalErr
	}

	for _, producer := range l.chain.KafkaSender().Producers() {
		senderInfo.Producers = append(senderInfo.Producers, createKafkaProducerInfo(producer))
	}

	for _, producer := range l.chain.KafkaSender().RunProducers() {
		senderInfo.RunProducers = append(senderInfo.RunProducers, createKafkaProducerInfo(producer))
	}

	return senderInfo, nil
}

func (l *LedgerApi) GetBlockMeta(hash *types.Hash) (*ledger.AccountBlockMeta, error) {
	return l.chain.GetAccountBlockMetaByHash(hash)
}

func (l *LedgerApi) GetFittestSnapshotHash() (*types.Hash, error) {
	//latestBlock := l.chain.GetLatestSnapshotBlock()
	return generator.GetFitestGeneratorSnapshotHash(l.chain, nil)

	//gap := uint64(0)
	//targetHeight := latestBlock.Height
	//
	//if targetHeight > gap {
	//	targetHeight = latestBlock.Height - gap
	//} else {
	//	targetHeight = 1
	//}
	//
	//targetSnapshotBlock, err := l.chain.GetSnapshotBlockByHeight(targetHeight)
	//if err != nil {
	//	return nil, err
	//}
	//return &targetSnapshotBlock.Hash, nil

}

func (l *LedgerApi) GetNeedSnapshotContent() map[types.Address]*ledger.HashHeight {
	return l.chain.GetNeedSnapshotContent()
}

func (l *LedgerApi) SetSenderHasSend(producerId uint8, hasSend uint64) {
	l.log.Info("SetSenderHasSend")

	if l.chain.KafkaSender() == nil {
		return
	}
	l.chain.KafkaSender().SetHasSend(producerId, hasSend)
}

func (l *LedgerApi) StopSender(producerId uint8) {
	l.log.Info("StopSender")

	if l.chain.KafkaSender() == nil {
		return
	}
	l.chain.KafkaSender().StopById(producerId)
}
