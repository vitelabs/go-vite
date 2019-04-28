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
	DirName                = "p2p"
)

// Config is the essential configuration to create a p2p server
type Config struct {
	*discovery.Config

	// Discover means whether discover other nodes in the networks, default true
	Discover bool

	// Name is our node name, NO need to be unique in the whole network, just for readability, default is `vite-node`
	Name string

	MaxPeers int

	MaxInboundRatio int

	// maxPeers means each level can accept how many peers, default:
	// Inbound: 5
	// Outbound: 5
	// Superior: 30
	maxPeers map[Level]int

	// MinPeers server will keep finding nodes and try to connect until number of peers is larger than `MinPeers`,
	// default 5
	MinPeers int

	// MaxPendingPeers how many inbound peers can be connect concurrently, more inbound connection will be blocked
	// this value is for defend DDOS attack, default 10
	MaxPendingPeers int

	// StaticNodes will be connect directly
	StaticNodes []string
}

func (cfg *Config) Ensure() (err error) {
	err = cfg.Config.Ensure()
	if err != nil {
		return
	}

	if cfg.Name == "" {
		cfg.Name = DefaultNodeName
	}

	if cfg.MaxPeers == 0 {
		cfg.MaxPeers = DefaultMaxPeers
	}

	if cfg.MinPeers == 0 {
		cfg.MinPeers = DefaultMinPeers
	}

	if cfg.MaxPeers < cfg.MinPeers {
		cfg.MaxPeers = cfg.MinPeers
	}

	if cfg.MaxInboundRatio == 0 {
		cfg.MaxInboundRatio = DefaultMaxInboundRatio
	}

	if cfg.MaxPendingPeers == 0 {
		cfg.MaxPendingPeers = DefaultMaxPendingPeers
	}

	cfg.maxPeers = make(map[Level]int)
	cfg.maxPeers[Inbound] = cfg.MaxPeers / cfg.MaxInboundRatio
	cfg.maxPeers[Outbound] = cfg.MaxPeers - cfg.maxPeers[Inbound]
	cfg.maxPeers[Superior] = DefaultSuperiorPeers

	return nil
}
