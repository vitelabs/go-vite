package api

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/entropystore"
	"math/big"
)

type HexSignedTuple struct {
	Message    string `json:"message"`
	SignedData string `json:"signedData"`
	Pubkey     string `json:"pubkey"`
}

type NewStoreResponse struct {
	Mnemonic    string        `json:"mnemonic"`
	PrimaryAddr types.Address `json:"primaryAddr"`
	Filename    string        `json:"filename"`
}

type CreateTransferTxParms struct {
	SelfAddr    types.Address
	ToAddr      types.Address
	TokenTypeId types.TokenTypeId
	Passphrase  string
	Amount      string
	Data        []byte
	Difficulty  *big.Int
}

type IsMayValidKeystoreFileResponse struct {
	Maybe      bool
	MayAddress types.Address
}

func NewWalletApi(vite *vite.Vite) *WalletApi {
	return &WalletApi{
		wallet: vite.WalletManager(),
		chain:  vite.Chain(),
		pool:   vite.Pool(),
	}
}

type WalletApi struct {
	wallet *wallet.Manager
	chain  chain.Chain
	pool   pool.Writer
}

func (m WalletApi) String() string {
	return "WalletApi"
}

func (m WalletApi) ListEntropyFiles() ([]string, error) {
	return m.wallet.ListEntropyFiles()
}

func (m WalletApi) ListCurrentStoreAddress(maxIndex uint32) ([]types.Address, error) {
	log.Info("ListCurrentStoreAddress")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return nil, e
	}
	return manager.ListAddress(maxIndex)
}

func (m WalletApi) NewMnemonicAndEntropyStore(password string, switchToIt bool) (*NewStoreResponse, error) {
	log.Info("NewMnemonicAndEntropyStore")
	mnemonic, em, err := m.wallet.NewMnemonicAndEntropyStore(password, switchToIt)
	if err != nil {
		return nil, err
	}

	return &NewStoreResponse{
		Mnemonic:    mnemonic,
		PrimaryAddr: em.GetPrimaryAddr(),
		Filename:    em.GetEntropyStoreFile(),
	}, nil
}

func (m WalletApi) SwitchEntropyStore(absFilename string) error {
	log.Info("SwitchEntropyStore")
	return m.wallet.SwitchEntropyStore(absFilename)
}

func (m WalletApi) RecoverEntropyStoreFromMnemonic(mnemonic string, newpassword string, switchToIt bool) (*NewStoreResponse, error) {
	log.Info("RecoverEntropyStoreFromMnemonic")
	em, e := m.wallet.RecoverEntropyStoreFromMnemonic(mnemonic, newpassword, switchToIt)
	if e != nil {
		return nil, e
	}
	return &NewStoreResponse{
		Mnemonic:    mnemonic,
		PrimaryAddr: em.GetPrimaryAddr(),
		Filename:    em.GetEntropyStoreFile(),
	}, nil
}

// THESE Are enttropy store api
func (m WalletApi) IsAddrUnlocked(addr types.Address) bool {
	log.Info("IsAddrUnlocked")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return false
	}
	return manager.IsAddrUnlocked(addr)
}

func (m WalletApi) IsUnlocked() bool {
	log.Info("IsUnlocked")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return false
	}
	return manager.IsUnlocked()
}

func (m WalletApi) ListAddress(addressNum uint32) ([]types.Address, error) {
	log.Info("ListAddress")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return nil, e
	}
	return manager.ListAddress(addressNum)
}

func (m WalletApi) Unlock(password string) error {
	log.Info("Unlock")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return e
	}
	err := manager.Unlock(password)
	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return newerr
	}
	return nil
}

func (m WalletApi) Lock() error {
	log.Info("Lock")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return e
	}
	manager.Lock()
	return nil
}

func (m WalletApi) FindAddrWithPassword(password string, addr types.Address) (index uint32, e error) {
	log.Info("FindAddrWithPassword")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return 0, e
	}
	_, u, e := manager.FindAddrWithPassword(password, addr)
	return u, e
}

func (m WalletApi) FindAddr(addr types.Address) (index uint32, e error) {
	log.Info("FindAddr")
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return 0, e
	}
	_, u, e := manager.FindAddr(addr)
	return u, e
}

func (m WalletApi) SignData(addr types.Address, hexMsg string) (*HexSignedTuple, error) {
	log.Info("SignData")

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return nil, err
	}
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return nil, e
	}

	signedData, pubkey, err := manager.SignData(addr, msgbytes)
	if err != nil {
		return nil, err
	}

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return &t, nil
}

func (m WalletApi) CreateTxWithPassphrase(params CreateTransferTxParms) error {
	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok {
		return ErrStrToBigInt
	}

	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: params.SelfAddr,
		ToAddress:      &params.ToAddr,
		TokenId:        &params.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Difficulty:     params.Difficulty,
		Data:           params.Data,
	}

	g, e := generator.NewGenerator(m.chain, nil, nil, &params.SelfAddr)
	if e != nil {
		return e
	}
	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		manager, e := m.wallet.GetEntropyStoreManager()
		if e != nil {
			return nil, nil, e
		}
		return manager.SignDataWithPassphrase(addr, params.Passphrase, data)
	})
	if e != nil {
		newerr, _ := TryMakeConcernedError(e)
		return newerr
	}
	if result.Err != nil {
		newerr, _ := TryMakeConcernedError(result.Err)
		return newerr
	}
	if len(result.BlockGenList) > 0 && result.BlockGenList[0] != nil {
		return m.pool.AddDirectAccountBlock(params.SelfAddr, result.BlockGenList[0])
	} else {
		return errors.New("generator gen an empty block")
	}

}

func (m WalletApi) SignDataWithPassphrase(addr types.Address, hexMsg string, password string) (*HexSignedTuple, error) {
	log.Info("SignDataWithPassphrase")

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return nil, err
	}
	manager, e := m.wallet.GetEntropyStoreManager()
	if e != nil {
		return nil, e
	}
	signedData, pubkey, err := manager.SignDataWithPassphrase(addr, password, msgbytes)
	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return nil, newerr
	}

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return &t, nil
}

func (m WalletApi) IsMayValidKeystoreFile(path string) IsMayValidKeystoreFileResponse {
	log.Info("IsValidKeystoreFile")
	b, addr, _ := entropystore.IsMayValidEntropystoreFile(path)
	if b && addr != nil {
		return IsMayValidKeystoreFileResponse{
			true, *addr,
		}
	}
	return IsMayValidKeystoreFileResponse{
		false, types.Address{},
	}
}

func (m WalletApi) GetDataDir() string {
	log.Info("GetDataDir")
	return m.wallet.GetDataDir()
}
