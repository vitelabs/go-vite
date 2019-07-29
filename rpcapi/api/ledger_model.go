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
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`
	PublicKey      []byte        `json:"publicKey"`

	Producer types.Address `json:"producer"`

	FromAddress   types.Address     `json:"fromAddress"`
	ToAddress     types.Address     `json:"toAddress"`
	FromBlockHash types.Hash        `json:"fromBlockHash"`
	TokenId       types.TokenTypeId `json:"tokenId"`
	Amount        *string           `json:"amount"`
	Fee           *string           `json:"fee"`

	Data []byte `json:"data"`

	Difficulty *string `json:"difficulty"`
	Nonce      []byte  `json:"nonce"`

	Signature []byte `json:"signature"`

	Quota         *string         `json:"quota"`
	QuotaUsed     *string         `json:"quotaUsed"`
	UtUsed        *string         `json:"utUsed"`
	LogHash       *types.Hash     `json:"logHash"`
	SendBlockList []*AccountBlock `json:"sendBlockList"`

	// extra info below
	TokenInfo *RpcTokenInfo `json:"tokenInfo"`

	ConfirmedTimes *string     `json:"confirmedTimes"`
	ConfirmedHash  *types.Hash `json:"confirmedHash"`

	ReceiveBlockHeight *string     `json:"receiveBlockHeight"`
	ReceiveBlockHash   *types.Hash `json:"receiveBlockHash"`

	Timestamp int64 `json:"timestamp"`
}

type SnapshotBlock struct {
	Producer types.Address `json:"producer"`
	*ledger.SnapshotBlock
	Timestamp int64 `json:"timestamp"`
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

	if block.Quota != nil {
		lAb.Quota, err = strconv.ParseUint(*block.Quota, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if block.QuotaUsed != nil {
		lAb.QuotaUsed, err = strconv.ParseUint(*block.QuotaUsed, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if !types.IsContractAddr(block.AccountAddress) {
		return lAb, nil
	}

	if len(block.SendBlockList) > 0 {
		subLAbList := make([]*ledger.AccountBlock, len(block.SendBlockList))
		for k, v := range block.SendBlockList {
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

	// TokenInfo
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
		block.ConfirmedHash = &confirmedBlock.Hash
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

	rpcBlock.Timestamp = sb.Timestamp.Unix()
	return rpcBlock, nil
}

func ledgerToRpcBlock(chain chain.Chain, lAb *ledger.AccountBlock) (*AccountBlock, error) {
	rpcBlock := &AccountBlock{
		BlockType:      lAb.BlockType,
		Hash:           lAb.Hash,
		PrevHash:       lAb.PrevHash,
		AccountAddress: lAb.AccountAddress,
		PublicKey:      lAb.PublicKey,
		FromBlockHash:  lAb.FromBlockHash,
		TokenId:        lAb.TokenId,

		Data:      lAb.Data,
		Signature: lAb.Signature,
		LogHash:   lAb.LogHash,

		Producer: lAb.Producer(),
	}
	//height
	rpcBlock.Height = strconv.FormatUint(lAb.Height, 10)

	// Quota & QuotaUsed
	totalQuota := strconv.FormatUint(lAb.Quota, 10)
	rpcBlock.Quota = &totalQuota
	quotaUsed := strconv.FormatUint(lAb.QuotaUsed, 10)
	rpcBlock.QuotaUsed = &quotaUsed
	utUsed := Float64ToString(float64(lAb.Quota)/float64(quota.QuotaForUtps), 4)
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
	TotalAmount string        `json:"totalAmount"`      // big int
	Number      *string       `json:"number,omitempty"` // uint64
}

type RpcTokenInfo struct {
	TokenName     string            `json:"tokenName"`
	TokenSymbol   string            `json:"tokenSymbol"`
	TotalSupply   *string           `json:"totalSupply,omitempty"` // *big.Int
	Decimals      uint8             `json:"decimals"`
	Owner         types.Address     `json:"owner"`
	TokenId       types.TokenTypeId `json:"tokenId"`
	MaxSupply     *string           `json:"maxSupply"` // *big.Int
	OwnerBurnOnly bool              `json:"ownerBurnOnly"`
	IsReIssuable  bool              `json:"isReIssuable"`
	Index         uint16            `json:"index"`
}

func RawTokenInfoToRpc(tinfo *types.TokenInfo, tti types.TokenTypeId) *RpcTokenInfo {
	var rt *RpcTokenInfo = nil
	if tinfo != nil {
		rt = &RpcTokenInfo{
			TokenName:     tinfo.TokenName,
			TokenSymbol:   tinfo.TokenSymbol,
			TotalSupply:   nil,
			Decimals:      tinfo.Decimals,
			Owner:         tinfo.Owner,
			TokenId:       tti,
			OwnerBurnOnly: tinfo.OwnerBurnOnly,
			IsReIssuable:  tinfo.IsReIssuable,
			Index:         tinfo.Index,
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

type TxParam interface {
	LedgerAccountBlock() (*ledger.AccountBlock, error)
}

type NormalRequestRawTxParam struct {
	BlockType byte       `json:"blockType"` // 1
	Height    string     `json:"height"`
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`
	PublicKey      []byte        `json:"publicKey"`

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
		BlockType:      param.BlockType,
		Hash:           param.Hash,
		PrevHash:       param.PrevHash,
		AccountAddress: param.AccountAddress,
		PublicKey:      param.PublicKey,
		ToAddress:      param.ToAddress,
		TokenId:        param.TokenId,
		Data:           param.Data,

		Signature: param.Signature,
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
