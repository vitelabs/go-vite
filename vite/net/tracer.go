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
	"crypto/rand"
	"fmt"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

type tracer struct {
	id string

	peers broadcastPeerSet

	strategy forwardStrategy

	log log15.Logger

	filter blockFilter
}

func newTracer(id string, peers broadcastPeerSet, strategy forwardStrategy) *tracer {
	return &tracer{
		id:       id,
		peers:    peers,
		strategy: strategy,
		log:      netLog.New("module", "tracer"),
		filter:   newBlockFilter(filterCap),
	}
}

func (b *tracer) name() string {
	return "tracer"
}

func (b *tracer) codes() []code {
	return []code{codeTrace}
}

func (b *tracer) handle(msg p2p.Msg, sender Peer) (err error) {
	switch code(msg.Code) {
	case codeTrace:
		tm := &message.Tracer{}
		err = tm.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		b.log.Info(fmt.Sprintf("receive tracer %s from %s", tm.Hash, sender))

		sender.seeBlock(tm.Hash)

		// check if has exist or record, return true if has exist
		if exist := b.filter.lookAndRecord(tm.Hash[:]); exist {
			return nil
		}

		b.log.Info(fmt.Sprintf("record tracer %s from %s", tm.Hash, sender))

		if tm.TTL > 0 {
			tm.TTL--
			b.forward(tm, sender)
		}
	}

	return nil
}

func (b *tracer) Trace() {
	var hash types.Hash
	_, _ = rand.Read(hash[:])
	var msg = &message.Tracer{
		Hash: hash,
		Path: []string{
			b.id,
		},
		TTL: defaultBroadcastTTL,
	}

	var err error
	ps := b.peers.broadcastPeers()
	for _, p := range ps {
		if p.seeBlock(hash) {
			continue
		}

		err = p.send(codeTrace, 0, msg)
		if err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to broadcast trace %s to %s: %v", hash, p, err))
		} else {
			b.log.Info(fmt.Sprintf("broadcast trace %s to %s", hash, p))
		}
	}
}

func (b *tracer) forward(msg *message.Tracer, sender broadcastPeer) {
	msg.Path = append(msg.Path, b.id)

	pl := b.strategy.choosePeers(sender)
	for _, p := range pl {
		if p.seeBlock(msg.Hash) {
			continue
		}

		if err := p.send(codeTrace, 0, msg); err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to forward trace %s to %s: %v", msg.Hash, p, err))
		} else {
			b.log.Info(fmt.Sprintf("forward trace %s to %s", msg.Hash, p))
		}
	}
}
