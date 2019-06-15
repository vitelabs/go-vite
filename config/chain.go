package config

type Chain struct {
	LedgerGcRetain uint64
	GenesisFile    string
	LedgerGc       bool
	OpenPlugins    bool
}
