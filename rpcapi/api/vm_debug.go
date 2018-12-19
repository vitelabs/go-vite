package api

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/abi"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type VmDebugApi struct {
	vite       *vite.Vite
	log        log15.Logger
	wallet     *WalletApi
	tx         *Tx
	testapi    *TestApi
	onroad     *PublicOnroadApi
	contract   *ContractApi
	accountMap map[types.Address]string
}

func NewVmDebugApi(vite *vite.Vite) *VmDebugApi {
	api := &VmDebugApi{
		vite:       vite,
		log:        log15.New("module", "rpc_api/vmdebug_api"),
		wallet:     NewWalletApi(vite),
		tx:         NewTxApi(vite),
		onroad:     NewPublicOnroadApi(vite),
		contract:   NewContractApi(vite),
		accountMap: make(map[types.Address]string),
	}
	api.testapi = NewTestApi(api.wallet)
	return api
}

func (v VmDebugApi) String() string {
	return "VmDebugApi"
}

var (
	defaultPassphrase = "123"
	initAmount        = "0"
)

type AccountInfo struct {
	Addr       types.Address `json:"address"`
	PrivateKey string        `json:"privateKey"`
}

func (v *VmDebugApi) Init() (*AccountInfo, error) {
	// check genesis account status
	prevBlock, err := v.vite.Chain().GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		return nil, err
	}
	// receive genesis onroad tx
	if prevBlock == nil {
		onroadList, err := v.onroad.GetOnroadBlocksByAddress(ledger.GenesisAccountAddress, 0, 10)
		if err != nil {
			return nil, err
		}
		if len(onroadList) > 0 && onroadList[0].FromAddress == types.AddressMintage && onroadList[0].Height == "2" {
			err = v.testapi.ReceiveOnroadTx(CreateReceiveTxParms{
				SelfAddr:   ledger.GenesisAccountAddress,
				FromHash:   onroadList[0].Hash,
				PrivKeyStr: testapi_hexPrivKey,
			})
		}
		if err != nil {
			return nil, err
		}
	}
	// create new user account
	return v.NewAccount()
}

func (v *VmDebugApi) NewAccount() (*AccountInfo, error) {
	// create new user account
	response, err := v.wallet.NewMnemonicAndEntropyStore(defaultPassphrase)
	if err != nil {
		return nil, err
	}
	// unlock user account
	err = v.wallet.Unlock(response.Filename, defaultPassphrase)
	if err != nil {
		return nil, err
	}
	_, key, _, err := v.wallet.wallet.GlobalFindAddr(response.PrimaryAddr)
	if err != nil {
		return nil, err
	}
	privateKey, err := key.PrivateKey()
	if err != nil {
		return nil, err
	}
	acc := AccountInfo{
		Addr:       response.PrimaryAddr,
		PrivateKey: hex.EncodeToString(privateKey),
	}
	// transfer from genesis account to user account
	tid, _ := types.HexToTokenTypeId(testapi_tti)
	sendBlock, err := v.tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
		SelfAddr:    &ledger.GenesisAccountAddress,
		ToAddr:      &acc.Addr,
		TokenTypeId: tid,
		PrivateKey:  &testapi_hexPrivKey,
		Amount:      &initAmount,
	})
	if err != nil {
		return nil, err
	}

	err = v.testapi.ReceiveOnroadTx(CreateReceiveTxParms{
		SelfAddr:   acc.Addr,
		FromHash:   sendBlock.Hash,
		PrivKeyStr: acc.PrivateKey,
	})
	if err != nil {
		return nil, err
	}
	v.accountMap[acc.Addr] = acc.PrivateKey
	return &acc, nil
}

type CreateContractParam struct {
	FileName string   `json:"fileName"`
	Amount   string   `json:"amount"`
	Params   []string `json:"params"`
}
type CreateContractResult struct {
	AccountAddr       types.Address       `json:"accountAddr"`
	AccountPrivateKey string              `json:"accountPrivateKey"`
	ContractAddr      types.Address       `json:"contractAddr"`
	SendBlockHash     types.Hash          `json:"sendBlockHash"`
	MethodList        []CallContractParam `json:"methodList"'`
}

func (v *VmDebugApi) CreateContract(param CreateContractParam) (*CreateContractResult, error) {
	// compile solidity++ file
	code, abiJson, err := compile(param.FileName)
	if err != nil {
		return nil, err
	}
	// init and get test account
	testAccount, err := v.Init()
	if err != nil {
		return nil, err
	}
	// send create contract tx
	createContractData, err := v.contract.GetCreateContractData(types.DELEGATE_GID, code, abiJson, param.Params)
	if err != nil {
		return nil, err
	}
	if len(param.Amount) == 0 {
		param.Amount = "0"
	}
	sendBlock, err := v.tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
		SelfAddr:    &testAccount.Addr,
		TokenTypeId: ledger.ViteTokenId,
		PrivateKey:  &testAccount.PrivateKey,
		Amount:      &param.Amount,
		Data:        createContractData,
		BlockType:   ledger.BlockTypeSendCreate,
	})
	if err != nil {
		return nil, err
	}
	// save contractAddress and contract data
	if err := writeContractData(abiJson, sendBlock.ToAddress); err != nil {
		return nil, err
	}
	methodList, err := packMethodList(abiJson, sendBlock.ToAddress, testAccount.Addr)
	if err != nil {
		return nil, err
	}
	return &CreateContractResult{
		AccountAddr:       testAccount.Addr,
		AccountPrivateKey: testAccount.PrivateKey,
		ContractAddr:      sendBlock.ToAddress,
		SendBlockHash:     sendBlock.Hash,
		MethodList:        methodList,
	}, nil
}

