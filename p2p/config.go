package p2p

import (
	"github.com/vitelabs/go-vite/crypto"
	"encoding/hex"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"log"
)

var pubs = [...]string{
	"33e43481729850fc66cef7f42abebd8cb2f1c74f0b09a5bf03da34780a0a5606",
	"7194af5b7032cb470c41b313e2675e2c3ba3377e66617247012b8d638552fb17",
	"087c45631c3ec9a5dbd1189084ee40c8c4c0f36731ef2c2cb7987da421d08ba9",
	"7c6a2b920764b6dddbca05bb6efa1c9bcd90d894f6e9b107f137fc496c802346",
	"2840979ae06833634764c19e72e6edbf39595ff268f558afb16af99895aba3d8",
}

func pickPub(data, sign string) ed25519.PublicKey {
	payload := []byte(data)
	sig, err := hex.DecodeString(sign)
	if err != nil {
		log.Fatalf("error sign from cmd: %v\n", err)
	}

	for _, str := range pubs {
		pub, err := hex.DecodeString(str)
		if err != nil {
			continue
		}

		valid, err := crypto.VerifySig(pub, payload, sig)
		if err != nil {
			continue
		}

		if valid {
			return pub
		}
	}

	return nil
}
