package api

import (
	"encoding/binary"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/pow/remote"
	"math/big"
)

type Pow struct {
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
