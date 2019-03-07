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

package discovery

import (
	"os"

	"github.com/vitelabs/go-vite/p2p2/vnode"

	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	DefaultNetID         = 3
	DefaultListenAddress = "0.0.0.0:8483"
	PrivKeyFileName      = "peer.key"
)

type Config struct {
	ListenAddress string // TCP and UDP listen address

	DataDir string // the directory for storing p2p data, like nodes

	PeerKey ed25519.PrivateKey // use for encrypt message, the corresponding public key use for NodeID

	Node vnode.Node

	BootNodes []string // nodes as discovery seed
}

func (cfg *Config) GetPeerKey() error {
	if len(cfg.PeerKey) == 0 {

	}

	return nil
}

func getPeerKey(filename string) (priv ed25519.PrivateKey, err error) {
	var fd *os.File
	fd, err = os.Open(filename)

	// open file error
	if err != nil {
		if _, priv, err = ed25519.GenerateKey(nil); err != nil {
			return
		}

		if fd, err = os.Create(filename); err == nil {
			defer fd.Close()
		}
	} else {
		defer fd.Close()

		priv = make([]byte, 64)
		var n int
		if n, err = fd.Read(priv); err != nil || n != len(priv) {
			// read file error
			if _, priv, err = ed25519.GenerateKey(nil); err != nil {
				return
			}
		}
	}

	if fd != nil {
		fd.Write(priv)
	}

	return
}

// EnsureConfig will set default value to fields missing value of Config
// NOT include PeerKey
func EnsureConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = new(Config)
	}

	if cfg.NetID == 0 {
		cfg.NetID = DefaultNetID
	}

	if cfg.ListenAddress == "" {
		cfg.ListenAddress = DefaultListenAddress
	}

	return cfg
}
