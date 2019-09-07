/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package net

import (
	"errors"
	"io"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

const (
	CodeDisconnect  Code = 1
	CodeHandshake   Code = 2
	CodeControlFlow Code = 3
	CodeHeartBeat   Code = 4

	CodeGetHashList       Code = 25
	CodeHashList          Code = 26
	CodeGetSnapshotBlocks Code = 27
	CodeSnapshotBlocks    Code = 28
	CodeGetAccountBlocks  Code = 29
	CodeAccountBlocks     Code = 30
	CodeNewSnapshotBlock  Code = 31
	CodeNewAccountBlock   Code = 32

	CodeSyncHandshake   Code = 60
	CodeSyncHandshakeOK Code = 61
	CodeSyncRequest     Code = 62
	CodeSyncReady       Code = 63

	CodeException Code = 127
	CodeTrace     Code = 128
)

const version = iota

type Code = byte
type MsgId = uint32

type Msg struct {
	Code       Code
	Id         uint32
	Payload    []byte
	ReceivedAt int64
	Sender     *Peer
}

// Recycle will put Msg.Payload back to pool
func (m Msg) Recycle() {
}

type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	WriteMsg(Msg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type MsgWriteCloser interface {
	MsgWriter
	io.Closer
}

type Serializable interface {
	Serialize() ([]byte, error)
}

func Disconnect(c MsgWriteCloser, err error) (e2 error) {
	var msg = Msg{
		Code: CodeDisconnect,
	}

	if err != nil {
		if pe, ok := err.(PeerError); ok {
			msg.Payload, _ = pe.Serialize()
		}
	} else {
		msg.Payload, _ = PeerQuitting.Serialize()
	}

	e2 = c.WriteMsg(msg)

	_ = c.Close()
	return nil
}

var errDeserialize = errors.New("deserialize error")

// @section GetSnapshotBlocks

type GetSnapshotBlocks struct {
	From    ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetSnapshotBlocks) String() string {
	var from string
	if b.From.Hash == types.ZERO_HASH {
		from = strconv.FormatUint(b.From.Height, 10)
	} else {
		from = b.From.Hash.String()
	}

	return "GetSnapshotBlocks<" + from + "/" + strconv.FormatUint(b.Count, 10) + "/" + strconv.FormatBool(b.Forward) + ">"
}

func (b *GetSnapshotBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.GetSnapshotBlocks)
	pb.From = &vitepb.HashHeight{
		Hash:   b.From.Hash[:],
		Height: b.From.Height,
	}
	pb.Count = b.Count
	pb.Forward = b.Forward

	return proto.Marshal(pb)
}

func (b *GetSnapshotBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.GetSnapshotBlocks)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	if pb.From == nil {
		return errDeserialize
	}

	b.From = ledger.HashHeight{
		Height: pb.From.Height,
	}
	copy(b.From.Hash[:], pb.From.Hash)

	b.Count = pb.Count
	b.Forward = pb.Forward

	return nil
}

// SnapshotBlocks is batch of snapshot blocks
type SnapshotBlocks struct {
	Blocks []*ledger.SnapshotBlock
}

func (b *SnapshotBlocks) String() string {
	return "SnapshotBlocks<" + strconv.Itoa(len(b.Blocks)) + ">"
}

func (b *SnapshotBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.SnapshotBlocks)

	pb.Blocks = make([]*vitepb.SnapshotBlock, len(b.Blocks))

	for i, block := range b.Blocks {
		pb.Blocks[i] = block.Proto()
	}

	return proto.Marshal(pb)
}

func (b *SnapshotBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.SnapshotBlocks)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	b.Blocks = make([]*ledger.SnapshotBlock, len(pb.Blocks))
	var j int
	for _, bp := range pb.Blocks {
		if bp == nil {
			return errDeserialize
		}
		block := new(ledger.SnapshotBlock)
		err = block.DeProto(bp)
		if err != nil {
			return err
		}
		b.Blocks[j] = block
		j++
	}

	return nil
}

// @section GetAccountBlocks

type GetAccountBlocks struct {
	Address types.Address
	From    ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetAccountBlocks) String() string {
	var from string
	if b.From.Hash == types.ZERO_HASH {
		from = strconv.FormatUint(b.From.Height, 10)
	} else {
		from = b.From.Hash.String()
	}

	return "GetAccountBlocks<" + from + "/" + strconv.FormatUint(b.Count, 10) + "/" + strconv.FormatBool(b.Forward) + ">"
}

func (b *GetAccountBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.GetAccountBlocks)
	pb.Address = b.Address[:]
	pb.From = &vitepb.HashHeight{
		Hash:   b.From.Hash[:],
		Height: b.From.Height,
	}
	pb.Count = b.Count
	pb.Forward = b.Forward

	return proto.Marshal(pb)
}

func (b *GetAccountBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.GetAccountBlocks)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	if pb.From == nil {
		return errDeserialize
	}

	b.From = ledger.HashHeight{
		Height: pb.From.Height,
	}
	copy(b.From.Hash[:], pb.From.Hash)

	b.Count = pb.Count
	b.Forward = pb.Forward
	copy(b.Address[:], pb.Address)

	return nil
}

// AccountBlocks is batch of account blocks
type AccountBlocks struct {
	Blocks []*ledger.AccountBlock
	TTL    int32
}

func (a *AccountBlocks) String() string {
	return "AccountBlocks<" + strconv.Itoa(len(a.Blocks)) + ">"
}

