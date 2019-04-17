package generator

import (
	"errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pow"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func IncomingMessageToBlock(vmDb vm_db.VmDb, im *IncomingMessage) (*ledger.AccountBlock, error) {
	block := &ledger.AccountBlock{
		BlockType:      im.BlockType,
		AccountAddress: im.AccountAddress,
		// after vm
		Quota:         0,
		SendBlockList: nil,
		LogHash:       nil,
		Hash:          types.Hash{},
		Signature:     nil,
		PublicKey:     nil,
	}
	switch im.BlockType {
	case ledger.BlockTypeSendCreate, ledger.BlockTypeSendRefund, ledger.BlockTypeSendReward, ledger.BlockTypeSendCall:
		block.Data = im.Data

		block.FromBlockHash = types.Hash{}
		if im.ToAddress != nil {
			block.ToAddress = *im.ToAddress
		} else if im.BlockType != ledger.BlockTypeSendCreate {
			return nil, errors.New("pack send failed, toAddress can't be nil")
		}

		zero_amount := big.NewInt(0)
		if im.TokenId == nil || *im.TokenId == types.ZERO_TOKENID {
			if im.Amount != nil && im.Amount.Cmp(zero_amount) <= 0 {
				return nil, errors.New("pack send failed, tokenId can't be empty when amount have actual value")
			}
			block.Amount = zero_amount
			block.TokenId = types.ZERO_TOKENID
		} else {
			if im.Amount == nil {
				block.Amount = zero_amount
			} else {
				if im.Amount.Sign() < 0 || im.Amount.BitLen() > math.MaxBigIntLen {
					return nil, errors.New("pack send failed, amount out of bounds")
				}
				block.Amount = im.Amount
			}
			block.TokenId = *im.TokenId
		}

		if im.Fee == nil {
			block.Fee = big.NewInt(0)
		} else {
			if im.Fee.Sign() < 0 || im.Fee.BitLen() > math.MaxBigIntLen {
				return nil, errors.New("pack send failed, fee out of bounds")
			}
			block.Fee = im.Fee
		}
		// PrevHash, Height
		prevBlock, err := vmDb.PrevAccountBlock()
		if err != nil {
			return nil, err
		}
		if prevBlock == nil {
			return nil, errors.New("account address doesn't exist")
		}

		block.Height = prevBlock.Height + 1
		block.PrevHash = prevBlock.Hash
	case ledger.BlockTypeReceive:
		block.Data = nil

		block.ToAddress = types.Address{}
		if im.FromBlockHash != nil && *im.FromBlockHash != types.ZERO_HASH {
			block.FromBlockHash = *im.FromBlockHash
		} else {
			return nil, errors.New("pack recvBlock failed, cause sendBlock.Hash is invaild")
		}

		if im.Amount != nil && im.Amount.Cmp(big.NewInt(0)) != 0 {
			return nil, errors.New("pack recvBlock failed, amount is invalid")
		}
		if im.TokenId != nil && *im.TokenId != types.ZERO_TOKENID {
			return nil, errors.New("pack recvBlock failed, cause tokenId is invaild")
		}
		if im.Fee != nil && im.Fee.Cmp(big.NewInt(0)) != 0 {
			return nil, errors.New("pack recvBlock failed, fee is invalid")
		}

		// PrevHash, Height
		prevBlock, err := vmDb.PrevAccountBlock()
		if err != nil {
			return nil, err
		}
		var prevHash types.Hash
		var preHeight uint64 = 0
		if prevBlock != nil {
			prevHash = prevBlock.Hash
			preHeight = prevBlock.Height
		}
		block.PrevHash = prevHash
		block.Height = preHeight + 1

	default:
		//ledger.BlockTypeReceiveError:
		return nil, errors.New("generator can't solve this block type " + string(im.BlockType))
	}
	// Difficulty,Nonce
	if im.Difficulty != nil {
		nonce, err := pow.GetPowNonce(im.Difficulty, types.DataHash(append(block.AccountAddress.Bytes(), block.PrevHash.Bytes()...)))
		if err != nil {
			return nil, err
		}
		block.Nonce = nonce[:]
		block.Difficulty = im.Difficulty
	}
	return block, nil
}

type IncomingMessage struct {
	BlockType byte

	AccountAddress types.Address
	ToAddress      *types.Address
	FromBlockHash  *types.Hash

	TokenId *types.TokenTypeId
	Amount  *big.Int
	Fee     *big.Int
	Data    []byte

	Difficulty *big.Int
}
