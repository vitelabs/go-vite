package api

import (
	"errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain/sender"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"strconv"
	"time"
)

type AccountBlock struct {
	*ledger.AccountBlock

	FromAddress types.Address `json:"fromAddress"`

	Height string  `json:"height"`
	Quota  *string `json:"quota"`

	Amount     *string `json:"amount"`
	Fee        *string `json:"fee"`
	Difficulty *string `json:"difficulty"`

	Timestamp int64 `json:"timestamp"`

	ConfirmedTimes *string       `json:"confirmedTimes"`
	TokenInfo      *RpcTokenInfo `json:"tokenInfo"`

	ReceiveBlockHeights []uint64 `json:receiveBlockHeights`
}

func (ab *AccountBlock) LedgerAccountBlock() (*ledger.AccountBlock, error) {
	lAb := ab.AccountBlock
	if lAb == nil {
		lAb = &ledger.AccountBlock{}
	}

	var err error
	lAb.Height, err = strconv.ParseUint(ab.Height, 10, 64)
	if err != nil {
		return nil, err
	}
	if ab.Quota != nil {
		lAb.Quota, err = strconv.ParseUint(*ab.Quota, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	lAb.Amount = big.NewInt(0)
	if ab.Amount != nil {
		if _, ok := lAb.Amount.SetString(*ab.Amount, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	lAb.Fee = big.NewInt(0)
	if ab.Fee != nil {
		if _, ok := lAb.Fee.SetString(*ab.Fee, 10); !ok {
			return nil, ErrStrToBigInt
		}
	}

	if ab.AccountBlock != nil && ab.AccountBlock.Nonce != nil {
		if ab.Difficulty == nil {
			return nil, errors.New("lack of difficulty field")
		} else {
			setString, ok := new(big.Int).SetString(*ab.Difficulty, 10)
			if !ok {
				return nil, ErrStrToBigInt
			}
			lAb.Difficulty = setString
		}
	}

	t := time.Unix(ab.Timestamp, 0)
	lAb.Timestamp = &t

	return lAb, nil
}

func createAccountBlock(ledgerBlock *ledger.AccountBlock, token *types.TokenInfo, confirmedTimes uint64) *AccountBlock {
	zero := "0"
	quota := strconv.FormatUint(ledgerBlock.Quota, 10)
	confirmedTimeStr := strconv.FormatUint(confirmedTimes, 10)
	ab := &AccountBlock{
		AccountBlock: ledgerBlock,

		Height: strconv.FormatUint(ledgerBlock.Height, 10),
		Quota:  &quota,

		Amount:         &zero,
		Fee:            &zero,
		TokenInfo:      RawTokenInfoToRpc(token, ledgerBlock.TokenId),
		ConfirmedTimes: &confirmedTimeStr,
	}

	if ledgerBlock.Difficulty != nil {
		difficulty := ledgerBlock.Difficulty.String()
		ab.Difficulty = &difficulty
	}

	if ledgerBlock.Timestamp != nil {
		ab.Timestamp = ledgerBlock.Timestamp.Unix()
	}

	if token != nil {
		ab.TokenInfo = RawTokenInfoToRpc(token, ledgerBlock.TokenId)
	}
	if ledgerBlock.Amount != nil {
		a := ledgerBlock.Amount.String()
		ab.Amount = &a
	}
	if ledgerBlock.Fee != nil {
		s := ledgerBlock.Fee.String()
		ab.Fee = &s
	}

	return ab
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
	TokenName      string            `json:"tokenName"`
	TokenSymbol    string            `json:"tokenSymbol"`
	TotalSupply    *string           `json:"totalSupply,omitempty"` // *big.Int
	Decimals       uint8             `json:"decimals"`
	Owner          types.Address     `json:"owner"`
	PledgeAmount   *string           `json:"pledgeAmount,omitempty"` // *big.Int
	WithdrawHeight string            `json:"withdrawHeight"`         // uint64
	TokenId        types.TokenTypeId `json:"tokenId"`
}

func RawTokenInfoToRpc(tinfo *types.TokenInfo, tti types.TokenTypeId) *RpcTokenInfo {
	var rt *RpcTokenInfo = nil
	if tinfo != nil {
		rt = &RpcTokenInfo{
			TokenName:      tinfo.TokenName,
			TokenSymbol:    tinfo.TokenSymbol,
			TotalSupply:    nil,
			Decimals:       tinfo.Decimals,
			Owner:          tinfo.Owner,
			PledgeAmount:   nil,
			WithdrawHeight: strconv.FormatUint(tinfo.WithdrawHeight, 10),
			TokenId:        tti,
		}
		if tinfo.TotalSupply != nil {
			s := tinfo.TotalSupply.String()
			rt.TotalSupply = &s
		}
		if tinfo.PledgeAmount != nil {
			s := tinfo.PledgeAmount.String()
			rt.PledgeAmount = &s
		}
	}
	return rt
}

type KafkaSendInfo struct {
	Producers    []*KafkaProducerInfo `json:"producers"`
	RunProducers []*KafkaProducerInfo `json:"runProducers"`
	TotalEvent   uint64               `json:"totalEvent"`
}

type KafkaProducerInfo struct {
	ProducerId uint8    `json:"producerId"`
	BrokerList []string `json:"brokerList"`
	Topic      string   `json:"topic"`
	HasSend    uint64   `json:"hasSend"`
	Status     string   `json:"status"`
}

func createKafkaProducerInfo(producer *sender.Producer) *KafkaProducerInfo {
	status := "unknown"
	switch producer.Status() {
	case sender.STOPPED:
		status = "stopped"
	case sender.RUNNING:
		status = "running"
	}

	producerInfo := &KafkaProducerInfo{
		ProducerId: producer.ProducerId(),
		BrokerList: producer.BrokerList(),
		Topic:      producer.Topic(),
		HasSend:    producer.HasSend(),
		Status:     status,
	}

	return producerInfo
}

func ledgerToRpcBlock(block *ledger.AccountBlock, chain chain.Chain) (*AccountBlock, error) {
	confirmTimes, err := chain.GetConfirmTimes(&block.Hash)

	if err != nil {
		return nil, err
	}

	var fromAddress, toAddress types.Address
	if block.IsReceiveBlock() {
		toAddress = block.AccountAddress
		sendBlock, err := chain.GetAccountBlockByHash(&block.FromBlockHash)
		if err != nil {
			return nil, err
		}

		if sendBlock != nil {
			fromAddress = sendBlock.AccountAddress
			block.TokenId = sendBlock.TokenId
			block.Amount = sendBlock.Amount
			block.Fee = sendBlock.Fee
		}
	} else {
		fromAddress = block.AccountAddress
		toAddress = block.ToAddress
	}

	token, _ := chain.GetTokenInfoById(&block.TokenId)
	rpcAccountBlock := createAccountBlock(block, token, confirmTimes)
	rpcAccountBlock.FromAddress = fromAddress
	rpcAccountBlock.ToAddress = toAddress

	if block.IsSendBlock() {
		if block.Meta == nil {
			var err error
			block.Meta, err = chain.ChainDb().Ac.GetBlockMeta(&block.Hash)
			if err != nil {
				return nil, err
			}
		}

		if block.Meta != nil {
			rpcAccountBlock.ReceiveBlockHeights = block.Meta.ReceiveBlockHeights
		}
	}
	return rpcAccountBlock, nil
}
