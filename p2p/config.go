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

package p2p

import (
	"github.com/vitelabs/go-vite/p2p/discovery"
)

const (
	DefaultNodeName        = "vite-node"
	DefaultMaxPeers        = 10
	DefaultMaxInboundRatio = 2
	DefaultOutboundPeers   = 5
	DefaultSuperiorPeers   = 30
	DefaultMinPeers        = DefaultOutboundPeers
	DefaultMaxPendingPeers = 10
)

// Config is the essential configuration to create a p2p server
type Config struct {
	*discovery.Config

	// discover means whether discover other nodes in the networks, default true
	discover bool

	// name is our node name, NO need to be unique in the whole network, just for readability, default is `vite-node`
	name string

	// maxPeers means each level can accept how many peers, default:
	// Inbound: 5
	// Outbound: 5
	// Superior: 30
	maxPeers map[Level]int

	// minPeers, server will keep finding nodes and try to connect until number of peers is larger than `MinPeers`,
	// default 5
	minPeers int

	// maxPendingPeers: how many inbound peers can be connect concurrently, more inbound connection will be blocked
	// this value is for defend DDOS attack, default 10
	maxPendingPeers int

	// staticNodes will be connect directly
	staticNodes []string
}

func NewConfig(listenAddress, publicAddress, dataDir, peerKey string, bootNodes, bootSeed []string, netId int,
	discover bool,
	name string, maxPeers, minPeers, maxInboundRatio, maxPendingPeers int, staticNodes []string) (*Config, error) {

	cfg, err := discovery.NewConfig(listenAddress, publicAddress, dataDir, peerKey, bootNodes, bootSeed, netId)
	if err != nil {
		return nil, err
	}

	if name == "" {
		name = DefaultNodeName
	}
	if maxPeers == 0 {
		maxPeers = DefaultMaxPeers
	}
	if minPeers == 0 {
		minPeers = DefaultMinPeers
	}
	if maxPeers < minPeers {
		maxPeers = minPeers
	}
	if maxInboundRatio == 0 {
		maxInboundRatio = DefaultMaxInboundRatio
	}
	if maxPendingPeers == 0 {
		maxPendingPeers = DefaultMaxPendingPeers
	}

	p2pConfig := &Config{
		Config:          cfg,
		discover:        discover,
		name:            name,
		minPeers:        minPeers,
		maxPendingPeers: maxPendingPeers,
		staticNodes:     staticNodes,
	}

	p2pConfig.maxPeers = make(map[Level]int)
	p2pConfig.maxPeers[Inbound] = maxPeers / maxInboundRatio
	p2pConfig.maxPeers[Outbound] = maxPeers - p2pConfig.maxPeers[Inbound]
	p2pConfig.maxPeers[Superior] = DefaultSuperiorPeers

	return p2pConfig, nil
}
