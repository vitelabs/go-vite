package net

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

// GetAccountBlocks
func mockGetAccountBlocks() GetAccountBlocks {
	var ga GetAccountBlocks

	ga.From.Height = mrand.Uint64()
	_, _ = crand.Read(ga.From.Hash[:])

	ga.Count = mrand.Uint64()
	ga.Forward = mrand.Intn(10) > 5

	_, _ = crand.Read(ga.Address[:])

	return ga
}

func equalGetAccountBlocks(g, g2 GetAccountBlocks) bool {
	if g.From.Height != g2.From.Height {
		return false
	}

	if g.From.Hash != g2.From.Hash {
		return false
	}

	if g.Count != g2.Count {
		return false
	}

	if g.Forward != g2.Forward {
		return false
	}

	if g.Address != g2.Address {
		return false
	}

	return true
}

func TestGetAccountBlocks_Serialize(t *testing.T) {
	ga := mockGetAccountBlocks()

	buf, err := ga.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g GetAccountBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalGetAccountBlocks(ga, g) {
		t.Fail()
	}
}

func mockGetSnapshotBlocks() GetSnapshotBlocks {
	var ga GetSnapshotBlocks

	ga.From.Height = mrand.Uint64()
	_, _ = crand.Read(ga.From.Hash[:])

	ga.Count = mrand.Uint64()
	ga.Forward = mrand.Intn(10) > 5

	return ga
}

func equalGetSnapshotBlocks(g, g2 GetSnapshotBlocks) bool {
	if g.From.Height != g2.From.Height {
		return false
	}

	if g.From.Hash != g2.From.Hash {
		return false
	}

	if g.Count != g2.Count {
		return false
	}

	if g.Forward != g2.Forward {
		return false
	}

	return true
}

func TestGetSnapshotBlocks_Deserialize(t *testing.T) {
	gs := mockGetSnapshotBlocks()

	buf, err := gs.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g GetSnapshotBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalGetSnapshotBlocks(gs, g) {
		t.Fail()
	}
}

// AccountBlocks
func mockAccountBlocks() AccountBlocks {
	var ga AccountBlocks

	n := mrand.Intn(100)
	ga.Blocks = make([]*ledger.AccountBlock, n)

	for i := 0; i < n; i++ {
		ga.Blocks[i] = &ledger.AccountBlock{
			BlockType:      0,
			Hash:           types.Hash{},
			Height:         0,
			PrevHash:       types.Hash{},
			AccountAddress: types.Address{},
			PublicKey:      nil,
			ToAddress:      types.Address{},
			FromBlockHash:  types.Hash{},
			Amount:         new(big.Int),
			TokenId:        types.TokenTypeId{},
			Quota:          0,
			Fee:            new(big.Int),
			Data:           nil,
			LogHash:        nil,
			Difficulty:     nil,
			Nonce:          nil,
			Signature:      nil,
		}
	}

	return ga
}

func equalAccountBlocks(g, g2 AccountBlocks) bool {
	if len(g.Blocks) != len(g2.Blocks) {
		return false
	}

	return true
}

func TestAccountBlocks_Deserialize(t *testing.T) {
	gs := mockAccountBlocks()

	buf, err := gs.Serialize()
	if err != nil {
		t.Error(err)
	}

	var g AccountBlocks
	err = g.Deserialize(buf)
	if err != nil {
		t.Error(err)
	}

	if !equalAccountBlocks(gs, g) {
		t.Fail()
	}
}

