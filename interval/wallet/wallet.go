package wallet

import "sort"

type Wallet interface {
	Accounts() []string
	CreateAccount(address string) string
	SetCoinBase(string)
	CoinBase() string
}

func NewWallet() Wallet {
	w := &wallet{}
	w.accounts = make(map[string]string)
	return w
}

type wallet struct {
	accounts map[string]string
	current  string
}

func (w *wallet) Accounts() []string {
	var accs []string
	for k, _ := range w.accounts {
		accs = append(accs, k)
	}
	sort.Strings(accs)
	return accs
}
func (w *wallet) SetCoinBase(address string) {
	w.accounts[address] = address
	w.current = address
}
func (w *wallet) CoinBase() string {
	if w.current == "" {
		accs := w.Accounts()
		if len(accs) > 0 {
			w.current = accs[0]
		}
	}
	return w.current
}
func (w *wallet) CreateAccount(address string) string {
	w.accounts[address] = address
	return address
}

func (w *wallet) Sign(address string, data []byte) []byte {
	return data
}
