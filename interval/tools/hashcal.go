package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/vitelabs/go-vite/interval/common"
)

// SHA256 hasing
// calculateHash is a simple SHA256 hashing function
func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func CalculateAccountHash(block *common.AccountStateBlock) string {
	str := blockStr(block) + strconv.Itoa(block.Amount) +
		strconv.Itoa(block.ModifiedAmount) +
		strconv.FormatUint(block.SnapshotHeight, 10) +
		block.SnapshotHash +
		block.BlockType.String() +
		block.From +
		block.To
	if block.Source != nil {
		str += block.Source.Hash + strconv.FormatUint(block.Source.Height, 10)
	}
	return calculateHash(str)
}

func blockStr(block common.Block) string {
	return strconv.FormatInt(block.Timestamp().Unix(), 10) + string(block.Signer()) + string(block.PreHash()) + strconv.FormatUint(block.Height(), 10)
}

func CalculateSnapshotHash(block *common.SnapshotBlock) string {
	accStr := ""
	if block.Accounts != nil {
		for _, account := range block.Accounts {
			accStr = accStr + strconv.FormatUint(account.Height, 10) + account.Hash + account.Addr
		}
	}
	record := blockStr(block) + accStr
	return calculateHash(record)
}
