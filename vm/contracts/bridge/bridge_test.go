package bridge

import (
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

	header := simpleHeader{height: 1}

	// 2
	proof, err := b.Proof(header)
	assert.NoError(t, err)
	assert.False(t, proof)

	// 3
	err = b.Submit(header)
	assert.NoError(t, err)

	// 4
	proof, err = b.Proof(header)
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
	header := &simpleHeader{height: 1}
	tx := randSimpleTx(1)

	// 2
	result, err = input.Input(tx)
	assert.NoError(t, err)
	assert.Equal(t, Input_Failed_Error, result)

	// 3
	err = b.Submit(header)
	assert.NoError(t, err)

	// 4
	result, err = input.Input(tx)
	assert.NoError(t, err)
	assert.Equal(t, Input_Success, result)

	// 5
	result, err = input.Input(tx)
	assert.NoError(t, err)
	assert.Equal(t, Input_Failed_Duplicated, result)
}
