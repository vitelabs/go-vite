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
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	DefaultNetID           = 3
	DefaultListenAddress   = "0.0.0.0:8483"
	PeerKeyFileName        = "key"
	DefaultPort            = 8483
	DefaultListenInterface = "0.0.0.0"
	DefaultBucketSize      = 32
	DefaultBucketCount     = 32
)

type Config struct {
	// ListenAddress is the network address where socket listen on, usually is the inner address
	// default value is "0.0.0.0:8483"
	ListenAddress string

	// PublicAddress is the network address can be access by other nodes, usually is the public Internet address
	PublicAddress string

	// DataDir is the directory to storing p2p data, if is null-string, will use memory as database
	DataDir string

	// PeerKey is to encrypt message, the corresponding public key use for NodeID, MUST NOT be revealed
	PeerKey string

	// privateKey is derived from PeerKey or read from file, or generate randomly
	privateKey ed25519.PrivateKey

	// node represent our endpoint, NodeID is derived from PeerKey
	node *vnode.Node

	// BootNodes are roles as network entrance. Node can discovery more other nodes by send UDP query BootNodes,
	// but not create a TCP connection to BootNodes directly
	BootNodes []string

	// BootSeeds are the address where can query BootNodes, is a more flexible option than BootNodes
	BootSeeds []string

	// NetID is to mark which network our node in, nodes from different network can`t connect each other
	NetID int

	BucketCount int
	BucketSize  int
}

// Node MUST NOT be invoked before Ensure
func (cfg *Config) Node() *vnode.Node {
	return cfg.node
}

func (cfg *Config) PrivateKey() ed25519.PrivateKey {
	return cfg.privateKey
}

func getPeerKey(filename string) (privateKey ed25519.PrivateKey, err error) {
	var fd *os.File
	fd, err = os.Open(filename)

	// open file error
	if err != nil {
		fd = nil

		if _, privateKey, err = ed25519.GenerateKey(nil); err != nil {
			return
		}

		if fd, err = os.Create(filename); err == nil {
			defer func() {
				_ = fd.Close()
			}()
		}
	} else {
		defer func() {
			_ = fd.Close()
		}()

		privateKey = make(ed25519.PrivateKey, ed25519.PrivateKeySize)
		var n int
		if n, err = fd.Read(privateKey); err != nil || n != len(privateKey) {
			// read file error
			if _, privateKey, err = ed25519.GenerateKey(nil); err != nil {
				return
			}
		}
	}

	if fd != nil {
		_, _ = fd.Write(privateKey)
	}

	return
}

// Ensure will set default value to missing fields and construct node. MUST be invoked before use.
// Will generate a random PeerKey and store in local file, `${DataDir}/peer.key`, if missing one.
func (cfg *Config) Ensure() (err error) {
	if cfg.NetID == 0 {
		cfg.NetID = DefaultNetID
	}

	if cfg.ListenAddress == "" {
		cfg.ListenAddress = DefaultListenAddress
	}

	if cfg.BucketCount == 0 {
		cfg.BucketCount = DefaultBucketCount
	}

	if cfg.BucketSize == 0 {
		cfg.BucketSize = DefaultBucketSize
	}

	if cfg.PeerKey == "" {
		if cfg.DataDir == "" {
			_, cfg.privateKey, err = ed25519.GenerateKey(nil)
		} else {
			keyFile := filepath.Join(cfg.DataDir, PeerKeyFileName)
			cfg.privateKey, err = getPeerKey(keyFile)
		}

		if err != nil {
			return fmt.Errorf("failed to generate peerKey: %v", err)
		}
	} else {
		cfg.privateKey, err = ed25519.HexToPrivateKey(cfg.PeerKey)
		if err != nil {
			return fmt.Errorf("failed to parse PeerKey: %v", err)
		}
	}

	id, _ := vnode.Bytes2NodeID(cfg.privateKey.PubByte())

	var e vnode.EndPoint
	if cfg.PublicAddress != "" {
		e, err = vnode.ParseEndPoint(cfg.PublicAddress)
		if err != nil {
			return fmt.Errorf("failed to parse PublicAddress: %v", err)
		}
	}

	cfg.node = &vnode.Node{
		ID:       id,
		EndPoint: e,
		Net:      cfg.NetID,
		Ext:      nil,
	}

	return nil
}
