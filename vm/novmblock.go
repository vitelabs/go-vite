package vm

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type NoVmAccountBlock struct {
	height            *big.Int
	toAddress         types.Address
	accountAddress    types.Address
	fromBlockHash     types.Hash
	blockType         uint64
	prevHash          types.Hash
	amount            *big.Int
	tokenId           types.TokenTypeId
	createFee         *big.Int
	data              []byte
	stateHash         types.Hash
	sendBlockHashList []types.Hash
	logListHash       types.Hash
	snapshotHash      types.Hash
	depth             uint64
	quota             uint64
}

func CreateNoVmAccountBlock(accountAddress, toAddress types.Address, blockType, depth uint64) VmAccountBlock {
	return &NoVmAccountBlock{accountAddress: accountAddress, toAddress: toAddress, blockType: blockType, depth: depth}
}

func (b *NoVmAccountBlock) Height() *big.Int                     { return b.height }
func (b *NoVmAccountBlock) SetHeight(height *big.Int)            { b.height = height }
func (b *NoVmAccountBlock) ToAddress() types.Address             { return b.toAddress }
func (b *NoVmAccountBlock) SetToAddress(toAddress types.Address) { b.toAddress = toAddress }
func (b *NoVmAccountBlock) AccountAddress() types.Address        { return b.accountAddress }
func (b *NoVmAccountBlock) SetAccountAddress(accountAddress types.Address) {
	b.accountAddress = accountAddress
}
func (b *NoVmAccountBlock) FromBlockHash() types.Hash                 { return b.fromBlockHash }
func (b *NoVmAccountBlock) SetFromBlockHash(fromBlockHash types.Hash) { b.fromBlockHash = fromBlockHash }
func (b *NoVmAccountBlock) BlockType() uint64                         { return b.blockType }
func (b *NoVmAccountBlock) SetBlockType(blockType uint64)             { b.blockType = blockType }
func (b *NoVmAccountBlock) PrevHash() types.Hash                      { return b.prevHash }
func (b *NoVmAccountBlock) SetPrevHash(prevHash types.Hash)           { b.prevHash = prevHash }
func (b *NoVmAccountBlock) Amount() *big.Int                          { return b.amount }
func (b *NoVmAccountBlock) SetAmount(amount *big.Int)                 { b.amount = amount }
func (b *NoVmAccountBlock) TokenId() types.TokenTypeId                { return b.tokenId }
func (b *NoVmAccountBlock) SetTokenId(tokenId types.TokenTypeId)      { b.tokenId = tokenId }
func (b *NoVmAccountBlock) CreateFee() *big.Int                       { return b.createFee }
func (b *NoVmAccountBlock) SetCreateFee(createFee *big.Int)           { b.createFee = createFee }
func (b *NoVmAccountBlock) Data() []byte                              { return b.data }
func (b *NoVmAccountBlock) SetData(data []byte)                       { b.data = data }
func (b *NoVmAccountBlock) StateHash() types.Hash                     { return b.stateHash }
func (b *NoVmAccountBlock) SetStateHash(stateHash types.Hash)         { b.stateHash = stateHash }
func (b *NoVmAccountBlock) SendBlockHashList() []types.Hash           { return b.sendBlockHashList }
func (b *NoVmAccountBlock) AppendSendBlockHash(sendBlockHash types.Hash) {
	b.sendBlockHashList = append(b.sendBlockHashList, sendBlockHash)
}
func (b *NoVmAccountBlock) LogListHash() types.Hash                 { return b.logListHash }
func (b *NoVmAccountBlock) SetLogListHash(logListHash types.Hash)   { b.logListHash = logListHash }
func (b *NoVmAccountBlock) SnapshotHash() types.Hash                { return b.snapshotHash }
func (b *NoVmAccountBlock) SetSnapshotHash(snapshotHash types.Hash) { b.snapshotHash = snapshotHash }
func (b *NoVmAccountBlock) Depth() uint64                           { return b.depth }
func (b *NoVmAccountBlock) SetDepth(depth uint64)                   { b.depth = depth }
func (b *NoVmAccountBlock) Quota() uint64                           { return b.quota }
func (b *NoVmAccountBlock) SetQuota(quota uint64)                   { b.quota = quota }
func (b *NoVmAccountBlock) SummaryHash() types.Hash {
	// hash of from + to + data + amount + tokenId + depth + height + txType
	var buf = make([]byte, 8)
	summaryData := append(b.accountAddress.Bytes(), b.toAddress.Bytes()...)
	summaryData = append(summaryData, b.data...)
	summaryData = append(summaryData, b.amount.Bytes()...)
	summaryData = append(summaryData, b.tokenId.Bytes()...)
	binary.BigEndian.PutUint64(buf, b.depth)
	summaryData = append(summaryData, buf...)
	summaryData = append(summaryData, b.height.Bytes()...)
	binary.BigEndian.PutUint64(buf, b.blockType)
	summaryData = append(summaryData, buf...)
	return types.DataHash(summaryData)
}
