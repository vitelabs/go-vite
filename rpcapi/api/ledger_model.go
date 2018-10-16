package api

import (
	"github.com/vitelabs/go-vite/chain/sender"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts"
	"math/big"
	"strconv"
	"time"
)

type AccountBlock struct {
	*ledger.AccountBlock

	FromAddress types.Address `json:"fromAddress"`

	Height string `json:"height"`
	Quota  *string `json:"quota"`

	Amount *string `json:"amount"`
	Fee    *string `json:"fee"`

	Timestamp int64 `json:"timestamp"`

	ConfirmedTimes *string        `json:"confirmedTimes"`
	TokenInfo      *RpcTokenInfo `json:"tokenInfo"`
}

func (ab *AccountBlock) LedgerAccountBlock() (*ledger.AccountBlock, error) {
	lAb := ab.AccountBlock

	var err error
	lAb.Height, err = strconv.ParseUint(ab.Height, 10, 8)
	if err != nil {
		return nil, err
	}
	if ab.Quota != nil {
		lAb.Quota, err = strconv.ParseUint(*ab.Quota, 10, 8)
		if err != nil {
			return nil, err
		}
	}


	lAb.Amount = big.NewInt(0)
	if ab.Amount != nil {
		lAb.Amount.SetString(*ab.Amount, 10)
	}

	lAb.Fee = big.NewInt(0)
	if ab.Fee != nil {
		lAb.Fee.SetString(*ab.Fee, 10)
	}

	t := time.Unix(ab.Timestamp, 0)
	lAb.Timestamp = &t

	return lAb, nil
}

func createAccountBlock(ledgerBlock *ledger.AccountBlock, token *contracts.TokenInfo, confirmedTimes uint64) *AccountBlock {
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

func RawTokenInfoToRpc(tinfo *contracts.TokenInfo, tti types.TokenTypeId) *RpcTokenInfo {
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
		BrokerList: producer.BrokerList(),
		Topic:      producer.Topic(),
		HasSend:    producer.HasSend(),
		Status:     status,
	}

	return producerInfo
}