func TestNewSnapshotBlock_Serialize(t *testing.T) {
	var nb = &NewSnapshotBlock{}

	now := time.Now()
	nb.Block = &ledger.SnapshotBlock{
		Hash:            types.Hash{},
		PrevHash:        types.Hash{},
		Height:          0,
		PublicKey:       []byte("hello"),
		Signature:       []byte("hello"),
		Timestamp:       &now,
		Seed:            0,
		SeedHash:        &types.Hash{},
		SnapshotContent: nil,
	}

	data, err := nb.Serialize()
	if err != nil {
		t.Error("should not error")
	}

	var nb2 = &NewSnapshotBlock{}
	err = nb2.Deserialize(data)
	if err != nil {
		t.Errorf("deserialize error: %v", err)
	}

	if nb.Block.ComputeHash() != nb2.Block.ComputeHash() {
		t.Error("different hash")
	}
}

func TestNewAccountBlock_Serialize(t *testing.T) {
	var nb = &NewAccountBlock{}

	nb.Block = &ledger.AccountBlock{
		BlockType:      0,
		Hash:           types.Hash{1, 1, 1},
		Height:         0,
		PrevHash:       types.Hash{},
		AccountAddress: types.Address{},
		PublicKey:      nil,
		ToAddress:      types.Address{},
		FromBlockHash:  types.Hash{},
		Amount:         new(big.Int),
		TokenId:        types.TokenTypeId{},
		Quota:          0,
		Fee:            new(big.Int),
		Data:           nil,
		LogHash:        nil,
		Difficulty:     nil,
		Nonce:          nil,
		Signature:      nil,
	}

	data, err := nb.Serialize()
	if err != nil {
		t.Error("should not error")
	}

	var nb2 = &NewAccountBlock{}
	err = nb2.Deserialize(data)
	if err != nil {
		t.Errorf("deserialize error: %v", err)
	}

	if nb.Block.ComputeHash() != nb2.Block.ComputeHash() {
		t.Error("different hash")
	}

	if nb.Block.Hash != nb2.Block.Hash {
		t.Error("different hash")
	}
}

//func ExampleNewSnapshotBlock() {
//	var nb = &NewSnapshotBlock{}
//
//	now := time.Now()
//	nb.Block = &ledger.SnapshotBlock{
//		Hash:            types.Hash{},
//		PrevHash:        types.Hash{},
//		Height:          0,
//		PublicKey:       []byte("hello"),
//		Signature:       []byte("hello"),
//		Timestamp:       &now,
//		Seed:            0,
//		SeedHash:        &types.Hash{},
//		SnapshotContent: nil,
//	}
//
//	data, err := nb.Serialize()
//	if err != nil {
//		panic(err)
//	}
//
//	var ab = &NewSnapshotBlock{}
//	err = ab.Deserialize(data)
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println(false)
//
//	// Output:
//	// false
//}
//
//func ExampleNewSnapBlock() {
//	ab := &ledger.AccountBlock{
//		BlockType:      0,
//		Hash:           types.Hash{1, 1, 1},
//		Height:         0,
//		PrevHash:       types.Hash{},
//		AccountAddress: types.Address{},
//		PublicKey:      nil,
//		ToAddress:      types.Address{},
//		FromBlockHash:  types.Hash{},
//		Amount:         new(big.Int),
//		TokenId:        types.TokenTypeId{},
//		Quota:          0,
//		Fee:            new(big.Int),
//		Data:           nil,
//		LogHash:        nil,
//		Difficulty:     nil,
//		Nonce:          nil,
//		Signature:      nil,
//	}
//
//	data, err := ab.Serialize()
//	if err != nil {
//		panic(err)
//	}
//
//	var sb = new(ledger.AccountBlock)
//	err = sb.Deserialize(data)
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println(false)
//	// Output:
//	// false
//}

