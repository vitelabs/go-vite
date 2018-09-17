package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"math/big"
)

type IncomingMessage struct {
	BlockType byte

	AccountAddress types.Address
	Passphrase     ed25519.PublicKey
	ToAddress      *types.Address
	FromBlockHash  *types.Hash

	TokenId types.TokenTypeId
	Amount  *big.Int
	Data    []byte
	Quota   uint64

	SnapshotHeight *types.Hash
}

type RpcMessage struct {
}
