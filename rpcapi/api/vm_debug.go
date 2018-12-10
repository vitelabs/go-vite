package api

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
)

type VmDebugApi struct {
	c chain.Chain
	log   log15.Logger
	wallet *WalletApi
	tx *Tx
	testapi *TestApi
	onroad *PublicOnroadApi
}

func NewVmDebugApi(vite *vite.Vite) *VmDebugApi {
	api :=  &VmDebugApi{
		c: vite.Chain(),
		log:   log15.New("module", "rpc_api/vmdebug_api"),
		wallet: NewWalletApi(vite),
		tx: NewTxApi(vite),
		onroad:NewPublicOnroadApi(vite),
	}
	api.testapi = NewTestApi(api.wallet)
	return api
}

func (v VmDebugApi) String() string {
	return "VmDebugApi"
}

var (
	defaultPassphrase = "123"
	initAmount = "0"
)

type AccountInfo struct {
	Addr types.Address `json:"address"`
	PrivateKey string `json:"privateKey"`
}

func (v *VmDebugApi) Init() (*AccountInfo, error) {
	// check genesis account status
	prevBlock, err := v.c.GetLatestAccountBlock(&ledger.GenesisAccountAddress)
	if err != nil {
		return nil, err
	}
	// receive genesis onroad tx
	if prevBlock == nil {
		onroadList, err := v.onroad.GetOnroadBlocksByAddress(ledger.GenesisAccountAddress, 0, 10)
		if err != nil {
			return nil, err
		}
		if len(onroadList) > 0 && onroadList[0].FromAddress == types.AddressMintage && onroadList[0].Height == "2"{
			err = v.testapi.ReceiveOnroadTx(CreateReceiveTxParms{
				SelfAddr:ledger.GenesisAccountAddress,
				FromHash:onroadList[0].Hash,
				PrivKeyStr:testapi_hexPrivKey,
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
		Addr:response.PrimaryAddr,
		PrivateKey:hex.EncodeToString(privateKey),
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
		SelfAddr:acc.Addr,
		FromHash:sendBlock.Hash,
		PrivKeyStr:acc.PrivateKey,
		})
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

