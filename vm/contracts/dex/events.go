package dex

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
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
const errEventName = "errEvent"

type DexEvent interface {
	GetTopicId() types.Hash
	toDataBytes() []byte
	FromBytes([]byte) interface{}
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

type ErrEvent struct {
	error
}

func (od NewOrderEvent) GetTopicId() types.Hash {
	return fromNameToHash(newOrderEventName)
}

func (od NewOrderEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.NewOrderInfo)
	return data
}

func (od NewOrderEvent) FromBytes(data []byte) interface{} {
	event := NewOrderEvent{}
	if err := proto.Unmarshal(data, &event.NewOrderInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (od OrderUpdateEvent) GetTopicId() types.Hash {
	return fromNameToHash(orderUpdateEventName)
}

func (od OrderUpdateEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&od.OrderUpdateInfo)
	return data
}

func (od OrderUpdateEvent) FromBytes(data []byte) interface{} {
	event := OrderUpdateEvent{}
	if err := proto.Unmarshal(data, &event.OrderUpdateInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (tx TransactionEvent) GetTopicId() types.Hash {
	return fromNameToHash(txEventName)
}

func (tx TransactionEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&tx.Transaction)
	return data
}

func (tx TransactionEvent) FromBytes(data []byte) interface{} {
	event := TransactionEvent{}
	if err := proto.Unmarshal(data, &event.Transaction); err != nil {
		return nil
	} else {
		return event
	}
}

func (te TokenEvent) GetTopicId() types.Hash {
	return fromNameToHash(tokenEventName)
}

func (te TokenEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&te.TokenInfo)
	return data
}

func (te TokenEvent) FromBytes(data []byte) interface{} {
	event := TokenEvent{}
	if err := proto.Unmarshal(data, &event.TokenInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (me MarketEvent) GetTopicId() types.Hash {
	return fromNameToHash(marketEventName)
}

func (me MarketEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&me.MarketInfo)
	return data
}

func (me MarketEvent) FromBytes(data []byte) interface{} {
	event := MarketEvent{}
	if err := proto.Unmarshal(data, &event.MarketInfo); err != nil {
		return nil
	} else {
		return event
	}
}

func (pb PeriodJobWithBizEvent) GetTopicId() types.Hash {
	return fromNameToHash(periodJobWithBizEventName)
}

func (pb PeriodJobWithBizEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&pb.PeriodJobForBiz)
	return data
}

func (pb PeriodJobWithBizEvent) FromBytes(data []byte) interface{} {
	event := PeriodJobWithBizEvent{}
	if err := proto.Unmarshal(data, &event.PeriodJobForBiz); err != nil {
		return nil
	} else {
		return event
	}
}

func (fde FeeDividendEvent) GetTopicId() types.Hash {
	return fromNameToHash(feeDividendForVxHolderEventName)
}

func (fde FeeDividendEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&fde.FeeDividendForVxHolder)
	return data
}

func (fde FeeDividendEvent) FromBytes(data []byte) interface{} {
	event := FeeDividendEvent{}
	if err := proto.Unmarshal(data, &event.FeeDividendForVxHolder); err != nil {
		return nil
	} else {
		return event
	}
}

func (bfd OperatorFeeDividendEvent) GetTopicId() types.Hash {
	return fromNameToHash(operatorFeeDividendEventName)
}

func (bfd OperatorFeeDividendEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&bfd.OperatorFeeDividend)
	return data
}

func (bfd OperatorFeeDividendEvent) FromBytes(data []byte) interface{} {
	event := OperatorFeeDividendEvent{}
	if err := proto.Unmarshal(data, &event.OperatorFeeDividend); err != nil {
		return nil
	} else {
		return event
	}
}

func (mtf MinedVxForTradeFeeEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForTradeFeeEventName)
}

func (mtf MinedVxForTradeFeeEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mtf.MinedVxForFee)
	return data
}

func (mtf MinedVxForTradeFeeEvent) FromBytes(data []byte) interface{} {
	event := MinedVxForTradeFeeEvent{}
	if err := proto.Unmarshal(data, &event.MinedVxForFee); err != nil {
		return nil
	} else {
		return event
	}
}

func (mif MinedVxForInviteeFeeEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForInviteeFeeEventName)
}

func (mif MinedVxForInviteeFeeEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mif.MinedVxForFee)
	return data
}

func (mif MinedVxForInviteeFeeEvent) FromBytes(data []byte) interface{} {
	event := MinedVxForInviteeFeeEvent{}
	if err := proto.Unmarshal(data, &event.MinedVxForFee); err != nil {
		return nil
	} else {
		return event
	}
}

func (mp MinedVxForStakingEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForStakingEventName)
}

func (mp MinedVxForStakingEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mp.MinedVxForStaking)
	return data
}

func (mp MinedVxForStakingEvent) FromBytes(data []byte) interface{} {
	event := MinedVxForStakingEvent{}
	if err := proto.Unmarshal(data, &event.MinedVxForStaking); err != nil {
		return nil
	} else {
		return event
	}
}

func (mo MinedVxForOperationEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForOperationEventName)
}

func (mo MinedVxForOperationEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mo.MinedVxForOperation)
	return data
}

func (mo MinedVxForOperationEvent) FromBytes(data []byte) interface{} {
	event := MinedVxForOperationEvent{}
	if err := proto.Unmarshal(data, &event.MinedVxForOperation); err != nil {
		return nil
	} else {
		return event
	}
}

func (ir InviteRelationEvent) GetTopicId() types.Hash {
	return fromNameToHash(inviteRelationEventName)
}

func (ir InviteRelationEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&ir.InviteRelation)
	return data
}

func (ir InviteRelationEvent) FromBytes(data []byte) interface{} {
	event := InviteRelationEvent{}
	if err := proto.Unmarshal(data, &event.InviteRelation); err != nil {
		return nil
	} else {
		return event
	}
}

func (smmv SettleMakerMinedVxEvent) GetTopicId() types.Hash {
	return fromNameToHash(settleMakerMinedVxEventName)
}

func (smmv SettleMakerMinedVxEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&smmv.SettleMakerMinedVx)
	return data
}

func (smmv SettleMakerMinedVxEvent) FromBytes(data []byte) interface{} {
	event := SettleMakerMinedVxEvent{}
	if err := proto.Unmarshal(data, &event.SettleMakerMinedVx); err != nil {
		return nil
	} else {
		return event
	}
}

func (gmta GrantMarketToAgentEvent) GetTopicId() types.Hash {
	return fromNameToHash(grantMarketToAgentEventName)
}

func (gmta GrantMarketToAgentEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&gmta.MarketAgentRelation)
	return data
}

func (gmta GrantMarketToAgentEvent) FromBytes(data []byte) interface{} {
	event := GrantMarketToAgentEvent{}
	if err := proto.Unmarshal(data, &event.MarketAgentRelation); err != nil {
		return nil
	} else {
		return event
	}
}

func (rmfa RevokeMarketFromAgentEvent) GetTopicId() types.Hash {
	return fromNameToHash(revokeMarketFromAgentEventName)
}

func (rmfa RevokeMarketFromAgentEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&rmfa.MarketAgentRelation)
	return data
}

func (rmfa RevokeMarketFromAgentEvent) FromBytes(data []byte) interface{} {
	event := RevokeMarketFromAgentEvent{}
	if err := proto.Unmarshal(data, &event.MarketAgentRelation); err != nil {
		return nil
	} else {
		return event
	}
}

func (bv BurnViteEvent) GetTopicId() types.Hash {
	return fromNameToHash(burnViteEventName)
}

func (bv BurnViteEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&bv.BurnVite)
	return data
}

func (bv BurnViteEvent) FromBytes(data []byte) interface{} {
	event := BurnViteEvent{}
	if err := proto.Unmarshal(data, &event.BurnVite); err != nil {
		return nil
	} else {
		return event
	}
}

func (err ErrEvent) GetTopicId() types.Hash {
	return fromNameToHash(errEventName)
}

func (err ErrEvent) toDataBytes() []byte {
	return []byte(err.Error())
}

func (err ErrEvent) FromBytes(data []byte) interface{} {
	return ErrEvent{fmt.Errorf(string(data))}
}

func fromNameToHash(name string) types.Hash {
	hs := types.Hash{}
	hs.SetBytes(common.RightPadBytes([]byte(name), types.HashSize))
	return hs
}
