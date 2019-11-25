package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/plugins"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts/dex"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

// new api
func (l *LedgerApi) GetAccountBlocks(addr types.Address, originBlockHash *types.Hash, tokenTypeId *types.TokenTypeId, count uint64) ([]*AccountBlock, error) {
	if tokenTypeId == nil {
		if originBlockHash == nil {
			block, err := l.chain.GetLatestAccountBlock(addr)
			if err != nil {
				return nil, err
			}
			if block != nil {
				originBlockHash = &block.Hash
			}
		}

		if originBlockHash == nil {
			return nil, nil
		}

		list, getError := l.chain.GetAccountBlocks(*originBlockHash, count)
		if getError != nil {
			return nil, getError
		}

		if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
			l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetAccountBlocks")
			return nil, err
		} else {
			return blocks, nil
		}
	} else {
		if count == 0 {
			return nil, nil
		}
		plugins := l.chain.Plugins()
		if plugins == nil {
			err := errors.New("config.OpenPlugins is false, api can't work")
			return nil, err
		}

		plugin := plugins.GetPlugin("filterToken").(*chain_plugins.FilterToken)

		blocks, err := plugin.GetBlocks(addr, *tokenTypeId, originBlockHash, count)
		if err != nil {
			return nil, err
		}

		return l.ledgerBlocksToRpcBlocks(blocks)
	}
}

// new api
func (l *LedgerApi) GetAccountBlockByHash(blockHash types.Hash) (*AccountBlock, error) {
	block, getError := l.chain.GetAccountBlockByHash(blockHash)
	if getError != nil {
		l.log.Error("GetAccountBlockByHash failed, error is "+getError.Error(), "method", "GetAccountBlockByHash")

		return nil, getError
	}
	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

// new api
func (l *LedgerApi) GetAccountBlockByHeight(addr types.Address, height interface{}) (*AccountBlock, error) {
	heightUint64, err := parseHeight(height)
	if err != nil {
		return nil, err
	}

	accountBlock, err := l.chain.GetAccountBlockByHeight(addr, heightUint64)
	if err != nil {
		l.log.Error("GetAccountBlockByHeight failed, error is "+err.Error(), "method", "GetAccountBlockByHeight")
		return nil, err
	}

	if accountBlock == nil {
		return nil, nil
	}
	return l.ledgerBlockToRpcBlock(accountBlock)
}

// new api
func (l *LedgerApi) GetAccountBlocksByAddress(addr types.Address, index int, count int) ([]*AccountBlock, error) {
	l.log.Info("GetAccountBlocksByAddress")

	height, err := l.chain.GetLatestAccountHeight(addr)
	if err != nil {
		l.log.Error(fmt.Sprintf("GetLatestAccountHeight, addr is %s", addr), "err", err, "method", "GetAccountBlocksByAddress")
		return nil, err
	}

	num := uint64(index * count)
	if height < num {
		return nil, nil
	}

	list, getErr := l.chain.GetAccountBlocksByHeight(addr, height-num, uint64(count))

	if getErr != nil {
		l.log.Info("GetBlocksByAccAddr", "err", getErr, "method", "GetAccountBlocksByAddress")
		return nil, getErr
	}

	if blocks, err := l.ledgerBlocksToRpcBlocks(list); err != nil {
		l.log.Error("GetConfirmTimes failed, error is "+err.Error(), "method", "GetAccountBlocksByAddress")
		return nil, err
	} else {
		return blocks, nil
	}
}

// new api
func (l *LedgerApi) GetAccountInfoByAddress(addr types.Address) (*AccountInfo, error) {
	l.log.Info("GetAccountInfoByAddress")

	info, err := l.getAccountInfoByAddress(addr)
	if err != nil {
		return nil, err
	}
	return ToAccountInfo(l.chain, info), nil
}

func (l *LedgerApi) getAccountInfoByAddress(addr types.Address) (*ledger.AccountInfo, error) {
	latestAccountBlock, err := l.chain.GetLatestAccountBlock(addr)
	if err != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+err.Error(), "method", "GetAccountInfoByAddress")
		return nil, err
	}

	totalNum := uint64(0)
	if latestAccountBlock != nil {
		totalNum = latestAccountBlock.Height
	}

	balanceMap, err := l.chain.GetBalanceMap(addr)
	if err != nil {
		l.log.Error("GetAccountBalance failed, error is "+err.Error(), "method", "GetAccountInfoByAddress")
		return nil, err
	}

	tokenBalanceInfoMap := make(map[types.TokenTypeId]*ledger.TokenBalanceInfo)
	for tokenId, amount := range balanceMap {
		token, _ := l.chain.GetTokenInfoById(tokenId)
		if token == nil {
			continue
		}
		totalAmount := *big.NewInt(0)
		if amount != nil {
			totalAmount = *amount
		}
		tokenBalanceInfoMap[tokenId] = &ledger.TokenBalanceInfo{
			TotalAmount: totalAmount,
			Number:      0,
		}
	}
	return &ledger.AccountInfo{
		AccountAddress:      addr,
		TotalNumber:         totalNum,
		TokenBalanceInfoMap: tokenBalanceInfoMap,
	}, nil
}

