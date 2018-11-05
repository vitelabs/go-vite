package chain

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync"
	"testing"
)

func TestNewAbmLocker(t *testing.T) {
	abmLocker := newAbmLocker()
	hash1, _ := types.HexToHash("08bd789f93e9de76d32363d5ee044036c90bedf27317adc0240867743734250b")
	hash2, _ := types.HexToHash("14b4650404d3d0469db2d03e57f5d5f0447cdbd7a9d5ba95b4655ec4539bd31a")
	//hash3, _ := types.HexToHash("17552839420ef7be28165e8f4412eec0d9988823ecd0e9e8b66745debc0e9cfc")

	var wg sync.WaitGroup
	var counter = make(map[types.Hash]uint64, 1000000)
	abmLocker.Lock(hash1)
	abmLocker.Lock(hash2)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				abmLocker.Lock(hash1)
				counter[hash1]++
				abmLocker.Unlock(hash1)
			}

			//abmLocker.Lock(hash2)
			//counter[hash2]++
			//abmLocker.Unlock(hash2)
			//
			//abmLocker.Lock(hash3)
			//counter[hash3]++
			//abmLocker.Unlock(hash3)

		}()
	}

	wg.Wait()

}
