package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vitelabs/go-vite/crypto/ed25519"
)

const (
	DefaultSingle = false

	DefaultNodeName = "vite-node"

	DefaultNetID = 3

	DefaultListenInterface = "0.0.0.0"
	DefaultPort            = 8483
	DefaultFilePort        = 8484

	DefaultDiscover = true

	DefaultMaxPeers        = 60
	DefaultMaxInboundRatio = 2
	DefaultMinPeers        = 5
	DefaultMaxPendingPeers = 10

	DefaultNetDirName = "net"
	PeerKeyFileName   = "peerKey"

	DefaultForwardStrategy = "cross"
	DefaultAccessControl   = "any"
)

type Net struct {
	Single bool

	// Name is our node name, NO need to be unique in the whole network, just for readability, default is `vite-node`
	Name string

	// NetID is to mark which network our node in, nodes from different network can`t connect each other
	NetID int

	ListenInterface string

	Port int

	FilePort int

	// PublicAddress is the network address can be access by other nodes, usually is the public Internet address
	PublicAddress string

	FilePublicAddress string

	// DataDir is the directory to storing p2p data, if is null-string, will use memory as database
	DataDir string

	// PeerKey is to encrypt message, the corresponding public key use for NodeID, MUST NOT be revealed
	PeerKey string

	// Discover means whether discover other nodes in the networks, default true
	Discover bool

	// BootNodes are roles as network entrance. Node can discovery more other nodes by send UDP query BootNodes,
	// but not create a TCP connection to BootNodes directly
	BootNodes []string

	// BootSeeds are the address where can query BootNodes, is a more flexible option than BootNodes
	BootSeeds []string

	// StaticNodes will be connect directly
	StaticNodes []string

	MaxPeers int

	MaxInboundRatio int

	// MinPeers server will keep finding nodes and try to connect until number of peers is larger than `MinPeers`,
	// default 5
	MinPeers int

	// MaxPendingPeers how many inbound peers can be connect concurrently, more inbound connection will be blocked
	// this value is for defend DDOS attack, default 10
	MaxPendingPeers int

	ForwardStrategy string

	AccessControl   string
	AccessAllowKeys []string
	AccessDenyKeys  []string

	BlackBlockHashList []string
	WhiteBlockList     []string

	MineKey ed25519.PrivateKey
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

func (net *Net) Init() (privateKey ed25519.PrivateKey, err error) {
	err = os.MkdirAll(net.DataDir, 0700)
	if err != nil {
		return
	}

	if net.PeerKey == "" {
		if net.DataDir == "" {
			_, privateKey, err = ed25519.GenerateKey(nil)
		} else {
			keyFile := filepath.Join(net.DataDir, PeerKeyFileName)
			privateKey, err = getPeerKey(keyFile)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to generate peerKey: %v", err)
		}
	} else {
		privateKey, err = ed25519.HexToPrivateKey(net.PeerKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PeerKey: %v", err)
		}
	}

	return
}
