package p2p

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"os"
	"path/filepath"
)

var firmNodes = []string{
	"vnode://33e43481729850fc66cef7f42abebd8cb2f1c74f0b09a5bf03da34780a0a5606@150.109.40.224:8483",
	"vnode://7194af5b7032cb470c41b313e2675e2c3ba3377e66617247012b8d638552fb17@150.109.62.152:8483",
	"vnode://087c45631c3ec9a5dbd1189084ee40c8c4c0f36731ef2c2cb7987da421d08ba9@150.109.104.203:8483",
	"vnode://7c6a2b920764b6dddbca05bb6efa1c9bcd90d894f6e9b107f137fc496c802346@150.109.101.200:8483",
	"vnode://2840979ae06833634764c19e72e6edbf39595ff268f558afb16af99895aba3d8@150.109.105.192:8483",
	"vnode://298a693584e4fceebb3f7abab2c25e7a6c6b911f15c12362b23144dd72822c02@119.28.224.63:8483",
	"vnode://75f962a81a6d52d9dcc830ff7dc2f21c424eeed4a0f2e5bab8f39f44df833153@150.109.40.23:8483",
	"vnode://491d7a992cad4ba82ea10ad0f5da86a2e40f4068918bd50d8faae1f1b69d8510@150.109.49.145:8483",
	"vnode://c3debe5fc3f8839c4351834d454a0940872f973ad0d332811f0d9e84953cfdc2@150.109.103.170:8483",
	//"vnode://ba351c5df80eea3d78561507e40b3160ade4daf20571c5a011ca358b81d630f7@150.109.104.53:8483",
	//"vnode://a1d437148c48a44b709b0a0c1e99c847bb4450384a6eea899e35e78a1e54c92b@150.109.103.236:8483",
	//"vnode://657f6706b95005a33219ae26944e6db15edfe8425b46fa73fb6cae57707b4403@150.109.32.116:8483",
	//"vnode://20fd0ec94071ea99019d3c18a5311993b8fa91920b36803e5558783ca346cec1@150.109.102.180:8483",
	//"vnode://8669feb2fdeeedfbbfe8754050c0bd211a425e3b999d44856867d504cf13243e@150.109.59.74:8483",
	//"vnode://802d8033f6adea154e5fa8b356a45213e8a4e5e85e54da19688cb0ad3520db2b@150.109.105.23:8483",
	//"vnode://180af9c90f8444a452a9fb53f3a1975048e9b23ec492636190f9741ed3888a62@150.109.105.145:8483",
	//"vnode://6c8ce60b87199d46f7524d7da5a06759c054b9192818d0e8b211768412efec51@150.109.105.26:8483",
	//"vnode://3b3e7164827eb339548b9c895fc56acee7980a0044b5bef2e6f465e030944e40@150.109.62.157:8483",
	//"vnode://de1f3a591b551591fb7d20e478da0371def3d62726b90ffca6932e38a25ebe84@150.109.38.29:8483",
	//"vnode://9df2e11399398176fa58638592cf1b2e0e804ae92ac55f09905618fdb239c03c@150.109.40.169:8483",
}

const (
	DefaultMaxPeers        uint = 50
	DefaultMaxPendingPeers uint = 20
	DefaultMaxInboundRatio uint = 2
	DefaultPort            uint = 8483
	DefaultNetID                = Aquarius
)

const P2PDir = "p2p"
const privKeyFileName = "priv.key"

func getServerKey(p2pDir string) (pub ed25519.PublicKey, priv ed25519.PrivateKey, err error) {
	pub = make([]byte, 32)
	priv = make([]byte, 64)

	privKeyFile := filepath.Join(p2pDir, privKeyFileName)

	fd, err := os.Open(privKeyFile)
	getKeyOK := true

	if err != nil {
		getKeyOK = false
		p2pServerLog.Info("cannot read p2p priv.key, use random key", "error", err)
		pub, priv, err = ed25519.GenerateKey(nil)
		if err != nil {
			return nil, nil, err
		}

		fd, err = os.Create(privKeyFile)
		if err != nil {
			p2pServerLog.Info("create p2p priv.key fail", "error", err)
		} else {
			defer fd.Close()
		}
	} else {
		defer fd.Close()

		n, err := fd.Read([]byte(priv))

		if err != nil {
			getKeyOK = false
			p2pServerLog.Info("cannot read p2p priv.key, use random key", "error", err)
		}

		if n != len(priv) {
			p2pServerLog.Info("read incompelete p2p priv.key, use random key")
			getKeyOK = false
		}

		if !getKeyOK {
			pub, priv, err = ed25519.GenerateKey(nil)
			if err != nil {
				return nil, nil, err
			}
		} else {
			pub = priv.PubByte()
		}
	}

	if !getKeyOK {
		n, err := fd.Write([]byte(priv))
		if err != nil {
			p2pServerLog.Info("write privateKey to p2p priv.key fail", "error", err)
		} else if n != len(priv) {
			p2pServerLog.Info("write incompelete privateKey to p2p priv.key")
		} else {
			p2pServerLog.Info("write privateKey to p2p priv.key")
		}
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

	if cfg.Database == "" {
		cfg.Database = filepath.Join(common.DefaultDataDir(), P2PDir)
	}

	if cfg.PrivateKey == nil {
		_, priv, err := getServerKey(cfg.Database)
		if err != nil {
			p2pServerLog.Crit("generate privateKey error", "error", err)
		} else {
			cfg.PrivateKey = priv
		}
	}

	return cfg
}

func addFirmNodes(bootnodes []string) (nodes []*discovery.Node) {
	nodes = make([]*discovery.Node, 0, len(bootnodes)+len(firmNodes))

	//for _, list := range [][]string{firmNodes, bootnodes} {
	//	for _, nodeURL := range list {
	//		node, err := discovery.ParseNode(nodeURL)
	//		if err == nil {
	//			nodes = append(nodes, node)
	//		}
	//	}
	//}

	for _, nodeURL := range bootnodes {
		node, err := discovery.ParseNode(nodeURL)
		if err == nil {
			nodes = append(nodes, node)
		}
	}

	return
}

func copyNodes(nodes []*discovery.Node) []*discovery.Node {
	cps := make([]*discovery.Node, len(nodes))

	for i, n := range nodes {
		cps[i] = &(*n)
	}

	return cps
}
