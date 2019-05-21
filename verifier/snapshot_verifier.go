package verifier

import (
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	css "github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
)

type SnapshotVerifier struct {
	reader chain.Chain
	cs     css.Verifier
}

func NewSnapshotVerifier(ch chain.Chain, cs css.Verifier) *SnapshotVerifier {
	verifier := &SnapshotVerifier{reader: ch, cs: cs}
	return verifier
}

func (self *SnapshotVerifier) VerifyNetSb(block *ledger.SnapshotBlock) error {
	if err := self.verifyTimestamp(block); err != nil {
		return err
	}
	if err := self.verifyDataValidity(block); err != nil {
		return err
	}
	return nil
}

func (self *SnapshotVerifier) verifyTimestamp(block *ledger.SnapshotBlock) error {
	if block.Timestamp == nil {
		return errors.New("timestamp is nil")
	}

	if block.Timestamp.After(time.Now().Add(time.Hour)) {
		return errors.New("snapshot Timestamp not arrive yet")
	}
	return nil
}

func (self *SnapshotVerifier) verifyDataValidity(block *ledger.SnapshotBlock) error {
	computedHash := block.ComputeHash()
	if block.Hash.IsZero() || computedHash != block.Hash {
		return ErrVerifyHashFailed
	}

	if self.reader.IsGenesisSnapshotBlock(block.Hash) {
		return nil
	}

	if len(block.Signature) == 0 || len(block.PublicKey) == 0 {
		return errors.New("signature or publicKey is nil")
	}
	isVerified, _ := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)
	if !isVerified {
		return ErrVerifySignatureFailed
	}
	return nil
}

func (self *SnapshotVerifier) verifySelf(block *ledger.SnapshotBlock, stat *SnapshotBlockVerifyStat) error {
	defer monitor.LogTime("verify", "snapshotSelf", time.Now())

	if block.Height == types.GenesisHeight {
		flag := self.reader.IsGenesisSnapshotBlock(block.Hash)
		if !flag {
			stat.result = FAIL
			return errors.Errorf("genesis block[%s] error.", block.Hash)
		}
	}

	if block.Seed != 0 {
		seedBlock := self.getLastSeedBlock(block)
		if seedBlock != nil {
			hash := ledger.ComputeSeedHash(block.Seed, seedBlock.PrevHash, seedBlock.Timestamp)
			if hash != *seedBlock.SeedHash {
				return errors.Errorf("seed verify fail. %s-%d", seedBlock.Hash, seedBlock.Height)
			}
		}
	}

	head := self.reader.GetLatestSnapshotBlock()
	if head.Height != block.Height-1 {
		return errors.Errorf("snapshot fail for height:[%d]", head.Height)
	}
	if head.Hash != block.PrevHash {
		return errors.Errorf("block is not next. prevHash:%s, headHash:%s", block.PrevHash, head.Hash)
	}
	return nil
}

func (self *SnapshotVerifier) verifyAccounts(block *ledger.SnapshotBlock, prev *ledger.SnapshotBlock, stat *SnapshotBlockVerifyStat) error {
	defer monitor.LogTime("verify", "snapshotAccounts", time.Now())

	for addr, b := range block.SnapshotContent {
		ab, e := self.reader.GetAccountBlockByHeight(addr, b.Height)
		if e != nil {
			return e
		}
		if ab == nil {
			stat.results[addr] = PENDING
		} else if ab.Hash == b.Hash {
			stat.results[addr] = SUCCESS
		} else {
			stat.results[addr] = FAIL
			stat.result = FAIL
			return errors.New(fmt.Sprintf("account[%s] fork, height:[%d], hash:[%s]",
				addr.String(), b.Height, b.Hash))
		}
	}
	for _, v := range stat.results {
		if v != SUCCESS {
			return nil
		}
	}
	return nil
}

func (self *SnapshotVerifier) VerifyReferred(block *ledger.SnapshotBlock) *SnapshotBlockVerifyStat {
	defer monitor.LogTime("verify", "snapshotBlock", time.Now())
	stat := self.newVerifyStat(block)
	// todo add state_bak check
	err := self.verifySelf(block, stat)
	if err != nil {
		stat.errMsg = err.Error()
		return stat
	}

	head := self.reader.GetLatestSnapshotBlock()
	if !block.Timestamp.After(*head.Timestamp) {
		stat.result = FAIL
		stat.errMsg = "timestamp must be greater."
		return stat
	}

	// verify accounts exist
	err = self.verifyAccounts(block, head, stat)
	if err != nil {
		stat.errMsg = err.Error()
		return stat
	}
	for _, v := range stat.results {
		if v == FAIL || v == PENDING {
			return stat
		}
	}

	if block.Height != types.GenesisHeight {
		// verify producer
		result, e := self.cs.VerifySnapshotProducer(block)
		if e != nil {
			stat.result = FAIL
			stat.errMsg = e.Error()
			return stat
		}
		if !result {
			stat.result = FAIL
			stat.errMsg = "verify snapshot producer fail."
			return stat
		}
	}
	stat.result = SUCCESS
	return stat
}

const seedDuration = time.Minute * 10

func (self *SnapshotVerifier) getLastSeedBlock(head *ledger.SnapshotBlock) *ledger.SnapshotBlock {
	// todo
	//t := head.Timestamp.Add(-seedDuration)
	//addr := head.Producer()
	//blocks, err := self.reader.GetSnapshotBlocksAfterAndEqualTime(head.Height, &t, &addr)
	//if err != nil {
	//	return nil
	//}
	//for _, v := range blocks {
	//	var seedHash = v.SeedHash
	//	if seedHash != nil {
	//		return v
	//	}
	//}

	return nil
}

//func (self *SnapshotVerifier) VerifyProducer(block *ledger.SnapshotBlock) *SnapshotBlockVerifyStat {
//	defer monitor.LogTime("verify", "snapshotProducer", time.Now())
//	vStat := self.newVerifyStat(block)
//	return vStat
//}

type AccountHashH struct {
	Addr   *types.Address
	Hash   *types.Hash
	Height *big.Int
}

type SnapshotBlockVerifyStat struct {
	result       VerifyResult
	results      map[types.Address]VerifyResult
	errMsg       string
	accountTasks []*AccountPendingTask
	snapshotTask *SnapshotPendingTask
}

func (self *SnapshotBlockVerifyStat) ErrMsg() string {
	return self.errMsg
}

func (self *SnapshotBlockVerifyStat) VerifyResult() VerifyResult {
	return self.result
}

func (self *SnapshotBlockVerifyStat) Results() map[types.Address]VerifyResult {
	return self.results
}

func (self *SnapshotVerifier) newVerifyStat(b *ledger.SnapshotBlock) *SnapshotBlockVerifyStat {
	stat := &SnapshotBlockVerifyStat{result: PENDING}
	stat.results = make(map[types.Address]VerifyResult)
	for k := range b.SnapshotContent {
		stat.results[k] = PENDING
	}
	return stat
}