// new api
func (l *LedgerApi) GetLatestSnapshotHash() *types.Hash {
	l.log.Info("GetLatestSnapshotHash")
	return &l.chain.GetLatestSnapshotBlock().Hash
}

// new api
func (l *LedgerApi) GetLatestAccountBlock(addr types.Address) (*AccountBlock, error) {
	l.log.Info("GetLatestAccountBlock")
	block, getError := l.chain.GetLatestAccountBlock(addr)
	if getError != nil {
		l.log.Error("GetLatestAccountBlock failed, error is "+getError.Error(), "method", "GetLatestAccountBlock")
		return nil, getError
	}

	if block == nil {
		return nil, nil
	}

	return l.ledgerBlockToRpcBlock(block)
}

// new api
func (l *LedgerApi) GetVmLogs(blockHash types.Hash) (ledger.VmLogList, error) {
	block, err := l.chain.GetAccountBlockByHash(blockHash)
	if block == nil {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("get block failed")
	}

	return l.chain.GetVmLogList(block.LogHash)
}

// new api
func (l *LedgerApi) SendRawTransaction(block *AccountBlock) error {

	if block == nil {
		return errors.New("empty block")
	}
	if !checkTxToAddressAvailable(block.ToAddress) {
		return errors.New("ToAddress is invalid")
	}
	lb, err := block.RpcToLedgerBlock()
	if err != nil {
		return err
	}
	if err := checkTokenIdValid(l.chain, &lb.TokenId); err != nil {
		return err
	}
	latestSb := l.chain.GetLatestSnapshotBlock()
	if latestSb == nil {
		return errors.New("failed to get latest snapshotBlock")
	}
	if err := checkSnapshotValid(latestSb); err != nil {
		return err
	}
	if lb.ToAddress == types.AddressDexFund && !dex.VerifyNewOrderPriceForRpc(lb.Data) {
		return dex.InvalidOrderPriceErr
	}

	result, err := l.vite.Verifier().VerifyRPCAccountBlock(lb, latestSb)
	if err != nil {
		return err
	}

	if result != nil {
		return l.vite.Pool().AddDirectAccountBlock(result.AccountBlock.AccountAddress, result)
	} else {
		return errors.New("generator gen an empty block")
	}
}

// new api: ledger_getUnreceivedBlocksByAddress <- onroad_getOnroadBlocksByAddress
func (l *LedgerApi) GetUnreceivedBlocksByAddress(address types.Address, index, count uint64) ([]*AccountBlock, error) {

	log.Info("GetUnreceivedBlocksByAddress", "addr", address, "index", index, "count", count)

	blockList, err := l.chain.GetOnRoadBlocksByAddr(address, int(index), int(count))
	if err != nil {
		return nil, err
	}
	a := make([]*AccountBlock, len(blockList))
	sum := 0
	for _, v := range blockList {
		if v != nil {
			accountBlock, e := ledgerToRpcBlock(l.chain, v)
			if e != nil {
				return nil, e
			}
			a[sum] = accountBlock
			sum++
		}
	}
	return a[:sum], nil
}

