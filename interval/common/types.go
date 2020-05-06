package common

import (
	"strconv"
	"time"
)

type Block interface {
	Height() uint64
	Hash() string
	PreHash() string
	Signer() string
	Timestamp() time.Time
}

type HashHeight struct {
	Hash   string
	Height uint64
}

type AccountHashH struct {
	HashHeight
	Addr string
}

func NewAccountHashH(address string, hash string, height uint64) *AccountHashH {
	self := &AccountHashH{}
	self.Addr = address
	self.Hash = hash
	self.Height = height
	return self
}

type SnapshotPoint struct {
	SnapshotHeight uint64
	SnapshotHash   string
	AccountHeight  uint64
	AccountHash    string
}

func (self *SnapshotPoint) Equals(peer *SnapshotPoint) bool {
	if peer == nil {
		return false
	}
	if self.SnapshotHash == peer.SnapshotHash &&
		self.SnapshotHeight == peer.SnapshotHeight &&
		self.AccountHeight == peer.AccountHeight &&
		self.AccountHash == peer.AccountHash {
		return true
	}
	return false
}

func (self *SnapshotPoint) String() string {
	return "[" + strconv.FormatUint(self.SnapshotHeight, 10) + "][" + self.SnapshotHash + "][" + strconv.FormatUint(self.AccountHeight, 10) + "][" + self.AccountHash + "]"
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

func (self BlockType) String() string {
	if s, ok := txStr[self]; ok {
		return s
	}
	return "Unknown"
}

func (self NetMsgType) String() string {
	if s, ok := netMsgTypeStr[self]; ok {
		return s
	}
	return "Unknown"
}

type Tblock struct {
	Theight    uint64
	Thash      string
	TpreHash   string
	Tsigner    string
	Ttimestamp time.Time
}

func (self *Tblock) Height() uint64 {
	return self.Theight
}

func (self *Tblock) Hash() string {
	return self.Thash
}

func (self *Tblock) PreHash() string {
	return self.TpreHash
}

func (self *Tblock) Signer() string {
	return self.Tsigner
}
func (self *Tblock) Timestamp() time.Time {
	return self.Ttimestamp
}
func (self *Tblock) SetHash(hash string) {
	self.Thash = hash
}

type AccountStateBlock struct {
	Tblock
	Amount         int // the balance
	ModifiedAmount int
	SnapshotHeight uint64
	SnapshotHash   string
	BlockType      BlockType // 1: send  2:received
	From           string
	To             string
	Source         *HashHeight
}

type SnapshotBlock struct {
	Tblock
	Accounts []*AccountHashH
}

func NewSnapshotBlock(
	height uint64,
	hash string,
	preHash string,
	signer string,
	timestamp time.Time,
	accounts []*AccountHashH,
) *SnapshotBlock {

	block := &SnapshotBlock{}
	block.Theight = height
	block.Thash = hash
	block.TpreHash = preHash
	block.Tsigner = signer
	block.Ttimestamp = timestamp
	block.Accounts = accounts
	return block
}

type Address []byte

func HexToAddress(hexStr string) Address {
	return []byte(hexStr)
}

func (self Address) String() string {
	return string((self)[:])
}

func NewAccountBlock(
	height uint64,
	hash string,
	preHash string,
	signer string,
	timestamp time.Time,

	amount int,
	modifiedAmount int,
	snapshotHeight uint64,
	snapshotHash string,
	blockType BlockType,
	from string,
	to string,
	source *HashHeight,
) *AccountStateBlock {

	block := &AccountStateBlock{}
	block.Theight = height
	block.Thash = hash
	block.TpreHash = preHash
	block.Tsigner = signer
	block.Ttimestamp = timestamp
	block.Amount = amount
	block.ModifiedAmount = modifiedAmount
	block.SnapshotHash = snapshotHash
	block.SnapshotHeight = snapshotHeight
	block.BlockType = blockType
	block.From = from
	block.To = to
	block.Source = source
	return block
}

func NewAccountBlockFrom(
	accountBlock *AccountStateBlock,
	signer string,
	timestamp time.Time,

	modifiedAmount int,
	snapshotBlock *SnapshotBlock,

	blockType BlockType,
	from string,
	to string,
	source *HashHeight,
) *AccountStateBlock {
	block := &AccountStateBlock{}
	if accountBlock == nil {
		block.Theight = 1
		block.TpreHash = ""
		block.Amount = modifiedAmount
	} else {
		block.Theight = accountBlock.Height() + 1
		block.TpreHash = accountBlock.Hash()
		block.Amount = accountBlock.Amount + modifiedAmount
	}
	block.Tsigner = signer
	block.Ttimestamp = timestamp
	block.ModifiedAmount = modifiedAmount
	block.SnapshotHash = snapshotBlock.Hash()
	block.SnapshotHeight = snapshotBlock.Height()
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

var FirstHeight = uint64(1)
var EmptyHeight = uint64(0)

type StoreDBType uint8

const (
	Memory  StoreDBType = 1
	LevelDB StoreDBType = 2
)
