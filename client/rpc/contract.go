//

package rpc

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/rpc"
	"github.com/vitelabs/go-vite/v2/rpcapi/api"
)

// ContractApi ...
type ContractApi interface {
	CallOffChainMethod(param api.CallOffChainMethodParam) ([]byte, error)  // Deprecated: Use Query() instead
	Query(param api.QueryParam) ([]byte, error)  // Executes a synchronous call immediately without sending a transaction to the blockchain
	GetCreateContractData(param api.CreateContractDataParam) ([]byte, error)
	GetContractStorage(addr types.Address, prefix string) (map[string]string, error)
	GetContractInfo(addr types.Address) (*api.ContractInfo, error)
	GetSBPVoteList() ([]*api.SBPVoteInfo, error)
}

type contractApi struct {
	cc *rpc.Client
}

func NewContractApi(cc *rpc.Client) ContractApi {
	return &contractApi{cc: cc}
}

func (ci contractApi) GetCreateContractData(param api.CreateContractDataParam) (result []byte, err error) {
	err = ci.cc.Call(&result, "contract_getCreateContractData", param)
	return
}
func (ci contractApi) CallOffChainMethod(param api.CallOffChainMethodParam) (result []byte, err error) {
	err = ci.cc.Call(&result, "contract_callOffChainMethod", param)
	return
}

func (ci contractApi) Query(param api.QueryParam) (result []byte, err error) {
	err = ci.cc.Call(&result, "contract_query", param)
	return
}

func (ci contractApi) GetContractStorage(addr types.Address, prefix string) (result map[string]string, err error) {
	result = make(map[string]string)
	err = ci.cc.Call(&result, "contract_getContractStorage", addr, prefix)
	return
}

func (ci contractApi) GetContractInfo(addr types.Address) (result *api.ContractInfo, err error) {
	result = &api.ContractInfo{}
	err = ci.cc.Call(&result, "contract_getContractInfo", addr)
	return
}

func (ci contractApi) GetSBPVoteList() (result []*api.SBPVoteInfo, err error) {
	err = ci.cc.Call(&result, "contract_getSBPVoteList")
	return
}
