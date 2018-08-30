package vm

import (
	"encoding/binary"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type NoAccountBlock struct {
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

func CreateNoAccountBlock(accountAddress, toAddress types.Address, blockType, depth uint64) VmAccountBlock {
	return &NoAccountBlock{accountAddress: accountAddress, toAddress: toAddress, blockType: blockType, depth: depth}
}

func (b *NoAccountBlock) Height() *big.Int                     { return b.height }
func (b *NoAccountBlock) SetHeight(height *big.Int)            { b.height = height }
func (b *NoAccountBlock) ToAddress() types.Address             { return b.toAddress }
func (b *NoAccountBlock) SetToAddress(toAddress types.Address) { b.toAddress = toAddress }
func (b *NoAccountBlock) AccountAddress() types.Address        { return b.accountAddress }
func (b *NoAccountBlock) SetAccountAddress(accountAddress types.Address) {
	b.accountAddress = accountAddress
}
func (b *NoAccountBlock) FromBlockHash() types.Hash                 { return b.fromBlockHash }
func (b *NoAccountBlock) SetFromBlockHash(fromBlockHash types.Hash) { b.fromBlockHash = fromBlockHash }
func (b *NoAccountBlock) BlockType() uint64                         { return b.blockType }
func (b *NoAccountBlock) SetBlockType(blockType uint64)             { b.blockType = blockType }
func (b *NoAccountBlock) PrevHash() types.Hash                      { return b.prevHash }
func (b *NoAccountBlock) SetPrevHash(prevHash types.Hash)           { b.prevHash = prevHash }
func (b *NoAccountBlock) Amount() *big.Int                          { return b.amount }
func (b *NoAccountBlock) SetAmount(amount *big.Int)                 { b.amount = amount }
func (b *NoAccountBlock) TokenId() types.TokenTypeId                { return b.tokenId }
func (b *NoAccountBlock) SetTokenId(tokenId types.TokenTypeId)      { b.tokenId = tokenId }
func (b *NoAccountBlock) CreateFee() *big.Int                       { return b.createFee }
func (b *NoAccountBlock) SetCreateFee(createFee *big.Int)           { b.createFee = createFee }
func (b *NoAccountBlock) Data() []byte                              { return b.data }
func (b *NoAccountBlock) SetData(data []byte)                       { b.data = data }
func (b *NoAccountBlock) StateHash() types.Hash                     { return b.stateHash }
func (b *NoAccountBlock) SetStateHash(stateHash types.Hash)         { b.stateHash = stateHash }
func (b *NoAccountBlock) SendBlockHashList() []types.Hash           { return b.sendBlockHashList }
func (b *NoAccountBlock) AppendSendBlockHash(sendBlockHash types.Hash) {
	b.sendBlockHashList = append(b.sendBlockHashList, sendBlockHash)
}
func (b *NoAccountBlock) LogListHash() types.Hash                 { return b.logListHash }
func (b *NoAccountBlock) SetLogListHash(logListHash types.Hash)   { b.logListHash = logListHash }
func (b *NoAccountBlock) SnapshotHash() types.Hash                { return b.snapshotHash }
func (b *NoAccountBlock) SetSnapshotHash(snapshotHash types.Hash) { b.snapshotHash = snapshotHash }
func (b *NoAccountBlock) Depth() uint64                           { return b.depth }
func (b *NoAccountBlock) SetDepth(depth uint64)                   { b.depth = depth }
func (b *NoAccountBlock) Quota() uint64                           { return b.quota }
func (b *NoAccountBlock) SetQuota(quota uint64)                   { b.quota = quota }
func (b *NoAccountBlock) SummaryHash() types.Hash {
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

type NoSnapshotBlock struct {
	height    *big.Int
	timestamp int64
	hash      types.Hash
	prevHash  types.Hash
	producer  types.Address
}

func (b *NoSnapshotBlock) Height() *big.Int        { return b.height }
func (b *NoSnapshotBlock) Timestamp() int64        { return b.timestamp }
func (b *NoSnapshotBlock) Hash() types.Hash        { return b.hash }
func (b *NoSnapshotBlock) PrevHash() types.Hash    { return b.prevHash }
func (b *NoSnapshotBlock) Producer() types.Address { return b.producer }
