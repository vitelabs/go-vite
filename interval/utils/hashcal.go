package utils

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

func CalculateAccountHash(block *common.AccountStateBlock) common.Hash {
	str := blockStr(block) + block.Amount.String() +
		block.ModifiedAmount.String() +
		block.BlockType.String() +
		block.From.String() +
		block.To.String()
	if block.Source != nil {
		str += block.Source.Hash.String() + block.Source.Height.String()
	}
	return common.Hash(calculateHash(str))
}

func blockStr(block common.Block) string {
	return strconv.FormatInt(block.Timestamp().Unix(), 10) + block.Signer().String() + block.PrevHash().String() + block.Height().String()
}

func CalculateSnapshotHash(block *common.SnapshotBlock) common.Hash {
	accStr := ""
	if block.Accounts != nil {
		for _, account := range block.Accounts {
			accStr = accStr + account.Height.String() + account.Hash.String() + account.Addr.String()
		}
	}
	record := blockStr(block) + accStr
	return common.Hash(calculateHash(record))
}
