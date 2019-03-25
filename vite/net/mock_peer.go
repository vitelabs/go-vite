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
 * You should have received chain copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package net

import (
	"fmt"
	net2 "net"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
)

type Handler func(msgId uint64, payload p2p.Serializable)
type MockPeer struct {
	Handlers map[ViteCmd]Handler
	addr     *net2.TCPAddr
	faddr    *net2.TCPAddr
}

func NewMockPeer() *MockPeer {
	mp := &MockPeer{
		Handlers: make(map[ViteCmd]Handler),
		addr:     &net2.TCPAddr{},
		faddr:    &net2.TCPAddr{},
	}
	mp.Handlers[FileListCode] = defFileListHandler

	return mp
}

func (mp *MockPeer) RemoteAddr() *net2.TCPAddr {
	return mp.addr
}

func (mp *MockPeer) FileAddress() *net2.TCPAddr {
	return mp.faddr
}

func (mp *MockPeer) SetHead(head types.Hash, height uint64) {
	panic("implement me")
}

func (mp *MockPeer) SeeBlock(hash types.Hash) {
	panic("implement me")
}

func (mp *MockPeer) SendSnapshotBlocks(bs []*ledger.SnapshotBlock, msgId uint64) (err error) {
	panic("implement me")
}

func (mp *MockPeer) SendAccountBlocks(bs []*ledger.AccountBlock, msgId uint64) (err error) {
	panic("implement me")
}

func (mp *MockPeer) SendNewSnapshotBlock(b *ledger.SnapshotBlock) (err error) {
	panic("implement me")
}

func (mp *MockPeer) SendNewAccountBlock(b *ledger.AccountBlock) (err error) {
	panic("implement me")
}

func (mp *MockPeer) Send(code ViteCmd, msgId uint64, payload p2p.Serializable) (err error) {
	return
}

func (mp *MockPeer) SendMsg(msg *p2p.Msg) (err error) {
	panic("implement me")
}

func (mp *MockPeer) Report(err error) {
	panic("implement me")
}

func (mp *MockPeer) ID() string {
	panic("implement me")
}

func (mp *MockPeer) Height() uint64 {
	panic("implement me")
}

func (mp *MockPeer) Head() types.Hash {
	panic("implement me")
}

func (mp *MockPeer) Disconnect(reason p2p.DiscReason) {
	panic("implement me")
}

func defFileListHandler(msgId uint64, payload p2p.Serializable) {
	fileList := payload.(*message.FileList)
	fmt.Printf("msgId: %d\n", msgId)
	for _, f := range fileList.Files {
		fmt.Println(f)
	}

	fmt.Println(fileList.Chunks)
}
