package mobile_test

import (
	"encoding/base64"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/mobile"
	"math/big"
	"testing"
	"time"
)

func TestClient_GetBlocksByAccAddr(t *testing.T) {
	client, _ := mobile.Dial("https://testnet.vitewallet.com/ios")
	a, _ := mobile.NewAddressFromString("vite_328aecc858abe80f4d9530cfdf15536082e51f4ef8cb870f33")
	s, err := client.GetBlocksByAccAddr(a, 0, 20)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}

}

func TestClient_GetAccountByAccAddr(t *testing.T) {
	client, _ := mobile.Dial("https://testnet.vitewallet.com/ios")
	a, _ := mobile.NewAddressFromString("vite_328aecc858abe80f4d9530cfdf15536082e51f4ef8cb870f33")
	s, err := client.GetAccountByAccAddr(a)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}

}

func TestClient_GetFittestSnapshotHash(t *testing.T) {
	client, _ := mobile.Dial("https://testnet.vitewallet.com/ios")
	a, _ := mobile.NewAddressFromString("vite_328aecc858abe80f4d9530cfdf15536082e51f4ef8cb870f33")
	s, err := client.GetFittestSnapshotHash(a, "")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(s)
	}
}

func TestClient_SendRawTx(t *testing.T) {
	block := ledger.AccountBlock{
		BlockType: 4,
		Height:    3,
	}
	block.AccountAddress, _ = types.HexToAddress("vite_328aecc858abe80f4d9530cfdf15536082e51f4ef8cb870f33")
	block.Difficulty = new(big.Int).SetInt64(67108864)
	block.FromBlockHash, _ = types.HexToHash("1867836bdbffb5910b4aa80710eaf11e7d50e8553ef0decc803f5d415703b4ea")
	block.Nonce, _ = base64.StdEncoding.DecodeString("pmwhyLOe2+I=")
	block.PrevHash, _ = types.HexToHash("8e263bfb967a67ff6ff7ecd5a94126f0fc6eb290795f0f618132609c5d1a76bc")
	block.SnapshotHash, _ = types.HexToHash("5911c59bdb7ae7b5a52203b9c77b866159252654996eb468036842796b9b7272")
	tt := time.Unix(int64(1543910066), 0)
	block.Timestamp = &tt
	fmt.Println(block.ComputeHash())

	block.PublicKey, _ = base64.StdEncoding.DecodeString("E43oIp3F0eFgRBy7MlaH8GFvIPnizfbejWgD7G2GZq0=")
	block.Signature, _ = base64.StdEncoding.DecodeString("WHwZA8r1RTn7s5LIqwIEomwGZ3AVoxDEVf3HXjEopXpapHk+c4tNoRaR9nirCN0B7WViAqrEDK4XJN9HLnOZDw==")
	block.Hash, _ = types.HexToHash("A75B469193FD60038CFE2F465113D2CDD13540D73526D1A6F8F16B7D41419CCA")
}
