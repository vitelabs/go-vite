package consensus

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	core "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type simpleVerifier struct {
}

func NewSimpleVerifier() interfaces.ConsensusVerifier {
	return &simpleVerifier{}
}

func (simpleVerifier) VerifyAccountProducer(block *core.AccountBlock) (bool, error) {
	return true, nil
}

func (simpleVerifier) VerifyABsProducer(abs map[types.Gid][]*core.AccountBlock) ([]*core.AccountBlock, error) {
	var result []*core.AccountBlock
	for _, v := range abs {
		result = append(result, v...)
	}
	return result, nil
}

func (simpleVerifier) VerifySnapshotProducer(block *core.SnapshotBlock) (bool, error) {
	return true, nil
}
