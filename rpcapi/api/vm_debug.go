package api

import (
	"bytes"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_db"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	initAmount        = "100000000000000000000000"
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

	if vm.IsTest() {
		err = v.testapi.ReceiveOnroadTx(CreateReceiveTxParms{
			SelfAddr:   acc.Addr,
			FromHash:   sendBlock.Hash,
			PrivKeyStr: acc.PrivateKey,
		})
		if err != nil {
			return nil, err
		}
	}
	v.accountMap[acc.Addr] = acc.PrivateKey
	return &acc, nil
}

type CreateContractParam struct {
	FileName    string                      `json:"fileName"`
	Params      map[string]ConstructorParam `json:"params"`
	AccountAddr *types.Address              `json:"accountAddr"`
}
type ConstructorParam struct {
	Amount string   `json:"amount"`
	Params []string `json:"params"`
}
type CreateContractResult struct {
	AccountAddr       types.Address       `json:"accountAddr"`
	AccountPrivateKey string              `json:"accountPrivateKey"`
	ContractAddr      types.Address       `json:"contractAddr"`
	SendBlockHash     types.Hash          `json:"sendBlockHash"`
	MethodList        []CallContractParam `json:"methodList"'`
}

func (v *VmDebugApi) CreateContract(param CreateContractParam) ([]*CreateContractResult, error) {
	// compile solidity++ file
	compileResultList, err := compile(param.FileName)
	if err != nil {
		return nil, err
	}
	// init and get test account
	var testAccount *AccountInfo
	if param.AccountAddr == nil || len(v.accountMap[*param.AccountAddr]) == 0 {
		testAccount, err = v.Init()
		if err != nil {
			return nil, err
		}
	} else {
		testAccount = &AccountInfo{*param.AccountAddr, v.accountMap[*param.AccountAddr]}
	}

	resultList := make([]*CreateContractResult, 0)
	for _, c := range compileResultList {
		txParam := param.Params[c.name]
		// send create contract tx
		createContractData, err := v.contract.GetCreateContractData(types.DELEGATE_GID, 1, c.code, c.abiJson, txParam.Params)
		if err != nil {
			return nil, err
		}
		if len(txParam.Amount) == 0 {
			txParam.Amount = "0"
		}
		sendBlock, err := v.tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
			SelfAddr:    &testAccount.Addr,
			TokenTypeId: ledger.ViteTokenId,
			PrivateKey:  &testAccount.PrivateKey,
			Amount:      &txParam.Amount,
			Data:        createContractData,
			BlockType:   ledger.BlockTypeSendCreate,
		})
		if err != nil {
			return nil, err
		}
		// save contractAddress and contract data
		if err := writeContractData(c.abiJson, sendBlock.ToAddress); err != nil {
			return nil, err
		}
		methodList, err := packMethodList(c.abiJson, sendBlock.ToAddress, testAccount.Addr)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, &CreateContractResult{
			AccountAddr:       testAccount.Addr,
			AccountPrivateKey: testAccount.PrivateKey,
			ContractAddr:      sendBlock.ToAddress,
			SendBlockHash:     sendBlock.Hash,
			MethodList:        methodList,
		})
	}
	return resultList, nil
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

func (v *VmDebugApi) GetContractStorage(addr types.Address) (map[string]string, error) {
	sb := v.vite.Chain().GetLatestSnapshotBlock()
	prev, err := v.vite.Chain().GetLatestAccountBlock(&addr)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(v.vite.Chain(), &addr, &sb.Hash, &prev.Hash)
	if err != nil {
		return nil, err
	}
	iter, err := db.NewStorageIterator(nil)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for {
		if !iter.Next() {
			return m, nil
		}
		if !bytes.HasPrefix(iter.Key(), []byte("$code")) && !bytes.HasPrefix(iter.Key(), []byte("$balance")) {
			m["0x"+hex.EncodeToString(iter.Key())] = "0x" + hex.EncodeToString(iter.Value())
		}
	}
}

type compileResult struct {
	name    string
	code    string
	abiJson string
}

func compile(fileName string) ([]compileResult, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("solc", "--bin", "--abi", fileName)
	} else {
		cmd = exec.Command("./solc", "--bin", "--abi", fileName)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(strings.Trim(string(out), "\n"))
	}
	list := strings.Split(string(out), "\n")
	codeList := make([]compileResult, 0)
	for i := 0; i < len(list); i++ {
		if runtime.GOOS == "windows" {
			if strings.HasPrefix(list[i], "=======") && i < len(list)-4 && strings.HasPrefix(list[i+1], "Binary: ") && strings.HasPrefix(list[i+3], "Contract JSON ABI") {
				if len(list[i+2]) == 0 || len(list[i+4]) == 0 {
					return nil, errors.New("code len is 0")
				}
				name := list[i][9+len(fileName) : len(list[i])-9]
				codeList = append(codeList, compileResult{name, list[i+2][:len(list[i+2])-1], list[i+4][:len(list[i+4])-1]})
				i = i + 4
			}
		} else {
			if strings.HasPrefix(list[i], "=======") && i < len(list)-4 && list[i+1] == "Binary: " && list[i+3] == "Contract JSON ABI " {
				if len(list[i+2]) == 0 || len(list[i+4]) == 0 {
					return nil, errors.New("code len is 0")
				}
				name := list[i][9+len(fileName) : len(list[i])-8]
				codeList = append(codeList, compileResult{name, list[i+2], list[i+4]})
				i = i + 4
			}
		}
	}
	if len(codeList) == 0 {
		return nil, errors.New("contract len is 0")
	}
	return codeList, nil
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
