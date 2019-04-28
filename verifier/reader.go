package verifier

import (
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
)

type consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	generator.Consensus
}
