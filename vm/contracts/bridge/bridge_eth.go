package bridge

import (
	"encoding/json"
	"math/big"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type ethBridge struct {
	// the latest confirmed header
	latest *wrappedEthHeader
	// all confirmed header, and these can be used as proof
	history map[uint64]*wrappedEthHeader

	// all pending headers, and headers with height less than the latest height will be deleted
	unconfirmed map[uint64][]*wrappedEthHeader

	// any header less than this difficulty will not be submitted successfully
	leastDifficulty *big.Int

	// any header less than this height will not be submitted successfully
	leastHeight uint64

	// confirmed num
	confirmedThreshold uint64

	v ethVerifier
}

// the wrapped header contains some additional data
type wrappedEthHeader struct {
	TotalDifficulty *big.Int         `json:"totalDifficulty"`
	Header          *ethtypes.Header `json:"header"`
}

// EthBridgeInitArgs arguments when calling the Init method
type EthBridgeInitArgs struct {
	TotalDifficulty    *big.Int `json:"totalDifficulty"`
	Header             []byte   `json:"header"`
	LeastDifficulty    *big.Int `json:"leastDifficulty"`
	ConfirmedThreshold uint64   `json:"confirmedThreshold"`
}

// This method can only be called once
func (b *ethBridge) Init(content interface{}) (err error) {
	if b.latest != nil {
		return InitErr
	}
	args := content.(*EthBridgeInitArgs)
	header := &ethtypes.Header{}

	err = b.unmarshalHeader(header, args.Header)
	if err != nil {
		return
	}

	b.leastDifficulty = args.LeastDifficulty
	b.leastHeight = header.Number.Uint64()
	b.latest = &wrappedEthHeader{
		TotalDifficulty: args.TotalDifficulty,
		Header:          header,
	}
	return nil
}

// EthBridgeSubmitArgs arguments when calling the Submit method
type EthBridgeSubmitArgs struct {
	header []byte `json:"header"`
}

func (b *ethBridge) Submit(content interface{}) (err error) {
	if b.latest == nil {
		return SubmitErr
	}
	// parse and verify header
	args := content.(*EthBridgeSubmitArgs)
	header := &ethtypes.Header{}

	err = b.unmarshalHeader(header, args.header)
	if err != nil {
		return
	}
	return b.submit(header)
}

func (b ethBridge) unmarshalHeader(header *ethtypes.Header, content []byte) error {
	err := json.Unmarshal(content, header)
	if err != nil {
		return err
	}
	if !b.v.verifyHeader(header) {
		return HeaderErr
	}
	return nil
}

func (b ethBridge) wrapHeader(header *ethtypes.Header) (*wrappedEthHeader, error) {
	result := &wrappedEthHeader{
		Header:          header,
		TotalDifficulty: big.NewInt(0),
	}
	height := header.Number.Uint64()

	parent := b.findHeader(height-1, header.ParentHash)
	if parent == nil {
		return nil, ParentHeaderErr
	}
	result.TotalDifficulty.Add(parent.TotalDifficulty, header.Difficulty)
	return result, nil
}

func (b ethBridge) findHeader(height uint64, hash ethcommon.Hash) *wrappedEthHeader {
	latestHeight := b.latest.Header.Number.Uint64()
	if height <= latestHeight {
		item, ok := b.history[height]
		if !ok {
			return nil
		}
		if hash.String() == item.Header.Hash().String() {
			return item
		}
		return nil
	} else {
		unconfirmedList, ok := b.unconfirmed[height]
		if !ok {
			return nil
		}
		for _, item := range unconfirmedList {
			if hash.String() == item.Header.Hash().String() {
				return item
			}
		}
		return nil
	}
}

func (b *ethBridge) submit(header *ethtypes.Header) error {
	got := b.findHeader(header.Number.Uint64(), header.Hash())
	if got != nil {
		return HeaderDuplicatedErr
	}

	wrapHeader, err := b.wrapHeader(header)
	if err != nil {
		return err
	}

	height := wrapHeader.Header.Number.Uint64()
	currentHeight := b.latest.Header.Number.Uint64()
	if height-currentHeight < b.confirmedThreshold {
		b.unconfirmed[height] = append(b.unconfirmed[height], wrapHeader)
		return nil
	}

	nextLatestBlock := b.findNextLatestBlock(currentHeight, wrapHeader)
	if nextLatestBlock == nil {
		return NextLatestBlockErr
	}
	b.updateLatestBlock(nextLatestBlock)
	return nil
}

func (b ethBridge) findNextLatestBlock(currentHeight uint64, trust *wrappedEthHeader) *wrappedEthHeader {
	for {
		height := trust.Header.Number.Uint64() - 1
		if height == currentHeight {
			return trust
		}
		nextLatestHash := trust.Header.ParentHash
		trust = b.findHeader(height, nextLatestHash)
		if trust == nil {
			return nil
		}
	}
}

func (b *ethBridge) updateLatestBlock(block *wrappedEthHeader) {
	height := block.Header.Number.Uint64()
	b.history[height] = block
	b.unconfirmed[height] = []*wrappedEthHeader{}
}

type EthBridgeProofArgs struct {
	BlockHeight uint64         `json:"blockHeight"`
	BlockHash   ethcommon.Hash `json:"blockHash"`
	ReceiptRoot ethcommon.Hash `json:"receiptHash"`
	Proofs      ethcommon.Hash `json:"proofs"`

	Receipt      ethcommon.Hash `json:"receipt"`
	ReceiptIndex int            `json:"receiptIndex"`
}

func (b ethBridge) Proof(content interface{}) (bool, error) {
	args := content.(*EthBridgeProofArgs)
	block := b.findHeader(args.BlockHeight, args.BlockHash)
	if block == nil {
		return false, nil
	}
	if block.Header.ReceiptHash.String() != args.ReceiptRoot.String() {
		return false, ProofReceiptErr
	}

	proof := b.v.verifyProof(args.ReceiptRoot, args.Receipt, args.ReceiptIndex)
	return proof, nil
}
