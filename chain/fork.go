package chain

import (
	"github.com/vitelabs/go-vite/common/types"
)

type fork struct {
	// example, need modify
	Vite1Height uint64
	Vite1Hash   *types.Hash

	chn Chain
}

func NewFork(chn Chain, config *GenesisConfig) *fork {
	// default config
	testHash, _ := types.HexToHash("8c40ac404065c1e63a06243c65fe313fb241b48029f9a6ea60c91aa5061ad991")
	f := &fork{
		chn:         chn,
		Vite1Height: 1,
		Vite1Hash:   &testHash,
	}

	cf := config.Fork
	if cf != nil {
		if cf.Vite1Hash != nil {
			f.Vite1Hash = cf.Vite1Hash
		}
		if cf.Vite1Height > 0 {
			f.Vite1Height = cf.Vite1Height
		}
	}
	return f
}
func (f *fork) checkForkPoints() (bool, error) {
	// check Vite1 upgrade
	vite1block, err := f.chn.GetSnapshotBlockByHash(f.Vite1Hash)
	if err != nil {
		return false, err
	}
	if vite1block == nil || vite1block.Height != f.Vite1Height {
		return false, nil
	}

	return true, nil
}

func (f *fork) IsVite1(blockHash types.Hash) bool {
	return f.Vite1Hash != nil && *f.Vite1Hash == blockHash
}

func (f *fork) IsVite1ByHeight(blockHeight uint64) bool {
	return f.Vite1Height == blockHeight
}
