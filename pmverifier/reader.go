package pmverifier

import "github.com/vitelabs/go-vite/ledger"

type Consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
}
