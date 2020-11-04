package bridge

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 1. init bridge
// 2. assert proof failed
// 3. submit
// 4. assert proof success
func TestBridgeSimple(t *testing.T) {
	var b Bridge
	var err error
	{
		// 1
		b, err = newBridgeSimple()
		assert.NoError(t, err)
		assert.NotNil(t, b)
	}

	// 2
	proof, err := b.Proof(big.NewInt(0), []byte{})
	assert.NoError(t, err)
	assert.False(t, proof)

	// 3
	err = b.Submit(big.NewInt(0), []byte{})
	assert.NoError(t, err)

	// 4
	proof, err = b.Proof(big.NewInt(0), []byte{})
	assert.NoError(t, err)
	assert.True(t, proof)
}

// 1. init bridge and collector
// 2. assert input fail
// 3. submit to bridge
// 4. assert input success
// 5. assert input duplicated
func TestTxInput(t *testing.T) {
	var b Bridge
	var input InputCollector
	var err error
	{
		// 1
		b, err = newBridgeSimple()
		assert.NoError(t, err)
		assert.NotNil(t, b)
		input, err = newInputSimpleTx(b)
		assert.NoError(t, err)
		assert.NotNil(t, input)
	}
	var result InputResult
	height := big.NewInt(1)
	tx := randSimpleTx()

	// 2
	result, err = input.Input(height, tx)
	assert.NoError(t, err)
	assert.Equal(t, result, Input_Failed_Error)

	// 3
	err = b.Submit(height, []byte{})
	assert.NoError(t, err)

	// 4
	result, err = input.Input(height, tx)
	assert.NoError(t, err)
	assert.Equal(t, result, Input_Success)

	// 5
	result, err = input.Input(height, tx)
	assert.NoError(t, err)
	assert.Equal(t, result, Input_Failed_Duplicated)
}
