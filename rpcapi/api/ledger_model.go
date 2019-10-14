package api

import (
	"errors"
	"github.com/vitelabs/go-vite/vm/quota"
	"math/big"
	"strconv"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotChunk struct {
	AccountBlocks []*ledger.AccountBlock
	SnapshotBlock *SnapshotBlock
}

type AccountBlock struct {
	BlockType byte       `json:"blockType"`
	Height    string     `json:"height"`
	Hash      types.Hash `json:"hash"`

	PrevHash     types.Hash `json:"prevHash"`
	PreviousHash types.Hash `json:"previousHash"`

	AccountAddress types.Address `json:"accountAddress"`
	Address        types.Address `json:"address"`

	PublicKey []byte `json:"publicKey"`

	Producer types.Address `json:"producer"`

	FromAddress   types.Address `json:"fromAddress"`
	ToAddress     types.Address `json:"toAddress"`
	FromBlockHash types.Hash    `json:"fromBlockHash"`
	SendBlockHash types.Hash    `json:"sendBlockHash"`

	TokenId types.TokenTypeId `json:"tokenId"`
	Amount  *string           `json:"amount"`
	Fee     *string           `json:"fee"`

	Data []byte `json:"data"`

	Difficulty *string `json:"difficulty"`
	Nonce      []byte  `json:"nonce"`

	Signature []byte `json:"signature"`

	Quota        *string `json:"quota"`
	QuotaByStake *string `json:"quotaByStake"`

	QuotaUsed  *string `json:"quotaUsed"`
	TotalQuota *string `json:"totalQuota"`

	UtUsed    *string     `json:"utUsed"` // TODO: to remove
	LogHash   *types.Hash `json:"logHash"`
	VmLogHash *types.Hash `json:"vmLogHash"`

	SendBlockList          []*AccountBlock `json:"sendBlockList"`
	TriggeredSendBlockList []*AccountBlock `json:"triggeredSendBlockList"`

	// extra info below
	TokenInfo *RpcTokenInfo `json:"tokenInfo"`

	ConfirmedTimes *string `json:"confirmedTimes"`
	Confirmations  *string `json:"confirmations"`

	ConfirmedHash     *types.Hash `json:"confirmedHash"`
	FirstSnapshotHash *types.Hash `json:"firstSnapshotHash"`

	ReceiveBlockHeight *string     `json:"receiveBlockHeight"`
	ReceiveBlockHash   *types.Hash `json:"receiveBlockHash"`

	Timestamp int64 `json:"timestamp"`
}

type SnapshotBlock struct {
	Producer types.Address `json:"producer"`
	*ledger.SnapshotBlock

	PreviousHash types.Hash             `json:"previousHash"`
	NextSeedHash *types.Hash            `json:"nextSeedHash"`
	SnapshotData ledger.SnapshotContent `json:"snapshotData"`
	Timestamp    int64                  `json:"timestamp"`
}

func (block *AccountBlock) RpcToLedgerBlock() (*ledger.AccountBlock, error) {
	lAb := &ledger.AccountBlock{
		BlockType:      block.BlockType,
		Hash:           block.Hash,
		PrevHash:       block.PrevHash,
		AccountAddress: block.AccountAddress,
		PublicKey:      block.PublicKey,
		ToAddress:      block.ToAddress,

		FromBlockHash: block.FromBlockHash,
		TokenId:       block.TokenId,

		Data:      block.Data,
		Nonce:     block.Nonce,
		Signature: block.Signature,
		LogHash:   block.LogHash,
	}

	// set vm log hash
	if block.VmLogHash != nil {
		lAb.LogHash = block.VmLogHash
	}

	// set prev hash
	if !block.PreviousHash.IsZero() {
		lAb.PrevHash = block.PreviousHash
	}
	// set account address
	if !block.Address.IsZero() {
		lAb.AccountAddress = block.Address
	}
	// set from block hash
	if !block.SendBlockHash.IsZero() {
		lAb.FromBlockHash = block.SendBlockHash
	}

	var err error

	lAb.Height, err = strconv.ParseUint(block.Height, 10, 64)
	if err != nil {
		return nil, err
	}

	lAb.Amount = big.NewInt(0)
	if block.Amount != nil {
		if _, ok := lAb.Amount.SetString(*block.Amount, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	lAb.Fee = big.NewInt(0)
	if block.Fee != nil {
		if _, ok := lAb.Fee.SetString(*block.Fee, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	if block.Nonce != nil {
		if block.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			difficultyStr, ok := new(big.Int).SetString(*block.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = difficultyStr
		}
	}

	// quota
	var quotaStr *string
	if block.QuotaByStake != nil {
		quotaStr = block.QuotaByStake
	} else if block.Quota != nil {
		quotaStr = block.Quota
	}
	if quotaStr != nil {
		lAb.Quota, err = strconv.ParseUint(*quotaStr, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	// total quota
	var totalQuotaStr *string
	if block.TotalQuota != nil {
		totalQuotaStr = block.TotalQuota
	} else if block.QuotaUsed != nil {
		totalQuotaStr = block.QuotaUsed
	}
	if totalQuotaStr != nil {
		lAb.QuotaUsed, err = strconv.ParseUint(*totalQuotaStr, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if !types.IsContractAddr(block.AccountAddress) {
		return lAb, nil
	}

	var sendBlockList []*AccountBlock
	if len(block.TriggeredSendBlockList) > 0 {
		sendBlockList = block.TriggeredSendBlockList
	} else if len(block.SendBlockList) > 0 {
		sendBlockList = block.SendBlockList
	}

	if len(sendBlockList) > 0 {
		subLAbList := make([]*ledger.AccountBlock, len(sendBlockList))
		for k, v := range sendBlockList {
			subLAb, subErr := v.RpcToLedgerBlock()
			if subErr != nil {
				return nil, subErr
			}
			subLAbList[k] = subLAb
		}
		lAb.SendBlockList = subLAbList
	}

	return lAb, nil
}

func (block *AccountBlock) ComputeHash() (*types.Hash, error) {
	lAb, err := block.RpcToLedgerBlock()
	if err != nil {
		return nil, err
	}
	hash := lAb.ComputeHash()
	return &hash, nil
}

func (block *AccountBlock) addExtraInfo(chain chain.Chain) error {

	// RpcTokenInfo
	if block.TokenId != types.ZERO_TOKENID {
		token, _ := chain.GetTokenInfoById(block.TokenId)
		block.TokenInfo = RawTokenInfoToRpc(token, block.TokenId)
	}

	// ReceiveBlockHeight & ReceiveBlockHash
	if ledger.IsSendBlock(block.BlockType) {
		receiveBlock, err := chain.GetReceiveAbBySendAb(block.Hash)
		if err != nil {
			return err
		}
		if receiveBlock != nil {
			heightStr := strconv.FormatUint(receiveBlock.Height, 10)
			block.ReceiveBlockHeight = &heightStr
			block.ReceiveBlockHash = &receiveBlock.Hash
		}
	}

	// ConfirmedTimes & ConfirmedHash
	latestSb := chain.GetLatestSnapshotBlock()
	confirmedBlock, err := chain.GetConfirmSnapshotHeaderByAbHash(block.Hash)
	if err != nil {
		return err
	}
	if confirmedBlock != nil && latestSb != nil && confirmedBlock.Height <= latestSb.Height {
		confirmedTimeStr := strconv.FormatUint(latestSb.Height-confirmedBlock.Height+1, 10)

		block.ConfirmedTimes = &confirmedTimeStr
		block.Confirmations = &confirmedTimeStr

		block.ConfirmedHash = &confirmedBlock.Hash
		block.FirstSnapshotHash = &confirmedBlock.Hash

		block.Timestamp = confirmedBlock.Timestamp.Unix()
	}
	return nil
}

func ledgerSnapshotBlockToRpcBlock(sb *ledger.SnapshotBlock) (*SnapshotBlock, error) {
	if sb == nil {
		return nil, nil
	}
	rpcBlock := &SnapshotBlock{
		SnapshotBlock: sb,
	}

	rpcBlock.Producer = sb.Producer()

	rpcBlock.PreviousHash = sb.PrevHash
	rpcBlock.NextSeedHash = sb.SeedHash
	rpcBlock.SnapshotData = sb.SnapshotContent

	rpcBlock.Timestamp = sb.Timestamp.Unix()
	return rpcBlock, nil
}

func ledgerToRpcBlock(chain chain.Chain, lAb *ledger.AccountBlock) (*AccountBlock, error) {
	rpcBlock := &AccountBlock{
		BlockType:    lAb.BlockType,
		Hash:         lAb.Hash,
		PrevHash:     lAb.PrevHash,
		PreviousHash: lAb.PrevHash,

		AccountAddress: lAb.AccountAddress,
		Address:        lAb.AccountAddress,

		PublicKey:     lAb.PublicKey,
		FromBlockHash: lAb.FromBlockHash,
		SendBlockHash: lAb.FromBlockHash,

		TokenId: lAb.TokenId,

		Data:      lAb.Data,
		Signature: lAb.Signature,
		LogHash:   lAb.LogHash,
		VmLogHash: lAb.LogHash,

		Producer: lAb.Producer(),
	}
	//height
	rpcBlock.Height = strconv.FormatUint(lAb.Height, 10)

	// Quota & QuotaUsed
	totalQuota := strconv.FormatUint(lAb.Quota, 10)
	rpcBlock.Quota = &totalQuota
	rpcBlock.QuotaByStake = rpcBlock.Quota

	quotaUsed := strconv.FormatUint(lAb.QuotaUsed, 10)
	rpcBlock.QuotaUsed = &quotaUsed
	rpcBlock.TotalQuota = rpcBlock.QuotaUsed

	utUsed := Float64ToString(float64(lAb.QuotaUsed)/float64(quota.QuotaPerUt), 4)
	rpcBlock.UtUsed = &utUsed

	// FromAddress & ToAddress
	var amount, fee string
	if lAb.IsSendBlock() {
		rpcBlock.FromAddress = lAb.AccountAddress
		rpcBlock.ToAddress = lAb.ToAddress
		if lAb.Amount != nil {
			amount = lAb.Amount.String()
			rpcBlock.Amount = &amount
		}
		if lAb.Fee != nil {
			fee = lAb.Fee.String()
			rpcBlock.Fee = &fee
		}
	} else {
		sendBlock, err := chain.GetAccountBlockByHash(lAb.FromBlockHash)
		if err != nil {
			return nil, err
		}
		if sendBlock != nil {
			rpcBlock.FromAddress = sendBlock.AccountAddress
			rpcBlock.ToAddress = sendBlock.ToAddress

			rpcBlock.FromBlockHash = sendBlock.Hash
			rpcBlock.TokenId = sendBlock.TokenId
			if sendBlock.Amount != nil {
				amount = sendBlock.Amount.String()
				rpcBlock.Amount = &amount
			}
			if sendBlock.Fee != nil {
				fee = sendBlock.Fee.String()
				rpcBlock.Fee = &fee
			}
		}
	}

	// Difficulty & Nonce
	rpcBlock.Nonce = lAb.Nonce
	if lAb.Difficulty != nil {
		difficulty := lAb.Difficulty.String()
		rpcBlock.Difficulty = &difficulty
	}

	if err := rpcBlock.addExtraInfo(chain); err != nil {
		return nil, err
	}

	// SendBlockList
	if len(lAb.SendBlockList) > 0 {
		subBlockList := make([]*AccountBlock, len(lAb.SendBlockList))
		for k, v := range lAb.SendBlockList {
			subRpcTx, err := ledgerToRpcBlock(chain, v)
			if err != nil {
				return nil, err
			}
			subBlockList[k] = subRpcTx
		}
		rpcBlock.SendBlockList = subBlockList
		rpcBlock.TriggeredSendBlockList = subBlockList
	}

	return rpcBlock, nil
}

type RpcAccountInfo struct {
	AccountAddress      types.Address                              `json:"accountAddress"`
	TotalNumber         string                                     `json:"totalNumber"` // uint64
	TokenBalanceInfoMap map[types.TokenTypeId]*RpcTokenBalanceInfo `json:"tokenBalanceInfoMap,omitempty"`
}

type RpcTokenBalanceInfo struct {
	TokenInfo   *RpcTokenInfo `json:"tokenInfo,omitempty"`
	TotalAmount string        `json:"totalAmount"`
	Number      *string       `json:"number,omitempty"`
}

type AccountInfo struct {
	Address        types.Address                      `json:"address"`
	BlockCount     string                             `json:"blockCount"`
	BalanceInfoMap map[types.TokenTypeId]*BalanceInfo `json:"balanceInfoMap,omitempty"`
}

type BalanceInfo struct {
	TokenInfo        *RpcTokenInfo `json:"tokenInfo,omitempty"`
	Balance          string        `json:"balance"`                    // big int
	TransactionCount *string       `json:"transactionCount,omitempty"` // uint64
}

type RpcTokenInfo struct {
	TokenName     string            `json:"tokenName"`
	TokenSymbol   string            `json:"tokenSymbol"`
	TotalSupply   *string           `json:"totalSupply,omitempty"` // *big.Int
	Decimals      uint8             `json:"decimals"`
	Owner         types.Address     `json:"owner"`
	TokenId       types.TokenTypeId `json:"tokenId"`
	MaxSupply     *string           `json:"maxSupply"`     // *big.Int
	OwnerBurnOnly bool              `json:"ownerBurnOnly"` // Deprecated: use IsOwnerBurnOnly instead
	IsReIssuable  bool              `json:"isReIssuable"`
	Index         uint16            `json:"index"`

	// mainnet new
	IsOwnerBurnOnly bool `json:"isOwnerBurnOnly"`
}

type PagingQueryBatch struct {
	Address types.Address `json:"address"`

	PageNumber uint64 `json:"pageNumber"`
	PageCount  uint64 `json:"pageCount"`
}

func RawTokenInfoToRpc(tinfo *types.TokenInfo, tti types.TokenTypeId) *RpcTokenInfo {
	var rt *RpcTokenInfo = nil
	if tinfo != nil {
		rt = &RpcTokenInfo{
			TokenName:       tinfo.TokenName,
			TokenSymbol:     tinfo.TokenSymbol,
			TotalSupply:     nil,
			Decimals:        tinfo.Decimals,
			Owner:           tinfo.Owner,
			TokenId:         tti,
			OwnerBurnOnly:   tinfo.OwnerBurnOnly,
			IsOwnerBurnOnly: tinfo.OwnerBurnOnly,
			IsReIssuable:    tinfo.IsReIssuable,
			Index:           tinfo.Index,
		}
		if tinfo.TotalSupply != nil {
			s := tinfo.TotalSupply.String()
			rt.TotalSupply = &s
		}
		if tinfo.MaxSupply != nil {
			s := tinfo.MaxSupply.String()
			rt.MaxSupply = &s
		}
	}
	return rt
}

func ToRpcAccountInfo(chain chain.Chain, info *ledger.AccountInfo) *RpcAccountInfo {
	if info == nil {
		return nil
	}
	var r RpcAccountInfo
	r.AccountAddress = info.AccountAddress
	r.TotalNumber = strconv.FormatUint(info.TotalNumber, 10)
	r.TokenBalanceInfoMap = make(map[types.TokenTypeId]*RpcTokenBalanceInfo)

	for tti, v := range info.TokenBalanceInfoMap {
		if v != nil {
			tinfo, _ := chain.GetTokenInfoById(tti)
			if tinfo == nil {
				continue
			}
			b := &RpcTokenBalanceInfo{
				TokenInfo:   RawTokenInfoToRpc(tinfo, tti),
				TotalAmount: v.TotalAmount.String(),
			}
			if v.Number > 0 {
				number := strconv.FormatUint(v.Number, 10)
				b.Number = &number
			}
			r.TokenBalanceInfoMap[tti] = b
		}
	}
	return &r
}

func ToAccountInfo(chain chain.Chain, info *ledger.AccountInfo) *AccountInfo {
	if info == nil {
		return nil
	}
	var r AccountInfo
	r.Address = info.AccountAddress
	r.BlockCount = strconv.FormatUint(info.TotalNumber, 10)
	r.BalanceInfoMap = make(map[types.TokenTypeId]*BalanceInfo)

	for tti, v := range info.TokenBalanceInfoMap {
		if v != nil {
			tinfo, _ := chain.GetTokenInfoById(tti)
			if tinfo == nil {
				continue
			}
			b := &BalanceInfo{
				TokenInfo: RawTokenInfoToRpc(tinfo, tti),
				Balance:   v.TotalAmount.String(),
			}
			if v.Number > 0 {
				number := strconv.FormatUint(v.Number, 10)
				b.TransactionCount = &number
			}
			r.BalanceInfoMap[tti] = b
		}
	}
	return &r
}

type TxParam interface {
	LedgerAccountBlock() (*ledger.AccountBlock, error)
}

type NormalRequestRawTxParam struct {
	BlockType byte       `json:"blockType"` // 1
	Height    string     `json:"height"`
	Hash      types.Hash `json:"hash"`

	PrevHash     types.Hash `json:"prevHash"`
	PreviousHash types.Hash `json:"previousHash"`

	AccountAddress types.Address `json:"accountAddress"`
	Address        types.Address `json:"address"`

	PublicKey []byte `json:"publicKey"`

	ToAddress types.Address     `json:"toAddress"`
	TokenId   types.TokenTypeId `json:"tokenId"`
	Amount    string            `json:"amount"`

	Data []byte `json:"data"`

	Difficulty *string `json:"difficulty"`
	Nonce      []byte  `json:"nonce"`

	Signature []byte `json:"signature"`
}

func (param NormalRequestRawTxParam) LedgerAccountBlock() (*ledger.AccountBlock, error) {
	if types.IsContractAddr(param.AccountAddress) {
		return nil, errors.New("can't send tx for the contract")
	}

	lAb := &ledger.AccountBlock{
		BlockType: param.BlockType,
		Hash:      param.Hash,
		PrevHash:  param.PrevHash,

		AccountAddress: param.AccountAddress,
		PublicKey:      param.PublicKey,
		ToAddress:      param.ToAddress,
		TokenId:        param.TokenId,
		Data:           param.Data,

		Signature: param.Signature,
	}

	if !param.PreviousHash.IsZero() {
		lAb.PrevHash = param.PreviousHash
	}

	if !param.Address.IsZero() {
		lAb.AccountAddress = param.Address
	}

	var err error

	lAb.Height, err = strconv.ParseUint(param.Height, 10, 64)
	if err != nil {
		return nil, err
	}

	lAb.Amount = big.NewInt(0)
	if _, ok := lAb.Amount.SetString(param.Amount, 10); !ok {
		return nil, ErrStrToBigInt
	}

	if param.Nonce != nil {
		if param.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			difficultyStr, ok := new(big.Int).SetString(*param.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = difficultyStr
			lAb.Nonce = param.Nonce
		}
	}
	return lAb, nil
}
