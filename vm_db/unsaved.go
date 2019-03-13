package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type Unsaved struct {
	contractMeta *ledger.ContractMeta

	code []byte

	logList ledger.VmLogList
	storage map[string][]byte

	balanceMap map[types.TokenTypeId]*big.Int
}

func NewUnsaved() *Unsaved {
	return &Unsaved{
		logList: make(ledger.VmLogList, 0),

		storage:    make(map[string][]byte),
		balanceMap: make(map[types.TokenTypeId]*big.Int),
	}
}

func (unsaved *Unsaved) SetValue(key []byte, value []byte) {
	unsaved.storage[string(key)] = value
}

func (unsaved *Unsaved) GetValue(key []byte) ([]byte, bool) {
	result, ok := unsaved.storage[string(key)]
	return result, ok
}

func (unsaved *Unsaved) GetBalance(tokenTypeId *types.TokenTypeId) (*big.Int, bool) {
	result, ok := unsaved.balanceMap[*tokenTypeId]
	return result, ok
}

func (unsaved *Unsaved) SetBalance(tokenTypeId *types.TokenTypeId, amount *big.Int) {
	unsaved.balanceMap[*tokenTypeId] = amount
}

func (unsaved *Unsaved) AddLog(log *ledger.VmLog) {
	unsaved.logList = append(unsaved.logList, log)
}

func (unsaved *Unsaved) GetLogListHash() *types.Hash {
	return unsaved.logList.Hash()
}

func (unsaved *Unsaved) SetContractMeta(contractMeta *ledger.ContractMeta) {
	unsaved.contractMeta = contractMeta
}
func (unsaved *Unsaved) SetCode(code []byte) {
	unsaved.code = code
}

func (unsaved *Unsaved) GetCode() []byte {
	return unsaved.code
}
