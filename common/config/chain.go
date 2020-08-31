package config

import "github.com/vitelabs/go-vite/common/types"

// chain config
type Chain struct {
	LedgerGcRetain uint64 // no use
	GenesisFile    string // genesis file path
	LedgerGc       bool   // open or close ledger garbage collector
	OpenPlugins    bool   // open or close chain plugins. eg, filter account blocks by token.

	VmLogWhiteList []types.Address // contract address white list which save VM logs
	VmLogAll       bool            // save all VM logs, it will cost more disk space
}
