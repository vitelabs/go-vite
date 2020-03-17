package api

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/crypto"

	"github.com/vitelabs/go-vite/common/hexutil"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/pow/remote"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/quota"
)

type Pow struct {
	vite   *vite.Vite
	pubKey []byte
}

func NewPow(vite *vite.Vite) *Pow {
	var pubKey []byte
	pub := vite.Config().SecretPub
	if pub != nil {
		p, err := hexutil.Decode(*pub)
		if err != nil {
			panic(err)
		}
		pubKey = p
	} else {
		p, err := hexutil.Decode("f4b37ea2a04d012835820fc480e0b87150f76112c87fce51412815bc90476d4e")
		if err != nil {
			panic(err)
		}
		pubKey = p
	}
	return &Pow{
		vite:   vite,
		pubKey: pubKey,
	}
}

func (p Pow) GetPowNoncePrivate(address types.Address, height uint64, difficulty string, data types.Hash, timestamp uint64, sig []byte) (result []byte, e error) {
	log.Info("GetPowNoncePrivate ", "address", address, "height", height,
		"difficulty", difficulty, "data", data.Hex(), "timestamp", timestamp, "sig", hexutil.Encode(sig))
	s := time.Now()
	defer func() {
		log.Info("GetPowNoncePrivate ", "address", address, "height", height,
			"difficulty", difficulty, "data", data.Hex(), "timestamp", timestamp, "sig", hexutil.Encode(sig),
			"err", e, "duration_ms", time.Now().Sub(s).Milliseconds())
	}()

	flag, err := crypto.VerifySig(p.pubKey, []byte(fmt.Sprintf("%d", timestamp)), sig)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, errors.New("auth fail")
	}
	realDifficulty, ok := new(big.Int).SetString(difficulty, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}
	return pow.GetPowNonce(realDifficulty, data)
}

func (p Pow) GetPowNonce(difficulty string, data types.Hash) ([]byte, error) {
	log.Info("GetPowNonce")

	if pow.VMTestParamEnabled {
		log.Info("use defaultTarget to calc")
		return pow.GetPowNonce(nil, data)
	}

	realDifficulty, ok := new(big.Int).SetString(difficulty, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}

	if _, _, isCongestion := quota.CalcQc(p.vite.Chain(), p.vite.Chain().GetLatestSnapshotBlock().Height); isCongestion {
		return nil, ErrPoWNotSupportedUnderCongestion
	}

	work, e := remote.GenerateWork(data.Bytes(), realDifficulty)
	if e != nil {
		return nil, e
	}

	nonceStr := *work
	nonceBig, ok := new(big.Int).SetString(nonceStr, 16)
	if !ok {
		return nil, errors.New("wrong nonce str")
	}
	nonceUint64 := nonceBig.Uint64()
	nn := make([]byte, 8)
	binary.LittleEndian.PutUint64(nn[:], nonceUint64)

	bd, ok := new(big.Int).SetString(difficulty, 10)
	if !ok {
		return nil, errors.New("wrong nonce difficulty")
	}

	if !pow.CheckPowNonce(bd, nn, data.Bytes()) {
		return nil, errors.New("check nonce failed")
	}

	return nn, nil
}

func (p Pow) CancelPow(data types.Hash) error {
	if err := remote.CancelWork(data.Bytes()); err != nil {
		return errors.New("pow cancel failed")
	}
	return nil
}
