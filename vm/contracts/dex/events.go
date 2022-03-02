package dex

import (
	"fmt"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	dexproto "github.com/vitelabs/go-vite/v2/vm/contracts/dex/proto"
)

const newOrderEventName = "newOrderEvent"
const orderUpdateEventName = "orderUpdateEvent"
const txEventName = "txEvent"
const tokenEventName = "tokenEvent"
const marketEventName = "marketEvent"
const periodJobWithBizEventName = "periodWithBizEvent"
const feeDividendForVxHolderEventName = "feeDividendForVxHolderEvent"
const operatorFeeDividendEventName = "brokerFeeDividendEvent"
const minedVxForTradeFeeEventName = "minedVxForTradeFeeEvent"
const minedVxForInviteeFeeEventName = "minedVxForInviteeFeeEvent"
const minedVxForStakingEventName = "minedVxForPledgeEvent"
const minedVxForOperationEventName = "minedVxForOperation"
const inviteRelationEventName = "inviteRelationEvent"
const settleMakerMinedVxEventName = "settleMakerMinedVxEvent"
const grantMarketToAgentEventName = "grantMarketToAgentEvent"
const revokeMarketFromAgentEventName = "revokeMarketFromAgentEvent"
const burnViteEventName = "burnViteEvent"
const transferAssetEventName = "transferAssetEvent"
const errEventName = "errEvent"

type DexEvent interface {
	GetTopicId() types.Hash
	toDataBytes() []byte
	FromBytes([]byte) error
}

type NewOrderEvent struct {
	dexproto.NewOrderInfo
}

type OrderUpdateEvent struct {
	dexproto.OrderUpdateInfo
}

type TransactionEvent struct {
	dexproto.Transaction
}

type TokenEvent struct {
	dexproto.TokenInfo
}

type MarketEvent struct {
	dexproto.MarketInfo
}

type PeriodJobWithBizEvent struct {
	dexproto.PeriodJobForBiz
}

type FeeDividendEvent struct {
	dexproto.FeeDividendForVxHolder
}

type OperatorFeeDividendEvent struct {
	dexproto.OperatorFeeDividend
}

type MinedVxForTradeFeeEvent struct {
	dexproto.MinedVxForFee
}

type MinedVxForInviteeFeeEvent struct {
	dexproto.MinedVxForFee
}

type MinedVxForStakingEvent struct {
	dexproto.MinedVxForStaking
}

type MinedVxForOperationEvent struct {
	dexproto.MinedVxForOperation
}

type InviteRelationEvent struct {
	dexproto.InviteRelation
}

type SettleMakerMinedVxEvent struct {
	dexproto.SettleMakerMinedVx
}

type GrantMarketToAgentEvent struct {
	dexproto.MarketAgentRelation
}

type RevokeMarketFromAgentEvent struct {
	dexproto.MarketAgentRelation
}

type BurnViteEvent struct {
	dexproto.BurnVite
}

type TransferAssetEvent struct {
	dexproto.TransferAsset
}

type ErrEvent struct {
	error
}

func (od *NewOrderEvent) GetTopicId() types.Hash {
	return fromNameToHash(newOrderEventName)
}

func (od *NewOrderEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.NewOrderInfo)
	return data
}

func (od *NewOrderEvent) FromBytes(data []byte) (err error) {
	protoEvent := dexproto.NewOrderInfo{}
	if err = proto.Unmarshal(data, &protoEvent); err == nil {
		od.NewOrderInfo = protoEvent
	}
	return
}

func (od *OrderUpdateEvent) GetTopicId() types.Hash {
	return fromNameToHash(orderUpdateEventName)
}

func (od *OrderUpdateEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.OrderUpdateInfo)
	return data
}

func (od *OrderUpdateEvent) FromBytes(data []byte) (err error) {
	protoEvent := dexproto.OrderUpdateInfo{}
	if err = proto.Unmarshal(data, &protoEvent); err == nil {
		od.OrderUpdateInfo = protoEvent
	}
	return
}

