package p2p

import (
	"os"
	"path/filepath"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/network"
)

const (
	DefaultMaxPeers        uint = 50
	DefaultMaxPendingPeers uint = 20
	DefaultMaxInboundRatio uint = 2
	DefaultPort            uint = 8483
	DefaultNetID                = network.Aquarius
)

const Dirname = "p2p"
const privKeyFileName = "priv.key"

func getServerKey(p2pDir string) (priv ed25519.PrivateKey, err error) {
	privKeyFile := filepath.Join(p2pDir, privKeyFileName)

	var fd *os.File
	fd, err = os.Open(privKeyFile)

	// open file error
	if err != nil {
		if _, priv, err = ed25519.GenerateKey(nil); err != nil {
			return
		}

		if fd, err = os.Create(privKeyFile); err == nil {
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

func EnsureConfig(cfg *Config) *Config {
	if cfg == nil {
		cfg = new(Config)
	}

	if cfg.NetID == 0 {
		cfg.NetID = DefaultNetID
	}

	if cfg.MaxPeers == 0 {
		cfg.MaxPeers = DefaultMaxPeers
	}

	if cfg.MaxPendingPeers == 0 {
		cfg.MaxPendingPeers = DefaultMaxPendingPeers
	}

	if cfg.MaxInboundRatio == 0 {
		cfg.MaxInboundRatio = DefaultMaxInboundRatio
	}

	if cfg.Port == 0 {
		cfg.Port = DefaultPort
	}

	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(common.DefaultDataDir(), Dirname)
	}

	if cfg.PeerKey == nil {
		priv, err := getServerKey(cfg.DataDir)

		if err != nil {
			panic(err)
		} else {
			cfg.PeerKey = priv
		}
	}

	return cfg
}

func parseNodes(urls []string) (nodes []*discovery.Node) {
	nodes = make([]*discovery.Node, len(urls))

	i := 0
	for _, nodeURL := range urls {
		if node, err := discovery.ParseNode(nodeURL); err == nil {
			nodes[i] = node
			i++
		}
	}

	return nodes[:i]
}
