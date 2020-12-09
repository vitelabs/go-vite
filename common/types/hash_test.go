package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashCmp(t *testing.T) {
	hash1, err := HexToHash("0000000000000000000000000000000000000000000000000000000000000001")
	assert.NoError(t, err)
	hash2, err := HexToHash("0000000000000000000000000000000000000000000000000000000000000002")
	assert.NoError(t, err)

	result := hash1.Cmp(hash2)

	t.Logf(fmt.Sprintf("%d", result))
}