func (tx *TransactionEvent) GetTopicId() types.Hash {
	return fromNameToHash(txEventName)
}

func (tx *TransactionEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&tx.Transaction)
	return data
}

func (tx *TransactionEvent) FromBytes(data []byte) (err error) {
	protoEvent := dexproto.Transaction{}
	if err = proto.Unmarshal(data, &protoEvent); err == nil {
		tx.Transaction = protoEvent
	}
	return
}

func (te *TokenEvent) GetTopicId() types.Hash {
	return fromNameToHash(tokenEventName)
}

func (te *TokenEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&te.TokenInfo)
	return data
}

func (te *TokenEvent) FromBytes(data []byte) (err error) {
	protoEvent := dexproto.TokenInfo{}
	if err = proto.Unmarshal(data, &protoEvent); err == nil {
		te.TokenInfo = protoEvent
	}
	return
}

func (me *MarketEvent) GetTopicId() types.Hash {
	return fromNameToHash(marketEventName)
}

func (me *MarketEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&me.MarketInfo)
	return data
}

func (me *MarketEvent) FromBytes(data []byte) (err error) {
	protoEvent := dexproto.MarketInfo{}
	if err = proto.Unmarshal(data, &protoEvent); err == nil {
		me.MarketInfo = protoEvent
	}
	return
}

func (pb *PeriodJobWithBizEvent) GetTopicId() types.Hash {
	return fromNameToHash(periodJobWithBizEventName)
}

func (pb *PeriodJobWithBizEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&pb.PeriodJobForBiz)
	return data
}

func (pb *PeriodJobWithBizEvent) FromBytes(data []byte) (err error) {
	event := dexproto.PeriodJobForBiz{}
	if err = proto.Unmarshal(data, &event); err == nil {
		pb.PeriodJobForBiz = event
	}
	return
}

func (fde *FeeDividendEvent) GetTopicId() types.Hash {
	return fromNameToHash(feeDividendForVxHolderEventName)
}

func (fde *FeeDividendEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&fde.FeeDividendForVxHolder)
	return data
}

func (fde *FeeDividendEvent) FromBytes(data []byte) (err error) {
	event := dexproto.FeeDividendForVxHolder{}
	if err = proto.Unmarshal(data, &event); err == nil {
		fde.FeeDividendForVxHolder = event
	}
	return
}

func (bfd *OperatorFeeDividendEvent) GetTopicId() types.Hash {
	return fromNameToHash(operatorFeeDividendEventName)
}

func (bfd *OperatorFeeDividendEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&bfd.OperatorFeeDividend)
	return data
}

func (bfd *OperatorFeeDividendEvent) FromBytes(data []byte) (err error) {
	event := dexproto.OperatorFeeDividend{}
	if err = proto.Unmarshal(data, &event); err == nil {
		bfd.OperatorFeeDividend = event
	}
	return
}

func (mtf *MinedVxForTradeFeeEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForTradeFeeEventName)
}

func (mtf *MinedVxForTradeFeeEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mtf.MinedVxForFee)
	return data
}

func (mtf *MinedVxForTradeFeeEvent) FromBytes(data []byte) (err error) {
	event := dexproto.MinedVxForFee{}
	if err = proto.Unmarshal(data, &event); err == nil {
		mtf.MinedVxForFee = event
	}
	return
}

func (mif *MinedVxForInviteeFeeEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForInviteeFeeEventName)
}

func (mif *MinedVxForInviteeFeeEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mif.MinedVxForFee)
	return data
}

func (mif *MinedVxForInviteeFeeEvent) FromBytes(data []byte) (err error) {
	event := dexproto.MinedVxForFee{}
	if err = proto.Unmarshal(data, &event); err == nil {
		mif.MinedVxForFee = event
	}
	return
}

func (mp *MinedVxForStakingEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForStakingEventName)
}

func (mp *MinedVxForStakingEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mp.MinedVxForStaking)
	return data
}

func (mp *MinedVxForStakingEvent) FromBytes(data []byte) (err error) {
	event := dexproto.MinedVxForStaking{}
	if err = proto.Unmarshal(data, &event); err == nil {
		mp.MinedVxForStaking = event
	}
	return
}

