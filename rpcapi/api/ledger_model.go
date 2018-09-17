package api

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type AccountBlockMeta struct {
	AccountId     *string `json:"accountId,omitempty"`
	Height        *string `json:"height,omitempty"`
	Status        int     `json:"status"`
	IsSnapshotted bool    `json:"isSnapshotted"`
}

type AccountBlock struct {
	Meta                   *AccountBlockMeta  `json:"meta,omitempty"`                   // if the block is generate by the client, it must not be nil and the Height in it must be validate. if the block is a response it maybe be nil
	AccountAddress         *types.Address     `json:"accountAddress,omitempty"`         // Self account
	PublicKey              string             `json:"publicKey,omitempty"`              // hex ed25519 public key string
	To                     *types.Address     `json:"to,omitempty"`                     // Receiver account, exists in send block
	From                   *types.Address     `json:"from,omitempty"`                   // [Optional] Sender account, exists in receive block
	FromHash               *types.Hash        `json:"fromHash,omitempty"`               // hex string. Correlative send block hash, exists in receive block
	PrevHash               *types.Hash        `json:"prevHash,omitempty"`               // Last block hash
	Hash                   *types.Hash        `json:"hash,omitempty"`                   // Block hash
	Balance                *string            `json:"balance,omitempty"`                // bigint. Balance of current account. if the block is generate by the client, the balance can be empty
	Amount                 *string            `json:"amount,omitempty"`                 // bigint. Amount of this transaction
	Timestamp              uint64             `json:"timestamp"`                        // Timestamp second
	TokenId                *types.TokenTypeId `json:"tokenId,omitempty"`                // Id of token received or sent
	LastBlockHeightInToken *string            `json:"lastBlockHeightInToken,omitempty"` // // [Optional] Height of last transaction block in this token. if the block is generate by the client it can be nil
	Data                   *string            `json:"data,omitempty"`                   // Data requested or repsonsed
	SnapshotTimestamp      *types.Hash        `json:"snapshotTimestamp,omitempty"`      // Snapshot timestamp second
	Signature              string             `json:"signature"`                        // Signature of current block
	Nonce                  string             `json:"nonce"`                            // PoW nounce
	Difficulty             string             `json:"difficulty"`                       // PoW difficulty
	FAmount                *string            `json:"fAmount,omitempty"`                // bigint. Service fee
	ConfirmedTimes         *string            `json:"confirmedTimes,omitempty"`         // bigint block`s confirmed times
}

func (ra *AccountBlock) ToLedgerAccBlock() (*ledger.AccountBlock, error) {

	PublicKey, e := hex.DecodeString(ra.PublicKey)
	if e != nil {
		log.Error("ToLedgerAccBlock decode PublicKey", "err", e)
		return nil, e
	}
	Signature, e := hex.DecodeString(ra.Signature)
	if e != nil {
		log.Error("ToLedgerAccBlock decode Signature ", "err", e)
		return nil, e
	}
	Nonce, e := hex.DecodeString(ra.Nonce)
	if e != nil {
		log.Error("ToLedgerAccBlock decode Nonce ", "err", e)
		return nil, e
	}
	Difficulty, e := hex.DecodeString(ra.Difficulty)
	if e != nil {
		log.Error("ToLedgerAccBlock decode Difficulty ", "err", e)
		return nil, e
	}

	var lam *ledger.AccountBlockMeta
	lam = nil
	if ra.Meta != nil {
		lam = &ledger.AccountBlockMeta{
			Height: stringToBigInt(ra.Meta.Height),
		}
	}
	Data := ""
	if ra.Data != nil {
		Data = *ra.Data
	}

	la := ledger.AccountBlock{
		Meta:                   lam,
		AccountAddress:         ra.AccountAddress,
		PublicKey:              PublicKey,
		To:                     ra.To,
		From:                   ra.From,
		FromHash:               ra.FromHash,
		PrevHash:               ra.PrevHash,
		Hash:                   ra.Hash,
		Balance:                stringToBigInt(ra.Balance),
		Amount:                 stringToBigInt(ra.Amount),
		Timestamp:              ra.Timestamp,
		TokenId:                ra.TokenId,
		LastBlockHeightInToken: stringToBigInt(ra.LastBlockHeightInToken),
		Data:                   Data,
		SnapshotTimestamp:      ra.SnapshotTimestamp,
		Signature:              Signature,
		Nounce:                 Nonce,
		Difficulty:             Difficulty,
		FAmount:                stringToBigInt(ra.FAmount),
	}

	return &la, nil
}

