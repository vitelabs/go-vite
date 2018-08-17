package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type NoVmBlock struct {
	height          *big.Int
	to              types.Address
	from            types.Address
	fromHash        types.Hash
	txType          int
	prevHash        types.Hash
	amount          *big.Int
	tokenId         types.TokenTypeId
	createFee       *big.Int
	data            []byte
	stateHash       types.Hash
	summaryHashList []types.Hash
	logHash         types.Hash
	snapshotHash    types.Hash
	depth           int
	quota           uint64
}

func CreateNoVmBlock(from, to types.Address, txType, depth int) VmBlock {
	return &NoVmBlock{from: from, to: to, txType: txType, depth: depth}
}

func (b *NoVmBlock) Height() *big.Int                     { return b.height }
func (b *NoVmBlock) SetHeight(height *big.Int)            { b.height = height }
func (b *NoVmBlock) To() types.Address                    { return b.to }
func (b *NoVmBlock) SetTo(to types.Address)               { b.to = to }
func (b *NoVmBlock) From() types.Address                  { return b.from }
func (b *NoVmBlock) SetFrom(from types.Address)           { b.from = from }
func (b *NoVmBlock) FromHash() types.Hash                 { return b.fromHash }
func (b *NoVmBlock) SetFromHash(fromHash types.Hash)      { b.fromHash = fromHash }
func (b *NoVmBlock) TxType() int                          { return b.txType }
func (b *NoVmBlock) SetTxType(txType int)                 { b.txType = txType }
func (b *NoVmBlock) PrevHash() types.Hash                 { return b.prevHash }
func (b *NoVmBlock) SetPrevHash(prevHash types.Hash)      { b.prevHash = prevHash }
func (b *NoVmBlock) Amount() *big.Int                     { return b.amount }
func (b *NoVmBlock) SetAmount(amount *big.Int)            { b.amount = amount }
func (b *NoVmBlock) TokenId() types.TokenTypeId           { return b.tokenId }
func (b *NoVmBlock) SetTokenId(tokenId types.TokenTypeId) { b.tokenId = tokenId }
func (b *NoVmBlock) CreateFee() *big.Int                  { return b.createFee }
func (b *NoVmBlock) SetCreateFee(createFee *big.Int)      { b.createFee = createFee }
func (b *NoVmBlock) Data() []byte                         { return b.data }
func (b *NoVmBlock) SetData(data []byte)                  { b.data = data }
func (b *NoVmBlock) StateHash() types.Hash                { return b.stateHash }
func (b *NoVmBlock) SetStateHash(stateHash types.Hash)    { b.stateHash = stateHash }
func (b *NoVmBlock) SummaryHashList() []types.Hash        { return b.summaryHashList }
func (b *NoVmBlock) AppendSummaryHash(sendTransactionHash types.Hash) {
	b.summaryHashList = append(b.summaryHashList, sendTransactionHash)
}
func (b *NoVmBlock) LogHash() types.Hash                     { return b.logHash }
func (b *NoVmBlock) SetLogHash(logHash types.Hash)           { b.logHash = logHash }
func (b *NoVmBlock) SnapshotHash() types.Hash                { return b.snapshotHash }
func (b *NoVmBlock) SetSnapshotHash(snapshotHash types.Hash) { b.snapshotHash = snapshotHash }
func (b *NoVmBlock) Depth() int                              { return b.depth }
func (b *NoVmBlock) SetDepth(depth int)                      { b.depth = depth }
func (b *NoVmBlock) Quota() uint64                           { return b.quota }
func (b *NoVmBlock) SetQuota(quota uint64)                   { b.quota = quota }
func (b *NoVmBlock) SummaryHash() types.Hash                 { return types.Hash{} }
