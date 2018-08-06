package p2p

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"log"
)

var pubs = [...]string{
	"33e43481729850fc66cef7f42abebd8cb2f1c74f0b09a5bf03da34780a0a5606",
	"7194af5b7032cb470c41b313e2675e2c3ba3377e66617247012b8d638552fb17",
	"087c45631c3ec9a5dbd1189084ee40c8c4c0f36731ef2c2cb7987da421d08ba9",
	"7c6a2b920764b6dddbca05bb6efa1c9bcd90d894f6e9b107f137fc496c802346",
	"2840979ae06833634764c19e72e6edbf39595ff268f558afb16af99895aba3d8",
	"298a693584e4fceebb3f7abab2c25e7a6c6b911f15c12362b23144dd72822c02",
	"75f962a81a6d52d9dcc830ff7dc2f21c424eeed4a0f2e5bab8f39f44df833153",
	"491d7a992cad4ba82ea10ad0f5da86a2e40f4068918bd50d8faae1f1b69d8510",
	"c3debe5fc3f8839c4351834d454a0940872f973ad0d332811f0d9e84953cfdc2",
	"ba351c5df80eea3d78561507e40b3160ade4daf20571c5a011ca358b81d630f7",
	"a1d437148c48a44b709b0a0c1e99c847bb4450384a6eea899e35e78a1e54c92b",
	"657f6706b95005a33219ae26944e6db15edfe8425b46fa73fb6cae57707b4403",
	"20fd0ec94071ea99019d3c18a5311993b8fa91920b36803e5558783ca346cec1",
	"8669feb2fdeeedfbbfe8754050c0bd211a425e3b999d44856867d504cf13243e",
	"802d8033f6adea154e5fa8b356a45213e8a4e5e85e54da19688cb0ad3520db2b",
	"180af9c90f8444a452a9fb53f3a1975048e9b23ec492636190f9741ed3888a62",
	"6c8ce60b87199d46f7524d7da5a06759c054b9192818d0e8b211768412efec51",
	"3b3e7164827eb339548b9c895fc56acee7980a0044b5bef2e6f465e030944e40",
	"de1f3a591b551591fb7d20e478da0371def3d62726b90ffca6932e38a25ebe84",
	"9df2e11399398176fa58638592cf1b2e0e804ae92ac55f09905618fdb239c03c",
	"0fbb7d7a2cf44755019b7ab939e33f75a3bc9540997df2146ac3a977efaf2858",
	"24a19ee5109bd5c6229df087bf295f543002f1f00026f8d0ade9dcccc36fcb6f",
	"dcf3e93b0ab38e1f359af9772d493e20414fbd552b21768f9837ae7b0f01f0cb",
	"58b9ac27b5578d0632ef9ac204298b3eba5047488c10cce1eb2e191db1c054f1",
	"fdddbd58b6fd9059d7751b16f51fb1db77cb2f0801aa6c447eefacbcecb6945d",
}

func pickPub(data, sign string) ed25519.PublicKey {
	payload := []byte(data)
	sig, err := hex.DecodeString(sign)
	if err != nil {
		log.Fatalf("invalid signature from cmd: %v\n", err)
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