func LedgerAccBlocksToRpcAccBlocks(lists ledger.AccountBlockList, l *LedgerApi) []AccountBlock {
	simpleBlocks := make([]AccountBlock, len(lists))
	for i, v := range lists {

		times := l.getBlockConfirmedTimes(v)
		block := LedgerAccBlockToRpc(v, times)
		simpleBlocks[i] = *block
	}
	return simpleBlocks
}

func LedgerAccBlockToRpc(lb *ledger.AccountBlock, confirmedTime *string) *AccountBlock {
	if lb == nil {
		return nil
	}
	ra := AccountBlock{
		LastBlockHeightInToken: bigIntToString(lb.LastBlockHeightInToken),
		AccountAddress:         lb.AccountAddress,
		To:                     lb.To,
		From:                   lb.From,
		FromHash:               lb.FromHash,
		PrevHash:               lb.PrevHash,
		Balance:                bigIntToString(lb.Balance),
		Amount:                 bigIntToString(lb.Amount),
		Timestamp:              lb.Timestamp,
		TokenId:                lb.TokenId,
		Data:                   &lb.Data,
		SnapshotTimestamp:      lb.SnapshotTimestamp,
		FAmount:                bigIntToString(lb.FAmount),
		Hash:                   lb.Hash,
		PublicKey:              hex.EncodeToString(lb.PublicKey),
		Signature:              hex.EncodeToString(lb.Signature),
		Nonce:                  hex.EncodeToString(lb.Nounce),
		Difficulty:             hex.EncodeToString(lb.Difficulty),

		ConfirmedTimes: confirmedTime,
	}

	if lb.Meta != nil {
		ra.Meta = &AccountBlockMeta{
			Height:        bigIntToString(lb.Meta.Height),
			Status:        lb.Meta.Status,
			IsSnapshotted: lb.Meta.IsSnapshotted,
		}
	}

	return &ra
}

// Send tx parms
type SendTxParms struct {
	SelfAddr    types.Address     `json:"selfAddr"`    // who sends the tx
	ToAddr      types.Address     `json:"toAddr"`      // who receives the tx
	TokenTypeId types.TokenTypeId `json:"tokenTypeId"` // which token will be sent
	Passphrase  string            `json:"passphrase"`  // sender`s passphrase
	Amount      string            `json:"amount"`      // the amount of specific token will be sent. bigInt
}

type BalanceInfo struct {
	TokenSymbol string            `json:"tokenSymbol"` // token symbol example  1200 (symbol)
	TokenName   string            `json:"tokenName"`   // token name
	TokenTypeId types.TokenTypeId `json:"tokenTypeId"`
	Balance     string            `json:"balance"`
}

type GetAccountResponse struct {
	Addr         types.Address `json:"addr"`         // Account address
	BalanceInfos []BalanceInfo `json:"balanceInfos"` // Account Balance Infos
	BlockHeight  string        `json:"blockHeight"`  // Account BlockHeight also represents all blocks belong to the account. bigInt.
}

type GetUnconfirmedInfoResponse struct {
	Addr                 types.Address `json:"addr"`                 // Account address
	BalanceInfos         []BalanceInfo `json:"balanceInfos"`         // Account unconfirmed BalanceInfos (In-transit money)
	UnConfirmedBlocksLen string        `json:"unConfirmedBlocksLen"` // the length of unconfirmed blocks. bigInt
}

type InitSyncResponse struct {
	StartHeight      string `json:"startHeight"`      // bigInt. where we start sync
	TargetHeight     string `json:"targetHeight"`     // bigInt. when CurrentHeight == TargetHeight means that sync complete
	CurrentHeight    string `json:"currentHeight"`    // bigInt.
	IsFirstSyncDone  bool   `json:"isFirstSyncDone"`  // true means sync complete
	IsStartFirstSync bool   `json:"isStartFirstSync"` // true means sync start
}

type Mintage struct {
	Name        string             `json:"name"`
	Id          *types.TokenTypeId `json:"id"`
	Symbol      string             `json:"symbol"`
	Owner       *types.Address     `json:"owner"`
	Decimals    int                `json:"decimals"`
	TotalSupply *string            `json:"totalSupply"`
}

func rawMintageToRpc(l *ledger.Mintage) *Mintage {
	if l == nil {
		return nil
	}
	return &Mintage{
		Name:        l.Name,
		Id:          l.Id,
		Symbol:      l.Symbol,
		Owner:       l.Owner,
		Decimals:    l.Decimals,
		TotalSupply: bigIntToString(l.TotalSupply),
	}
}