type CallContractParam struct {
	ContractAddr types.Address  `json:"contractAddr"`
	AccountAddr  *types.Address `json:"accountAddr"`
	Amount       string         `json:"amount"`
	MethodName   string         `json:"methodName"`
	Params       []string       `json:"params"`
}
type CallContractResult struct {
	ContractAddr      types.Address `json:"contractAddr"`
	AccountAddr       types.Address `json:"accountAddr"`
	AccountPrivateKey string        `json:"accountPrivateKey"`
	SendBlockHash     types.Hash    `json:"sendBlockHash"`
}

func (v *VmDebugApi) CallContract(param CallContractParam) (*CallContractResult, error) {
	abiJson, err := readContractData(param.ContractAddr)
	if err != nil {
		return nil, err
	}
	data, err := v.contract.GetCallContractData(abiJson, param.MethodName, param.Params)
	if err != nil {
		return nil, err
	}
	if len(param.Amount) == 0 {
		param.Amount = "0"
	}
	var privateKey string
	if param.AccountAddr == nil || len(v.accountMap[*param.AccountAddr]) == 0 {
		testAccount, err := v.Init()
		if err != nil {
			return nil, err
		}
		param.AccountAddr = &testAccount.Addr
		privateKey = testAccount.PrivateKey
	} else {
		privateKey = v.accountMap[*param.AccountAddr]
	}
	sendBlock, err := v.tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
		SelfAddr:    param.AccountAddr,
		ToAddr:      &param.ContractAddr,
		TokenTypeId: ledger.ViteTokenId,
		PrivateKey:  &privateKey,
		Amount:      &param.Amount,
		Data:        data,
		BlockType:   ledger.BlockTypeSendCall,
	})
	if err != nil {
		return nil, err
	} else {
		return &CallContractResult{
			ContractAddr:      param.ContractAddr,
			AccountAddr:       *param.AccountAddr,
			AccountPrivateKey: privateKey,
			SendBlockHash:     sendBlock.Hash,
		}, nil
	}
}

func (v *VmDebugApi) GetContractList() (map[types.Address][]CallContractParam, error) {
	fileList, err := ioutil.ReadDir(getDir())
	if err != nil {
		return nil, err
	}
	resultMap := make(map[types.Address][]CallContractParam)
	if len(fileList) == 0 {
		return resultMap, nil
	}
	testAccount, err := v.Init()
	if err != nil {
		return nil, err
	}
	for _, file := range fileList {
		addr, err := types.HexToAddress(file.Name())
		if err != nil {
			continue
		}
		abiJson, err := readContractData(addr)
		if err != nil {
			return nil, err
		}
		resultMap[addr], err = packMethodList(abiJson, addr, testAccount.Addr)
		if err != nil {
			return nil, err
		}
	}
	return resultMap, nil
}

func (v *VmDebugApi) ClearData() error {
	dir := getDir()
	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range fileList {
		if !file.IsDir() {
			os.Remove(filepath.Join(dir, file.Name()))
		}
	}
	v.accountMap = make(map[types.Address]string, 0)
	return nil
}

func compile(fileName string) (string, string, error) {
	cmd := exec.Command("./solc", "--bin", "--abi", fileName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", errors.New(string(out))
	}
	compileResult := strings.Split(string(out), "\n")
	var code, abiJson string
	for i := 0; i < len(compileResult); i++ {
		if compileResult[i] == "Binary: " && i < len(compileResult)-1 {
			code = compileResult[i+1]
			i++
		} else if compileResult[i] == "Contract JSON ABI " && i < len(compileResult)-1 {
			abiJson = compileResult[i+1]
			i++
		}
	}
	if len(code) == 0 || len(abiJson) == 0 {
		return "", "", errors.New("code len is 0")
	}
	return code, abiJson, nil
}

func packMethodList(abiJson string, contractAddr types.Address, accountAddr types.Address) ([]CallContractParam, error) {
	abiContract, err := abi.JSONToABIContract(strings.NewReader(abiJson))
	if err != nil {
		return nil, err
	}
	methodList := make([]CallContractParam, 0)
	for name, method := range abiContract.Methods {
		params := make([]string, len(method.Inputs))
		for i, input := range method.Inputs {
			params[i] = input.Type.String()
		}
		methodList = append(methodList, CallContractParam{
			ContractAddr: contractAddr,
			AccountAddr:  &accountAddr,
			Amount:       "0",
			MethodName:   name,
			Params:       params,
		})
	}
	return methodList, nil
}

func writeContractData(abiJson string, addr types.Address) error {
	os.MkdirAll(getDir(), os.ModePerm)
	return ioutil.WriteFile(getFileName(addr), []byte(abiJson), os.ModePerm)
}

func readContractData(addr types.Address) (string, error) {
	data, err := ioutil.ReadFile(getFileName(addr))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getDir() string {
	return filepath.Join(dataDir, "contracts")
}
func getFileName(addr types.Address) string {
	return filepath.Join(getDir(), addr.String())
}