// new api: ledger_getUnreceivedTransactionSummaryByAddress <- onroad_getOnroadInfoByAddress
func (l *LedgerApi) GetUnreceivedTransactionSummaryByAddress(address types.Address) (*AccountInfo, error) {

	log.Info("GetUnreceivedTransactionSummaryByAddress", "addr", address)

	info, e := l.chain.GetAccountOnRoadInfo(address)
	if e != nil || info == nil {
		return nil, e
	}
	return ToAccountInfo(l.chain, info), nil
}

// new api: ledger_getUnreceivedBlocksInBatch <- onroad_getOnroadBlocksInBatch
func (l *LedgerApi) GetUnreceivedBlocksInBatch(queryList []PagingQueryBatch) (map[types.Address][]*AccountBlock, error) {
	resultMap := make(map[types.Address][]*AccountBlock)
	for _, q := range queryList {
		if l, ok := resultMap[q.Address]; ok && l != nil {
			continue
		}
		blockList, err := l.GetUnreceivedBlocksByAddress(q.Address, q.PageNumber, q.PageCount)
		if err != nil {
			return nil, err
		}
		if len(blockList) <= 0 {
			continue
		}
		resultMap[q.Address] = blockList
	}
	return resultMap, nil
}

// new api:  ledger_getUnreceivedTransactionSummaryInBatch <-  onroad_getOnroadInfoInBatch
func (l *LedgerApi) GetUnreceivedTransactionSummaryInBatch(addressList []types.Address) ([]*AccountInfo, error) {

	// Remove duplicate
	addrMap := make(map[types.Address]bool, 0)
	for _, v := range addressList {
		addrMap[v] = true
	}

	resultList := make([]*AccountInfo, 0)
	for addr, _ := range addrMap {
		info, err := l.GetUnreceivedTransactionSummaryByAddress(addr)
		if err != nil {
			return nil, err
		}
		if info == nil {
			continue
		}
		resultList = append(resultList, info)
	}
	return resultList, nil
}

type VmLogFilterParam struct {
	AddrRange map[string]*Range `json:"addressHeightRange"`
	Topics    [][]types.Hash    `json:"topics"`
}
type Range struct {
	FromHeight string `json:"fromHeight"`
	ToHeight   string `json:"toHeight"`
}

func (r *Range) ToHeightRange() (*HeightRange, error) {
	if r != nil {
		fromHeight, err := StringToUint64(r.FromHeight)
		if err != nil {
			return nil, err
		}
		toHeight, err := StringToUint64(r.ToHeight)
		if err != nil {
			return nil, err
		}
		if toHeight < fromHeight && toHeight != 0 {
			return nil, errors.New("to height < from height")
		}
		return &HeightRange{fromHeight, toHeight}, nil
	}
	return nil, nil
}

type FilterParam struct {
	AddrRange map[types.Address]HeightRange
	Topics    [][]types.Hash
}
type HeightRange struct {
	FromHeight uint64
	ToHeight   uint64
}

func ToFilterParam(rangeMap map[string]*Range, topics [][]types.Hash) (*FilterParam, error) {
	var addrRange map[types.Address]HeightRange
	if len(rangeMap) == 0 {
		return nil, errors.New("addressHeightRange is nil")
	}
	addrRange = make(map[types.Address]HeightRange, len(rangeMap))
	for hexAddr, r := range rangeMap {
		hr, err := r.ToHeightRange()
		if err != nil {
			return nil, err
		}
		if hr == nil {
			hr = &HeightRange{0, 0}
		}
		addr, err := types.HexToAddress(hexAddr)
		if err != nil {
			return nil, err
		}
		addrRange[addr] = *hr
	}
	target := &FilterParam{
		AddrRange: addrRange,
		Topics:    topics,
	}
	return target, nil
}

type Logs struct {
	Log              *ledger.VmLog  `json:"vmlog"`
	AccountBlockHash types.Hash     `json:"accountBlockHash"`
	AccountHeight    string         `json:"accountBlockHeight"`
	Addr             *types.Address `json:"address"`
}