func TestComputeBlockSize(t *testing.T) {
	printSendBlock()
	printReceiveBlock()
}
func printSendBlock() {
	sendCallBlock := &ledger.AccountBlock{
		BlockType:     ledger.BlockTypeSendCall,
		Height:        757,
		FromBlockHash: types.ZERO_HASH,
		Data:          []byte{},
		Quota:         21000,
		QuotaUsed:     21000,
		Fee:           big.NewInt(0),
		Difficulty:    big.NewInt(75164738),
	}
	sendCallBlock.Hash, _ = types.HexToHash("8f85502f81fc544cb6700ad9ecc44f3eace065ae8e34d2269d7ff8d7c94ac920")
	sendCallBlock.PrevHash, _ = types.HexToHash("1f1d29452644496dd15117d85d4e4fb395dd5376293eb62e94b81d35f0f19a95")
	sendCallBlock.AccountAddress, _ = types.HexToAddress("vite_dfac99c41e98784d6e92b5de4428a4106103657b13c1433184")
	sendCallBlock.PublicKey, _ = hex.DecodeString("0e8a21f65c0ac55702512b426b9057ca4a1d39e260f37daae0c956a5d237a0b3")
	sendCallBlock.ToAddress, _ = types.HexToAddress("vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")
	sendCallBlock.Amount, _ = new(big.Int).SetString("10000000000000000000", 10)
	sendCallBlock.TokenId, _ = types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	sendCallBlock.Nonce, _ = hex.DecodeString("6d2f638c1941473f")
	sendCallBlock.Signature, _ = hex.DecodeString("3a95df1a01fc006261ac653910fea6523e9e452dddfaeeddf521a1cb083985bfdf1c314132d1f8aa608938edaec7b0dbb9f362dbfcf6d16871f96d46e61a840a")
	fmt.Println("send call block")
	bs, _ := sendCallBlock.Serialize()
	fmt.Println(len(bs)) // 269 bytes
	netB := &NewAccountBlock{Block: sendCallBlock, TTL: 32}
	netBs, _ := netB.Serialize()
	fmt.Println(len(netBs)) // 274 bytes
}
func printReceiveBlock() {
	receiveCallBlock := &ledger.AccountBlock{
		BlockType: ledger.BlockTypeReceive,
		Height:    1000215,
		Quota:     37589,
		QuotaUsed: 37589,
		Fee:       big.NewInt(0),
	}
	receiveCallBlock.FromBlockHash, _ = types.HexToHash("12fb277325de59489063b40839864aa12df362a6946af6d7da0fcbcd4c8f0c7e")
	receiveCallBlock.Data, _ = hex.DecodeString("000000000000000000000000000000000000000000000000000000000000000000")
	receiveCallBlock.Hash, _ = types.HexToHash("8f85502f81fc544cb6700ad9ecc44f3eace065ae8e34d2269d7ff8d7c94ac920")
	receiveCallBlock.PrevHash, _ = types.HexToHash("75291239501f1671c92f6248a79906c60b18b84ae983eea462e20f23b73013dc")
	receiveCallBlock.AccountAddress, _ = types.HexToAddress("vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")
	receiveCallBlock.PublicKey, _ = hex.DecodeString("d5c8311234f52f7c7e98fccab019d5e0348f894f914e663b6a46eb4c2c5c02b1")
	receiveCallBlock.Signature, _ = hex.DecodeString("271af4398f9bfeabb1fc8738e182b50e4bb3a3781aba8ac869dc628a7a39f5c30c545adaff8f54741000ebf9d28b7a5cc0549c0c5f7d8949d66c3b1b6abc8305")
	logHash, _ := types.HexToHash("252d00b215a1e294dff8a61dc8cc926cedb99a18772de764258858c31c101a29")
	receiveCallBlock.LogHash = &logHash
	fmt.Println("receive call block")
	bs, _ := receiveCallBlock.Serialize()
	fmt.Println(len(bs)) // 310 bytes
	netB := &NewAccountBlock{Block: receiveCallBlock, TTL: 32}
	netBs, _ := netB.Serialize()
	fmt.Println(len(netBs)) // 315 bytes

	sendBlock := &ledger.AccountBlock{
		BlockType:     ledger.BlockTypeSendCall,
		FromBlockHash: types.ZERO_HASH,
		Data:          []byte{},
		Fee:           big.NewInt(0),
	}
	sendBlock.Hash, _ = types.HexToHash("8f85502f81fc544cb6700ad9ecc44f3eace065ae8e34d2269d7ff8d7c94ac920")
	sendBlock.AccountAddress, _ = types.HexToAddress("vite_dfac99c41e98784d6e92b5de4428a4106103657b13c1433184")
	sendBlock.ToAddress, _ = types.HexToAddress("vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5")
	sendBlock.Amount, _ = new(big.Int).SetString("10000000000000000000", 10)
	sendBlock.TokenId, _ = types.HexToTokenTypeId("tti_5649544520544f4b454e6e40")
	fmt.Println("receive and send call block")
	receiveCallBlock.SendBlockList = []*ledger.AccountBlock{sendBlock}
	rsbs, _ := receiveCallBlock.Serialize()
	fmt.Println(len(rsbs)) // 417 bytes
	netRb := &NewAccountBlock{Block: receiveCallBlock, TTL: 32}
	netRbs, _ := netRb.Serialize()
	fmt.Println(len(netRbs)) // 422 bytes
}

