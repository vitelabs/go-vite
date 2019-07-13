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
const periodWithBizEventName = "periodWithBizEvent"
const feeDividendForVxHolderEventName = "feeDividendForVxHolderEvent"
const brokerFeeDividendEventName = "brokerFeeDividendEvent"
const minedVxForTradeFeeEventName = "minedVxForTradeFeeEvent"
const minedVxForInviteeFeeEventName = "minedVxForInviteeFeeEvent"
const minedVxForPledgeEventName = "minedVxForPledgeEvent"
const minedVxForOperationEventName = "minedVxForOperation"
const inviteRelationEventName = "inviteRelationEvent"
const settleMakerMinedVxEventName = "settleMakerMinedVxEvent"
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

type PeriodWithBizEvent struct {
	dexproto.PeriodWithBiz
}

type FeeDividendEvent struct {
	dexproto.FeeDividendForVxHolder
}

type BrokerFeeDividendEvent struct {
	dexproto.BrokerFeeDividend
}

type MinedVxForTradeFeeEvent struct {
	dexproto.MinedVxForFee
}

type MinedVxForInviteeFeeEvent struct {
	dexproto.MinedVxForFee
}

type MinedVxForPledgeEvent struct {
	dexproto.MinedVxForPledge
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

func (pb PeriodWithBizEvent) GetTopicId() types.Hash {
	return fromNameToHash(periodWithBizEventName)
}

func (pb PeriodWithBizEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&pb.PeriodWithBiz)
	return data
}

func (pb PeriodWithBizEvent) FromBytes(data []byte) interface{} {
	event := PeriodWithBizEvent{}
	if err := proto.Unmarshal(data, &event.PeriodWithBiz); err != nil {
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

func (bfd BrokerFeeDividendEvent) GetTopicId() types.Hash {
	return fromNameToHash(brokerFeeDividendEventName)
}

func (bfd BrokerFeeDividendEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&bfd.BrokerFeeDividend)
	return data
}

func (bfd BrokerFeeDividendEvent) FromBytes(data []byte) interface{} {
	event := BrokerFeeDividendEvent{}
	if err := proto.Unmarshal(data, &event.BrokerFeeDividend); err != nil {
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

func (mp MinedVxForPledgeEvent) GetTopicId() types.Hash {
	return fromNameToHash(minedVxForPledgeEventName)
}

func (mp MinedVxForPledgeEvent) toDataBytes() []byte {
	data, _ := proto.Marshal(&mp.MinedVxForPledge)
	return data
}

func (mp MinedVxForPledgeEvent) FromBytes(data []byte) interface{} {
	event := MinedVxForPledgeEvent{}
	if err := proto.Unmarshal(data, &event.MinedVxForPledge); err != nil {
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
