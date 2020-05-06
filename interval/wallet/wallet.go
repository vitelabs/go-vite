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

func (self *wallet) Accounts() []string {
	var accs []string
	for k, _ := range self.accounts {
		accs = append(accs, k)
	}
	sort.Strings(accs)
	return accs
}
func (self *wallet) SetCoinBase(address string) {
	self.accounts[address] = address
	self.current = address
}
func (self *wallet) CoinBase() string {
	if self.current == "" {
		accs := self.Accounts()
		if len(accs) > 0 {
			self.current = accs[0]
		}
	}
	return self.current
}
func (self *wallet) CreateAccount(address string) string {
	self.accounts[address] = address
	return address
}

func (self *wallet) Sign(address string, data []byte) []byte {
	return data
}
