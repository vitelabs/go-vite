package consensus

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	core "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type virtualVerifier struct {
}

func NewVirtualVerifier() interfaces.ConsensusVerifier {
	return &virtualVerifier{}
}

func (virtualVerifier) VerifyAccountProducer(block *core.AccountBlock) (bool, error) {
	return true, nil
}

func (virtualVerifier) VerifyABsProducer(abs map[types.Gid][]*core.AccountBlock) ([]*core.AccountBlock, error) {
	var result []*core.AccountBlock
	for _, v := range abs {
		result = append(result, v...)
	}
	return result, nil
}

func (virtualVerifier) VerifySnapshotProducer(block *core.SnapshotBlock) (bool, error) {
	return true, nil
}
