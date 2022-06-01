package consensus

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	core "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type noneVerifier struct {
}

func NewNoneVerifier() interfaces.ConsensusVerifier {
	return &noneVerifier{}
}

func (noneVerifier) VerifyAccountProducer(block *core.AccountBlock) (bool, error) {
	return true, nil
}

func (noneVerifier) VerifyABsProducer(abs map[types.Gid][]*core.AccountBlock) ([]*core.AccountBlock, error) {
	var result []*core.AccountBlock
	for _, v := range abs {
		result = append(result, v...)
	}
	return result, nil
}

func (noneVerifier) VerifySnapshotProducer(block *core.SnapshotBlock) (bool, error) {
	return true, nil
}
