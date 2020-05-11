package common

import (
	"encoding/binary"
	"strconv"
	"time"
)

type Block interface {
	Height() Height
	Hash() Hash
	PrevHash() Hash
	Signer() Address
	Timestamp() time.Time
}

type HashHeight struct {
	Hash   Hash
	Height Height
}

type AccountHashH struct {
	HashHeight
	Addr Address
}

func NewAccountHashH(address Address, hash Hash, height Height) *AccountHashH {
	self := &AccountHashH{}
	self.Addr = address
	self.Hash = hash
	self.Height = height
	return self
}

type SnapshotPoint struct {
	SnapshotHeight Height
	SnapshotHash   Hash
	AccountHeight  Height
	AccountHash    Hash
}

func (point *SnapshotPoint) Equals(peer *SnapshotPoint) bool {
	if peer == nil {
		return false
	}
	if point.SnapshotHash == peer.SnapshotHash &&
		point.SnapshotHeight == peer.SnapshotHeight &&
		point.AccountHeight == peer.AccountHeight &&
		point.AccountHash == peer.AccountHash {
		return true
	}
	return false
}

func (point *SnapshotPoint) String() string {
	return "[" + point.SnapshotHeight.String() + "][" + point.SnapshotHash.String() + "][" + point.AccountHeight.String() + "][" + point.AccountHash.String() + "]"
}

//BlockType is the type of Tx described by int.
type BlockType uint8

// BlockType types.
const (
	GENESIS BlockType = iota
	SEND
	RECEIVED
)

var txStr map[BlockType]string
var netMsgTypeStr map[NetMsgType]string

func init() {
	txStr = map[BlockType]string{
		SEND:     "send tx",
		RECEIVED: "received tx",
	}

	netMsgTypeStr = map[NetMsgType]string{
		State:                 "State",
		RequestAccountHash:    "RequestAccountHash",
		RequestSnapshotHash:   "RequestSnapshotHash",
		RequestAccountBlocks:  "RequestAccountBlocks",
		RequestSnapshotBlocks: "RequestSnapshotBlocks",
		AccountHashes:         "AccountHashes",
		SnapshotHashes:        "SnapshotHashes",
		AccountBlocks:         "AccountBlocks",
		SnapshotBlocks:        "SnapshotBlocks",
	}
}

func (tp BlockType) String() string {
	if s, ok := txStr[tp]; ok {
		return s
	}
	return "Unknown"
}

func (tp NetMsgType) String() string {
	if s, ok := netMsgTypeStr[tp]; ok {
		return s
	}
	return "Unknown"
}

type Tblock struct {
	Theight    Height
	Thash      Hash
	TprevHash  Hash
	Tsigner    Address
	Ttimestamp time.Time
}

func (self *Tblock) Height() Height {
	return self.Theight
}

func (self *Tblock) Hash() Hash {
	return self.Thash
}

func (self *Tblock) PrevHash() Hash {
	return self.TprevHash
}

func (self *Tblock) Signer() Address {
	return self.Tsigner
}
func (self *Tblock) Timestamp() time.Time {
	return self.Ttimestamp
}
func (self *Tblock) SetHash(hash Hash) {
	self.Thash = hash
}

type AccountStateBlock struct {
	Tblock
	Amount         Balance
	ModifiedAmount Balance
	BlockType      BlockType // 1: send  2:received
	From           Address
	To             Address
	Source         *HashHeight
}

type SnapshotBlock struct {
	Tblock
	Accounts []*AccountHashH
}

func NewSnapshotBlock(
	height Height,
	hash Hash,
	prevHash Hash,
	signer Address,
	timestamp time.Time,
	accounts []*AccountHashH,
) *SnapshotBlock {

	block := &SnapshotBlock{}
	block.Theight = height
	block.Thash = hash
	block.TprevHash = prevHash
	block.Tsigner = signer
	block.Ttimestamp = timestamp
	block.Accounts = accounts
	return block
}

type Address string

func HexToAddress(hexStr string) Address {
	return Address(hexStr)
}

func (addr Address) String() string {
	return string(addr)
}

func (addr Address) Bytes() []byte {
	return []byte(addr.String())
}
func NewAccountBlockEmpty() *AccountStateBlock {
	block := &AccountStateBlock{}
	block.Theight = EmptyHeight
	block.Thash = EmptyHash
	block.TprevHash = EmptyHash
	return block
}

func NewAccountBlock(
	height Height,
	hash Hash,
	prevHash Hash,
	signer Address,
	timestamp time.Time,

	amount Balance,
	modifiedAmount Balance,
	blockType BlockType,
	from Address,
	to Address,
	source *HashHeight,
) *AccountStateBlock {

	block := &AccountStateBlock{}
	block.Theight = height
	block.Thash = hash
	block.TprevHash = prevHash
	block.Tsigner = signer
	block.Ttimestamp = timestamp
	block.Amount = amount
	block.ModifiedAmount = modifiedAmount
	block.BlockType = blockType
	block.From = from
	block.To = to
	block.Source = source
	return block
}

func NewAccountBlockFrom(
	accountBlock *AccountStateBlock,
	signer Address,
	timestamp time.Time,

	modifiedAmount Balance,

	blockType BlockType,
	from *Address,
	to *Address,
	source *HashHeight,
) *AccountStateBlock {
	block := &AccountStateBlock{}
	if accountBlock == nil {
		block.Theight = 1
		block.TprevHash = ""
		block.Amount = modifiedAmount
	} else {
		block.Theight = accountBlock.Height() + 1
		block.TprevHash = accountBlock.Hash()
		block.Amount = accountBlock.Amount + modifiedAmount
	}
	block.Tsigner = signer
	block.Ttimestamp = timestamp
	block.ModifiedAmount = modifiedAmount
	block.BlockType = blockType
	block.From = from
	block.To = to
	block.Source = source
	return block
}

type NetMsgType uint8

const (
	PeerConnected         NetMsgType = 1
	PeerClosed            NetMsgType = 2
	State                 NetMsgType = 101
	RequestAccountHash    NetMsgType = 102
	RequestSnapshotHash   NetMsgType = 103
	RequestAccountBlocks  NetMsgType = 104
	RequestSnapshotBlocks NetMsgType = 105
	AccountHashes         NetMsgType = 121
	SnapshotHashes        NetMsgType = 122
	AccountBlocks         NetMsgType = 123
	SnapshotBlocks        NetMsgType = 124
)

var FirstHeight = Height(1)
var EmptyHeight = Height(0)
var EmptyBalance = Balance(0)
var EmptyHash = Hash("")
var EmptyHashHeight = HashHeight{
	Hash:   EmptyHash,
	Height: EmptyHeight,
}

type StoreDBType uint8
type Hash string
type Height uint64
type Sign string
type Balance uint64

func (hash Hash) String() string {
	return string(hash)
}

func (hash Hash) Bytes() []byte {
	return []byte(hash.String())
}

func BytesToHeight(value []byte) Height {
	height := binary.BigEndian.Uint64(value)
	return Height(height)
}

func (height Height) String() string {
	return strconv.FormatUint(uint64(height), 10)
}

func (height Height) Bytes() []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(height))
	return buf[:]
}

func (height Height) Uint64() uint64 {
	return uint64(height)
}

func (balance Balance) String() string {
	return strconv.FormatUint(uint64(balance), 10)
}

const (
	Memory  StoreDBType = 1
	LevelDB StoreDBType = 2
)
