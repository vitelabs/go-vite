package p2p

import (
	"fmt"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path/filepath"
)

const privKeyFileName = "priv.key"

var log = log15.New("module", "p2p/config")

func getServerKey(p2pDir string) (pub ed25519.PublicKey, priv ed25519.PrivateKey, err error) {
	pub = make([]byte, 32)
	priv = make([]byte, 64)

	privKeyFile := filepath.Join(p2pDir, privKeyFileName)

	fd, err := os.Open(privKeyFile)
	getKeyOK := true

	if err != nil {
		getKeyOK = false
		log.Info("cannot read p2p priv.key, use random key", "error", err)
		pub, priv, err = ed25519.GenerateKey(nil)
		if err != nil {
			return nil, nil, err
		}

		fd, err = os.Create(privKeyFile)
		if err != nil {
			log.Info("create p2p priv.key fail", "error", err)
		} else {
			defer fd.Close()
		}
	} else {
		defer fd.Close()

		n, err := fd.Read([]byte(priv))

		if err != nil {
			getKeyOK = false
			log.Info("cannot read p2p priv.key, use random key", "error", err)
		}

		if n != len(priv) {
			log.Info("read incompelete p2p priv.key, use random key")
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
		fmt.Println(fd == nil)
		if err != nil {
			log.Info("write privateKey to p2p priv.key fail", "error", err)
		} else if n != len(priv) {
			log.Info("write incompelete privateKey to p2p priv.key")
		} else {
			log.Info("write privateKey to p2p priv.key")
		}
	}

	return
}
