package vm

import "github.com/vitelabs/go-vite/common/types"

var (
	AddressRegister, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	AddressVote, _     = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
)

type precompiledContract interface {
	gasCost(block VmAccountBlock) uint64
	doSend(vm *VM, block VmAccountBlock) error
	doReceive(vm *VM, block VmAccountBlock) error
}

var simpleContracts = map[types.Address]precompiledContract{
	AddressRegister: &register{},
	AddressVote:     &vote{},
}

func getPrecompiledContract(address types.Address) (precompiledContract, bool) {
	p, ok := simpleContracts[address]
	return p, ok
}

type register struct{}

func (r *register) gasCost(block VmAccountBlock) uint64 {
	// TODO
	return 0
}
func (r *register) doSend(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}
func (r *register) doReceive(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}

type vote struct{}

func (v *vote) gasCost(block VmAccountBlock) uint64 {
	// TODO
	return 0
}
func (v *vote) doSend(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}
func (v *vote) doReceive(vm *VM, block VmAccountBlock) error {
	// TODO
	return nil
}
