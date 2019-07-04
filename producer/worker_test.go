package producer

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
)

func TestStoreSeed(t *testing.T) {
	w := newWorker(nil, nil)

	hash := types.HexToHashPanic(calculateHash("100"))
	w.storeSeedHash(100, &hash)
	result := w.getSeedByHash(&hash)
	assert.Equal(t, uint64(100), result)
}

func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}
