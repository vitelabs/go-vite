package wallet

import "github.com/vitelabs/go-vite/common/types"

type Manager struct {
	providers map[string][]Provider

	wallets []Wallet // all wallets in all providers
}

func (wm Manager) Find(a types.Address) (Wallet, error) {
	return nil, nil
}

func (wm Manager) Wallets() []Wallet {
	return nil
}

func (wm Manager) Providers(kind string) []Provider {
	return wm.providers[kind]
}
