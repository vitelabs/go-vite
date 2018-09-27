package node

import (
	"encoding/hex"
	"fmt"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	Name    string `json:"ConfigName"`
	Version string `json:"ConfigVersion"`
	DataDir string `json:"dataDir"`

	P2P config.P2P `json:"P2P"`

	KeyStoreDir string `json:"KeyStoreDir"`

	RPCEnabled bool `json:"RPCEnabled"`
	IPCEnabled bool `json:"IPCEnabled"`
	WSEnabled  bool `json:"WSEnabled"`

	IPCPath          string   `json:"IPCPath"`
	HttpHost         string   `json:"HttpHost"`
	HttpPort         int      `json:"HttpPort"`
	HttpVirtualHosts []string `json:"HttpVirtualHosts"`
	WSHost           string   `json:"WSHost"`
	WSPort           int      `json:"WSPort"`
}

// IPCEndpoint resolves an IPC endpoint based on a configured value, taking into
// account the set data folders as well as the designated platform we're currently
// running on.
//TODO use default settings
func (c *Config) IPCEndpoint() string {
	// Short circuit if IPC has not been enabled
	if c.IPCPath == "" {
		return ""
	}
	// On windows we can only use plain top-level pipes
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(c.IPCPath, `\\.\pipe\`) {
			return c.IPCPath
		}
		return `\\.\pipe\` + c.IPCPath
	}
	// Resolve names into the data directory full paths otherwise
	if filepath.Base(c.IPCPath) == c.IPCPath {
		if c.DataDir == "" {
			return filepath.Join(os.TempDir(), c.IPCPath)
		}
		return filepath.Join(c.DataDir, c.IPCPath)
	}
	return c.IPCPath
}

func (c *Config) RunLogDir() string {
	return filepath.Join(c.DataDir, "runlog")
}

func (c *Config) NewRunLogDirFile() (string, error) {
	filename := time.Now().Format("2006-01-02") + ".log"
	if err := os.MkdirAll(c.RunLogDir(), 0777); err != nil {
		return "", err
	}
	return filepath.Join(c.RunLogDir(), filename), nil

}

func (c *Config) makeViteConfig() config.Config {
	return config.Config{
		P2P:     &c.P2P,
		DataDir: c.DataDir,
	}
}

func (c *Config) makeP2PConfig() p2p.Config {
	return p2p.Config{
		Name:            c.P2P.Name,
		NetID:           p2p.NetworkID(c.P2P.NetID),
		MaxPeers:        c.P2P.MaxPeers,
		MaxPendingPeers: c.P2P.MaxPendingPeers,
		MaxInboundRatio: c.P2P.MaxPassivePeersRatio,
		Port:            c.P2P.Port,
		Database:        c.P2P.Datadir,
		PrivateKey:      c.PrivateKey(),
		//Protocols:nil,
		BootNodes: c.P2P.BootNodes,
		//KafKa:nil,
	}
}

func (c *Config) BootNodes() []*discovery.Node {

	if len(c.P2P.BootNodes) > 0 {
		var nodes []*discovery.Node
		for _, str := range c.P2P.BootNodes {
			n, err := discovery.ParseNode(str)
			if err == nil {
				return append(nodes, n)
			}
		}
	}
	return nil
}

func (c *Config) PrivateKey() ed25519.PrivateKey {

	if c.P2P.PrivateKey != "" {
		privateKey, err := hex.DecodeString(c.P2P.PrivateKey)
		if err == nil {
			return ed25519.PrivateKey(privateKey)
		}
	}

	return nil
}

// HTTPEndpoint resolves an HTTP endpoint based on the configured host interface
// and port parameters.
func (c *Config) HTTPEndpoint() string {
	if c.HttpHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.HttpHost, c.HttpPort)
}

// WSEndpoint resolves a websocket endpoint based on the configured host interface
// and port parameters.
func (c *Config) WSEndpoint() string {
	if c.WSHost == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", c.WSHost, c.WSPort)
}