func (l *LedgerApi) GetVmLogsByFilter(param VmLogFilterParam) ([]*Logs, error) {
	return GetLogs(l.chain, param.AddrRange, param.Topics)
}
func GetLogs(c chain.Chain, rangeMap map[string]*Range, topics [][]types.Hash) ([]*Logs, error) {
	filterParam, err := ToFilterParam(rangeMap, topics)
	if err != nil {
		return nil, err
	}
	var logs []*Logs
	for addr, hr := range filterParam.AddrRange {
		startHeight := hr.FromHeight
		endHeight := hr.ToHeight
		acc, err := c.GetLatestAccountBlock(addr)
		if err != nil {
			return nil, err
		}
		if acc == nil {
			continue
		}
		if startHeight == 0 {
			startHeight = 1
		}
		if endHeight == 0 || endHeight > acc.Height {
			endHeight = acc.Height
		}
		for {
			offset, count, finish := getHeightPage(startHeight, endHeight, 100)
			if count == 0 {
				break
			}
			startHeight = offset + 1
			blocks, err := c.GetAccountBlocksByHeight(addr, offset, count)
			if err != nil {
				return nil, err
			}
			for i := len(blocks); i > 0; i-- {
				if blocks[i-1].LogHash != nil {
					list, err := c.GetVmLogList(blocks[i-1].LogHash)
					if err != nil {
						return nil, err
					}
					for _, l := range list {
						if FilterLog(filterParam, l) {
							logs = append(logs, &Logs{l, blocks[i-1].Hash, Uint64ToString(blocks[i-1].Height), &addr})
						}
					}
				}
			}
			if finish {
				break
			}
		}
	}
	return logs, nil
}

func getHeightPage(start uint64, end uint64, count uint64) (uint64, uint64, bool) {
	gap := end - start + 1
	if gap <= count {
		return end, gap, true
	}
	return start + count - 1, count, false
}
func FilterLog(filter *FilterParam, l *ledger.VmLog) bool {
	if len(l.Topics) < len(filter.Topics) {
		return false
	}
	for i, topicRange := range filter.Topics {
		if len(topicRange) == 0 {
			continue
		}
		flag := false
		for _, topic := range topicRange {
			if topic == l.Topics[i] {
				flag = true
				continue
			}
		}
		if !flag {
			return false
		}
	}
	return true
}

type GetPoWDifficultyParam struct {
	SelfAddr  types.Address  `json:"address"`
	PrevHash  types.Hash     `json:"previousHash"`
	BlockType byte           `json:"blockType"`
	ToAddr    *types.Address `json:"toAddress"`
	Data      []byte         `json:"data"`
	Multiple  uint16         `json:"congestionMultiplier"`
}

type GetPoWDifficultyResult struct {
	Quota        string `json:"requiredQuota"`
	Difficulty   string `json:"difficulty"`
	Qc           string `json:"qc"`
	IsCongestion bool   `json:"isCongestion"`
}

var multipleDivision = big.NewInt(10)

func (t LedgerApi) GetPoWDifficulty(param GetPoWDifficultyParam) (*GetPoWDifficultyResult, error) {
	result, err := calcPoWDifficulty(t.chain,
		CalcPoWDifficultyParam{param.SelfAddr, param.PrevHash, param.BlockType, param.ToAddr, param.Data, true, param.Multiple})
	if err != nil {
		return nil, err
	}
	return &GetPoWDifficultyResult{result.QuotaRequired, result.Difficulty, *result.Qc, result.IsCongestion}, nil
}