func (mo *MinedVxForOperationEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForOperationEventName)
}

func (mo *MinedVxForOperationEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mo.MinedVxForOperation)
	return data
}

func (mo *MinedVxForOperationEvent) FromBytes(data []byte) (err error) {
	event := dexproto.MinedVxForOperation{}
	if err = proto.Unmarshal(data, &event); err == nil {
		mo.MinedVxForOperation = event
	}
	return
}

func (ir *InviteRelationEvent) GetTopicId() types.Hash {
	return fromNameToHash(inviteRelationEventName)
}

func (ir *InviteRelationEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&ir.InviteRelation)
	return data
}

func (ir *InviteRelationEvent) FromBytes(data []byte) (err error) {
	event := dexproto.InviteRelation{}
	if err = proto.Unmarshal(data, &event); err == nil {
		ir.InviteRelation = event
	}
	return
}

func (smmv *SettleMakerMinedVxEvent) GetTopicId() types.Hash {
	return fromNameToHash(settleMakerMinedVxEventName)
}

func (smmv *SettleMakerMinedVxEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&smmv.SettleMakerMinedVx)
	return data
}

func (smmv *SettleMakerMinedVxEvent) FromBytes(data []byte) (err error) {
	event := dexproto.SettleMakerMinedVx{}
	if err = proto.Unmarshal(data, &event); err == nil {
		smmv.SettleMakerMinedVx = event
	}
	return
}

func (gmta *GrantMarketToAgentEvent) GetTopicId() types.Hash {
	return fromNameToHash(grantMarketToAgentEventName)
}

func (gmta *GrantMarketToAgentEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&gmta.MarketAgentRelation)
	return data
}

func (gmta *GrantMarketToAgentEvent) FromBytes(data []byte) (err error) {
	event := dexproto.MarketAgentRelation{}
	if err = proto.Unmarshal(data, &event); err == nil {
		gmta.MarketAgentRelation = event
	}
	return
}

func (rmfa *RevokeMarketFromAgentEvent) GetTopicId() types.Hash {
	return fromNameToHash(revokeMarketFromAgentEventName)
}

func (rmfa *RevokeMarketFromAgentEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&rmfa.MarketAgentRelation)
	return data
}

func (rmfa *RevokeMarketFromAgentEvent) FromBytes(data []byte) (err error) {
	event := dexproto.MarketAgentRelation{}
	if err = proto.Unmarshal(data, &event); err == nil {
		rmfa.MarketAgentRelation = event
	}
	return
}

func (bv *BurnViteEvent) GetTopicId() types.Hash {
	return fromNameToHash(burnViteEventName)
}

func (bv *BurnViteEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&bv.BurnVite)
	return data
}

func (bv *BurnViteEvent) FromBytes(data []byte) (err error) {
	event := dexproto.BurnVite{}
	if err = proto.Unmarshal(data, &event); err == nil {
		bv.BurnVite = event
	}
	return
}

func (tfa *TransferAssetEvent) GetTopicId() types.Hash {
	return fromNameToHash(transferAssetEventName)
}

func (tfa *TransferAssetEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&tfa.TransferAsset)
	return data
}

func (tfa *TransferAssetEvent) FromBytes(data []byte) (err error) {
	protoEvent := dexproto.TransferAsset{}
	if err = proto.Unmarshal(data, &protoEvent); err == nil {
		tfa.TransferAsset = protoEvent
	}
	return
}

func (err *ErrEvent) GetTopicId() types.Hash {
	return fromNameToHash(errEventName)
}

func (err *ErrEvent) toDataBytes() []byte {
	return []byte(err.Error())
}

func (err *ErrEvent) FromBytes(data []byte) (errRes error) {
	err.error = fmt.Errorf(string(data))
	return nil
}

func fromNameToHash(name string) types.Hash {
	hs := types.Hash{}
	hs.SetBytes(common.RightPadBytes([]byte(name), types.HashSize))
	return hs
}
