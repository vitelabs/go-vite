package generator

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"time"
)

type IncomingMessage struct {
	BlockType byte

	AccountAddress types.Address
	ToAddress      *types.Address
	FromBlockHash  *types.Hash

	TokenId *types.TokenTypeId
	Amount  *big.Int
	Fee     *big.Int
	Nonce   []byte
	Data    []byte
}

func (im *IncomingMessage) ToBlock() (block *ledger.AccountBlock, err error) {
	switch {
	case im.BlockType == ledger.BlockTypeSendCall:
		block.BlockType = im.BlockType
		block.FromBlockHash = types.Hash{}

		block.Nonce = im.Nonce
		block.Data = im.Data

		if im.ToAddress != nil {
			block.ToAddress = *im.ToAddress
		} else {
			return nil, errors.New("BlockTypeSendCall's ToAddress can't be nil")
		}

		if im.Amount == nil {
			block.Amount = big.NewInt(0)
		} else {

			if block.Amount.Sign() < 0 || block.Amount.BitLen() > math.MaxBigIntLen {
				return nil, errors.New("block.Amount out of bounds")
			} else {
				block.Amount = im.Amount
			}
		}

		if im.TokenId != nil {
			block.TokenId = *im.TokenId
		} else {
			return nil, errors.New("BlockTypeSendCall's TokenId can't be nil")
		}

		if im.Fee == nil {
			block.Fee = big.NewInt(0)
		} else {
			if block.Fee.Sign() < 0 || block.Fee.BitLen() > math.MaxBigIntLen {
				return nil, errors.New("block.Fee out of bounds")
			}
			block.Fee = im.Fee
		}

	case im.BlockType == ledger.BlockTypeSendCreate:
		block.BlockType = im.BlockType
		block.FromBlockHash = types.Hash{}

		block.Nonce = im.Nonce

		if im.ToAddress != nil {
			block.ToAddress = *im.ToAddress
		} else {
			return nil, errors.New("BlockTypeSendCall's ToAddress can't be nil")
		}

		if im.Amount == nil {
			block.Amount = big.NewInt(0)
		} else {
			block.Amount = im.Amount
		}

		// fixme
		if im.TokenId != nil {
			block.TokenId = *im.TokenId
		} else {
			return nil, errors.New("BlockTypeSendCreate's TokenId can't be nil")
		}

		if im.Fee == nil {
			block.Fee = big.NewInt(0)
		} else {
			block.Fee = im.Fee
		}

		if len(im.Data) > 0 {
			block.Data = im.Data
		} else {
			return nil, errors.New("BlockTypeSendCreate's Data can't be nil")
		}

	default:
		block.BlockType = ledger.BlockTypeReceive
		block.ToAddress = types.Address{}

		block.Nonce = im.Nonce
		block.Data = im.Data

		if im.FromBlockHash != nil {
			block.FromBlockHash = *im.FromBlockHash
		} else {
			return nil, errors.New("BlockTypeReceive's FromBlockHash can't be nil")
		}
	}
	return block, err
}

type ConsensusMessage struct {
	SnapshotHash types.Hash
	Timestamp    time.Time
	Producer     types.Address
	gid          types.Gid
}
