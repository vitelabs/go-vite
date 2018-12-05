package client

import (
	"encoding/binary"
	"math/big"
	"strconv"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/rpc"
	"golang.org/x/crypto/ed25519"
)

type RawBlock struct {
	BlockType byte       `json:"blockType"`
	Hash      types.Hash `json:"hash"`
	PrevHash  types.Hash `json:"prevHash"`

	AccountAddress types.Address `json:"accountAddress"`

	PublicKey     ed25519.PublicKey `json:"publicKey"`
	ToAddress     types.Address     `json:"toAddress"`
	FromBlockHash types.Hash        `json:"fromBlockHash"`

	TokenId types.TokenTypeId `json:"tokenId"`

	SnapshotHash types.Hash `json:"snapshotHash"`
	Data         []byte     `json:"data"`

	Nonce     []byte `json:"nonce"`
	Signature []byte `json:"signature"`

	FromAddress types.Address `json:"fromAddress"`

	Height string `json:"height"`

	Amount     *string `json:"amount"`
	Fee        *string `json:"fee"`
	Difficulty *string `json:"difficulty"`

	Timestamp int64 `json:"timestamp"`
}

func (ab RawBlock) ComputeHash() (*types.Hash, error) {
	var source []byte
	// BlockType
	source = append(source, ab.BlockType)

	// PrevHash
	source = append(source, ab.PrevHash.Bytes()...)

	// Height
	heightBytes := make([]byte, 8)
	u, err := strconv.ParseUint(ab.Height, 10, 64)
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint64(heightBytes, u)
	source = append(source, heightBytes...)

	// AccountAddress
	source = append(source, ab.AccountAddress.Bytes()...)

	if ab.IsSendBlock() {
		// ToAddress
		source = append(source, ab.ToAddress.Bytes()...)
		amount := big.NewInt(0)
		if ab.Amount != nil {
			amount.SetString(*ab.Amount, 10)
		}
		// Amount
		source = append(source, amount.Bytes()...)
		// TokenId
		source = append(source, ab.TokenId.Bytes()...)
	} else {
		// FromBlockHash
		source = append(source, ab.FromBlockHash.Bytes()...)
	}

	fee := big.NewInt(0)
	if ab.Fee != nil {
		fee.SetString(*ab.Fee, 10)
	}
	source = append(source, fee.Bytes()...)

	// SnapshotHash
	source = append(source, ab.SnapshotHash.Bytes()...)

	// Data
	source = append(source, ab.Data...)

	// Timestamp
	unixTimeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(unixTimeBytes, uint64(ab.Timestamp))
	source = append(source, unixTimeBytes...)

	// Nonce
	source = append(source, ab.Nonce...)

	hash, _ := types.BytesToHash(crypto.Hash256(source))
	return &hash, nil
}
func (ab *RawBlock) IsSendBlock() bool {
	return ab.BlockType == ledger.BlockTypeSendCreate || ab.BlockType == ledger.BlockTypeSendCall || ab.BlockType == ledger.BlockTypeSendReward
}

type BlockHeader struct {
	Hash     types.Hash `json:"hash"`
	PrevHash types.Hash `json:"prevHash"`

	Height string `json:"height"`

	Timestamp int64 `json:"timestamp"`
}

type RpcClient interface {
	SubmitRaw(block RawBlock) error
	GetLatest(address types.Address) (*BlockHeader, error)
	getFittestSnapshot() (*types.Hash, error)
}

func NewRpcClient(rawurl string) (RpcClient, error) {
	c, err := rpc.Dial(rawurl)
	if err != nil {
		return nil, err
	}

	r := &rpcClient{cc: c}
	return r, nil
}

type rpcClient struct {
	cc *rpc.Client
}

func (c *rpcClient) getFittestSnapshot() (*types.Hash, error) {
	hash := types.Hash{}
	c.cc.Call(&hash, "ledger_getFittestSnapshotHash", nil)
	return &hash, nil
}

func (c *rpcClient) GetLatest(address types.Address) (*BlockHeader, error) {
	header := BlockHeader{}
	err := c.cc.Call(&header, "ledger_getLatestBlock", address)
	if err != nil {
		return nil, err
	}
	return &header, nil
}

func (c *rpcClient) SubmitRaw(block RawBlock) error {
	err := c.cc.Call(nil, "tx_sendRawTx", &block)
	return err
}