func (a *AccountBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.AccountBlocks)

	pb.Blocks = make([]*vitepb.AccountBlock, len(a.Blocks))

	for i, block := range a.Blocks {
		pb.Blocks[i] = block.Proto()
	}

	return proto.Marshal(pb)
}

func (a *AccountBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.AccountBlocks)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	a.Blocks = make([]*ledger.AccountBlock, len(pb.Blocks))
	var j int
	for _, bp := range pb.Blocks {
		if bp == nil {
			return errDeserialize
		}

		block := new(ledger.AccountBlock)
		err = block.DeProto(bp)
		if err != nil {
			return err
		}
		a.Blocks[j] = block
		j++
	}

	return nil
}

// NewSnapshotBlock is use to propagate block, stop propagate when TTL is decrease to zero
type NewSnapshotBlock struct {
	Block *ledger.SnapshotBlock
	TTL   int32
}

func (b *NewSnapshotBlock) Serialize() ([]byte, error) {
	pb := new(vitepb.NewSnapshotBlock)

	pb.Block = b.Block.Proto()

	pb.TTL = b.TTL

	return proto.Marshal(pb)
}

func (b *NewSnapshotBlock) Deserialize(buf []byte) error {
	pb := new(vitepb.NewSnapshotBlock)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	if pb.Block == nil {
		return errDeserialize
	}

	b.Block = new(ledger.SnapshotBlock)
	err = b.Block.DeProto(pb.Block)
	if err != nil {
		return err
	}

	b.TTL = pb.TTL

	return nil
}

// NewAccountBlock is use to propagate block, stop propagate when TTL is decrease to zero
type NewAccountBlock struct {
	Block *ledger.AccountBlock
	TTL   int32
}

func (b *NewAccountBlock) Serialize() ([]byte, error) {
	pb := new(vitepb.NewAccountBlock)

	pb.Block = b.Block.Proto()

	pb.TTL = b.TTL

	return proto.Marshal(pb)
}

func (b *NewAccountBlock) Deserialize(buf []byte) error {
	pb := new(vitepb.NewAccountBlock)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	if pb.Block == nil {
		return errDeserialize
	}

	b.Block = new(ledger.AccountBlock)
	err = b.Block.DeProto(pb.Block)
	if err != nil {
		return err
	}

	b.TTL = pb.TTL

	return nil
}

var errMissingPoints = errors.New("missing from points")
var errNilPoint = errors.New("nil HashHeightPoint")

func pointsToPb(points []*HashHeightPoint) []*vitepb.HashHeightPoint {
	pb := make([]*vitepb.HashHeightPoint, len(points))
	for i, h := range points {
		pb[i] = h.Proto()
	}

	return pb
}

func pbToPoints(pb []*vitepb.HashHeightPoint) (points []*HashHeightPoint, err error) {
	points = make([]*HashHeightPoint, len(pb))

	var j int
	for _, h := range pb {
		if h == nil {
			return nil, errNilPoint
		}

		hh := new(HashHeightPoint)
		err = hh.DeProto(h)
		if err != nil {
			return
		}
		points[j] = hh
		j++
	}

	return
}

func hashHeightToPbs(points []*ledger.HashHeight) []*vitepb.HashHeight {
	pb := make([]*vitepb.HashHeight, len(points))
	for i, h := range points {
		pb[i] = h.Proto()
	}

	return pb
}

func pbsToHashHeight(pb []*vitepb.HashHeight) (points []*ledger.HashHeight, err error) {
	points = make([]*ledger.HashHeight, len(pb))

	var j int
	for _, h := range pb {
		if h == nil {
			return nil, errNilPoint
		}

		hh := new(ledger.HashHeight)
		err = hh.DeProto(h)
		if err != nil {
			return
		}
		points[j] = hh
		j++
	}

	return
}

type HashHeightPoint struct {
	ledger.HashHeight
	Size uint64
}

func (p *HashHeightPoint) Proto() *vitepb.HashHeightPoint {
	return &vitepb.HashHeightPoint{
		Point: &vitepb.HashHeight{
			Hash:   p.Hash.Bytes(),
			Height: p.Height,
		},
		Size: p.Size,
	}
}

func (p *HashHeightPoint) DeProto(pb *vitepb.HashHeightPoint) (err error) {
	if pb == nil || pb.Point == nil {
		return errNilPoint
	}

	p.Hash, err = types.BytesToHash(pb.Point.Hash)
	if err != nil {
		return
	}
	p.Height = pb.Point.Height
	p.Size = pb.Size

	return
}

type HashHeightPointList struct {
	Points []*HashHeightPoint // from low to high
}

func (c *HashHeightPointList) Serialize() ([]byte, error) {
	pb := &vitepb.HashHeightList{
		Points: pointsToPb(c.Points),
	}

	return proto.Marshal(pb)
}

func (c *HashHeightPointList) Deserialize(data []byte) (err error) {
	pb := &vitepb.HashHeightList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	c.Points, err = pbToPoints(pb.Points)

	return
}

type GetHashHeightList struct {
	From []*ledger.HashHeight // from high to low
	Step uint64
	To   uint64
}

func (c *GetHashHeightList) Serialize() ([]byte, error) {
	pb := &vitepb.GetHashHeightList{
		From: hashHeightToPbs(c.From),
		Step: c.Step,
		To:   c.To,
	}

	return proto.Marshal(pb)
}

func (c *GetHashHeightList) Deserialize(data []byte) (err error) {
	pb := &vitepb.GetHashHeightList{}
	err = proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	if len(pb.From) == 0 {
		return errMissingPoints
	}

	c.From, err = pbsToHashHeight(pb.From)
	c.Step = pb.Step
	c.To = pb.To

	return
}