func compareHashHeightList(c1, c2 *HashHeightPointList) error {
	if len(c1.Points) != len(c2.Points) {
		return fmt.Errorf("different points length")
	}

	for i, p1 := range c1.Points {
		p2 := c2.Points[i]
		if p1.Height != p2.Height {
			return fmt.Errorf("different point height")
		}
		if p1.Hash != p2.Hash {
			return fmt.Errorf("different point hash %s %s", p1.Hash, p2.Hash)
		}
		if p1.Size != p2.Size {
			return fmt.Errorf("different point size %d %d", p1.Size, p2.Size)
		}
	}

	return nil
}

func TestHashHeightList_Serialize(t *testing.T) {
	var c = &HashHeightPointList{}

	for i := uint64(0); i < 5; i++ {
		hh := &HashHeightPoint{
			HashHeight: ledger.HashHeight{
				Height: i,
			},
			Size: i,
		}
		_, _ = crand.Read(hh.Hash[:])

		c.Points = append(c.Points, hh)
	}

	data, err := c.Serialize()
	if err != nil {
		panic(err)
	}

	var c2 = &HashHeightPointList{}
	err = c2.Deserialize(data)
	if err != nil {
		panic(err)
	}

	if err = compareHashHeightList(c, c2); err != nil {
		t.Error(err)
	}
}

func compareGetHashHeightList(c1, c2 *GetHashHeightList) error {
	for i, p := range c1.From {
		p2 := c2.From[i]
		if p2.Hash != p.Hash || p2.Height != p.Height {
			return fmt.Errorf("different fep hash: %s/%d %s/%d", p.Hash, p.Height, p2.Hash, p2.Height)
		}
	}

	if c1.Step != c2.Step {
		return fmt.Errorf("different step: %d %d", c1.Step, c2.Step)
	}

	if c1.To != c2.To {
		return fmt.Errorf("different to: %d %d", c1.To, c2.To)
	}

	return nil
}

func TestGetHashHeightList_Serialize(t *testing.T) {
	var c = &GetHashHeightList{
		Step: 100,
		To:   1000,
	}

	data, err := c.Serialize()
	if err != nil {
		panic(err)
	}

	var c2 = &GetHashHeightList{}
	err = c2.Deserialize(data)
	if err == nil {
		panic("error should not be nil")
	}

	var one, two types.Hash
	one[0] = 1
	two[0] = 2
	c.From = []*ledger.HashHeight{
		{100, one},
		{200, two},
	}

	data, err = c.Serialize()
	if err != nil {
		panic(err)
	}

	err = c2.Deserialize(data)
	if err != nil {
		panic(fmt.Sprintf("error should be nil: %v", err))
	}

	if err = compareGetHashHeightList(c, c2); err != nil {
		t.Error(err)
	}
}