func calcPoWDifficulty(c chain.Chain, param CalcPoWDifficultyParam) (result *CalcPoWDifficultyResult, err error) {
	latestBlock, err := c.GetLatestAccountBlock(param.SelfAddr)
	if err != nil {
		return nil, err
	}
	if (latestBlock == nil && !param.PrevHash.IsZero()) ||
		(latestBlock != nil && latestBlock.Hash != param.PrevHash) {
		return nil, util.ErrChainForked
	}
	// get quota required
	block := &ledger.AccountBlock{
		BlockType:      param.BlockType,
		AccountAddress: param.SelfAddr,
		PrevHash:       param.PrevHash,
		Data:           param.Data,
	}
	if param.ToAddr != nil {
		block.ToAddress = *param.ToAddr
	} else if param.BlockType == ledger.BlockTypeSendCall {
		return nil, errors.New("toAddress is nil")
	}
	sb := c.GetLatestSnapshotBlock()
	db, err := vm_db.NewVmDb(c, &param.SelfAddr, &sb.Hash, &param.PrevHash)
	if err != nil {
		return nil, err
	}
	quotaRequired, err := vm.GasRequiredForBlock(db, block, util.QuotaTableByHeight(sb.Height), sb.Height)
	if err != nil {
		return nil, err
	}

	qc, _, isCongestion := quota.CalcQc(db, sb.Height)

	// get current quota
	var stakeAmount *big.Int
	var q types.Quota
	if param.UseStakeQuota {
		stakeAmount, err = c.GetStakeBeneficialAmount(param.SelfAddr)
		if err != nil {
			return nil, err
		}
		q, err := quota.GetQuota(db, param.SelfAddr, stakeAmount, sb.Height)
		if err != nil {
			return nil, err
		}
		if q.Current() >= quotaRequired {
			return &CalcPoWDifficultyResult{quotaRequired, Uint64ToString(quotaRequired), "", Float64ToString(float64(quotaRequired)/float64(quota.QuotaPerUt), 4), bigIntToString(qc), isCongestion}, nil
		}
	} else {
		stakeAmount = big.NewInt(0)
		q = types.NewQuota(0, 0, 0, 0, false, 0)
	}
	// calc difficulty if current quota is not enough
	canPoW := quota.CanPoW(db, block.AccountAddress)
	if !canPoW {
		return nil, util.ErrCalcPoWTwice
	}
	d, err := quota.CalcPoWDifficulty(db, quotaRequired, q, sb.Height)
	if err != nil {
		return nil, err
	}
	if isCongestion && param.Multiple > uint16(multipleDivision.Uint64()) {
		d.Mul(d, multipleDivision)
		d.Div(d, big.NewInt(int64(param.Multiple)))
	}
	return &CalcPoWDifficultyResult{quotaRequired, Uint64ToString(quotaRequired), d.String(), Float64ToString(float64(quotaRequired)/float64(quota.QuotaPerUt), 4), bigIntToString(qc), isCongestion}, nil
}

type GetQuotaRequiredParam struct {
	SelfAddr  types.Address  `json:"address"`
	BlockType byte           `json:"blockType"`
	ToAddr    *types.Address `json:"toAddress"`
	Data      []byte         `json:"data"`
}
type GetQuotaRequiredResult struct {
	QuotaRequired string `json:"requiredQuota"`
}

func (t LedgerApi) GetRequiredQuota(param GetQuotaRequiredParam) (*GetQuotaRequiredResult, error) {
	result, err := calcQuotaRequired(t.chain,
		CalcQuotaRequiredParam{param.SelfAddr, param.BlockType, param.ToAddr, param.Data})
	if err != nil {
		return nil, err
	}
	return &GetQuotaRequiredResult{result.QuotaRequired}, nil
}
func calcQuotaRequired(c chain.Chain, param CalcQuotaRequiredParam) (*CalcQuotaRequiredResult, error) {
	latestBlock, err := c.GetLatestAccountBlock(param.SelfAddr)
	if err != nil {
		return nil, err
	}
	prevHash := types.Hash{}
	if latestBlock != nil {
		prevHash = latestBlock.Hash
	}
	// get quota required
	block := &ledger.AccountBlock{
		BlockType:      param.BlockType,
		AccountAddress: param.SelfAddr,
		Data:           param.Data,
	}
	if param.ToAddr != nil {
		block.ToAddress = *param.ToAddr
	} else if param.BlockType == ledger.BlockTypeSendCall {
		return nil, errors.New("toAddress is nil")
	}
	sb := c.GetLatestSnapshotBlock()
	db, err := vm_db.NewVmDb(c, &param.SelfAddr, &sb.Hash, &prevHash)
	if err != nil {
		return nil, err
	}
	quotaRequired, err := vm.GasRequiredForBlock(db, block, util.QuotaTableByHeight(sb.Height), sb.Height)
	if err != nil {
		return nil, err
	}
	return &CalcQuotaRequiredResult{Uint64ToString(quotaRequired), Float64ToString(float64(quotaRequired)/float64(quota.QuotaPerUt), 4)}, nil
}
