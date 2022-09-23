package api

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/common/hexutil"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto"
	"github.com/vitelabs/go-vite/v2/pow"
	"github.com/vitelabs/go-vite/v2/pow/remote"
	"github.com/vitelabs/go-vite/v2/vm/quota"
)

type UtilApi struct {
	vite *vite.Vite
}

func NewUtilApi(vite *vite.Vite) *UtilApi {
	return &UtilApi{
		vite: vite,
	}
}

func (p UtilApi) GetPoWNonce(difficulty string, data types.Hash) ([]byte, error) {
	log.Info("GetPowNonce")

	if pow.VMTestParamEnabled {
		log.Info("use defaultTarget to calc")
		return pow.GetPowNonce(nil, data)
	}

	realDifficulty, ok := new(big.Int).SetString(difficulty, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}

	difficultyCap := big.NewInt(8034995932)
	difficultyCap.Mul(difficultyCap, big.NewInt(50))
	if realDifficulty.Cmp(difficultyCap) > 0 {
		return nil, ErrDifficultyTooLarge
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

// Pow Plan Ref[] todo
func (p Pow) GetPowNoncePrivate(address types.Address, height uint64, difficulty string, data types.Hash, timestamp uint64, sig []byte, cnt uint64) (result []byte, e error) {
	log.Info("GetPowNoncePrivate ", "address", address, "height", height,
		"difficulty", difficulty, "data", data.Hex(), "timestamp", timestamp, "sig", hexutil.Encode(sig))
	s := time.Now()
	defer func() {
		log.Info("GetPowNoncePrivate ", "address", address, "height", height,
			"difficulty", difficulty, "data", data.Hex(), "timestamp", timestamp, "sig", hexutil.Encode(sig),
			"err", e, "duration_ms", time.Now().Sub(s).Nanoseconds())
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
	nonce, _, err := pow.MapPowNonce2(realDifficulty, data, cnt)
	return nonce, err
}
