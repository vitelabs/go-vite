package api

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
)

type RegisterApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewRegisterApi(vite *vite.Vite) *RegisterApi {
	return &RegisterApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/register_api"),
	}
}

func (r RegisterApi) String() string {
	return "RegisterApi"
}

func (r *RegisterApi) GetSignDataForRegister(pledgeAddr types.Address, gid types.Gid) []byte {
	return contracts.GetRegisterMessageForSignature(pledgeAddr, gid)
}

func (r *RegisterApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address, publicKey []byte, signature []byte) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameRegister, gid, name, nodeAddr, publicKey, signature)
}
func (r *RegisterApi) GetCancelRegisterData(gid types.Gid, name string) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameCancelRegister, gid, name)
}
func (r *RegisterApi) GetRewardData(gid types.Gid, name string, beneficialAddr types.Address, endHeight uint64, startHeight uint64) ([]byte, error) {
	db, err := vm_context.NewVmContext(r.chain, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	old := contracts.GetRegistration(db, name, gid)
	if old == nil {
		return nil, errors.New("registration not exist")
	}
	_, data, err := contracts.GetRewardData(db, old, gid, name, beneficialAddr, endHeight, startHeight)
	return data, err
}
func (r *RegisterApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address, publicKey []byte, signature []byte) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameUpdateRegistration, gid, name, nodeAddr, publicKey, signature)
}
