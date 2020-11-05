package bridge

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type ethVerifier struct {
}

func (ethVerifier) verifyHeader(hash ethcommon.Hash, header *ethtypes.Header) bool {
	return true
}

func (ethVerifier) verifyReceipt(receipt *ethtypes.Receipt) bool {
	return true
}

func (ethVerifier) verifyProof(root ethcommon.Hash, leaf ethcommon.Hash, index int) bool {
	return true
}
