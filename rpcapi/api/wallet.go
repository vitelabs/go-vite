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

type FindAddrResult struct {
	EntropyStoreFile string `json:"entropyStoreFile"`
	Index            uint32 `json:"index"`
}

type DeriveResult struct {
	Bip44Path  string        `json:"bip44Path"`
	Address    types.Address `json:"address"`
	PrivateKey []byte        `json:"privateKey"`
}

type SignDataParams struct {
	Address          types.Address `json:"address"`
	EntropystoreFile string        `json:"entropystoreFile"`
	Bip44Index       *uint32       `json:"bip44Index,omitempty"`
	ExtensionWord    *string       `json:"extensionWord,omitempty"`
	Passphrase       *string       `json:"passphrase, omitempty"`
	HexMsg           string        `json:"hexMsg"`
}

type CreateTransferTxParams struct {
	EntropystoreFile string            `json:"entropystoreFile"`
	Bip44Index       *uint32           `json:"bip44Index,omitempty"`
	ExtensionWord    *string           `json:"extensionWord,omitempty"`
	SelfAddr         types.Address     `json:"selfAddr"`
	ToAddr           types.Address     `json:"toAddr"`
	TokenTypeId      types.TokenTypeId `json:"tokenTypeId"`
	Passphrase       string            `json:"passphrase"`
	Amount           string            `json:"amount"`
	Data             []byte            `json:"data,omitempty"`
	Difficulty       *string           `json:"difficulty,omitempty"`
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

func (m WalletApi) ListAllEntropyFiles() []string {
	return m.wallet.ListAllEntropyFiles()
}

func (m WalletApi) ListEntropyFilesInStandardDir() ([]string, error) {
	return m.wallet.ListEntropyFilesInStandardDir()
}

func (m WalletApi) ListEntropyStoreAddresses(entropyStore string, from, to uint32, extensionWord *string) ([]types.Address, error) {
	if from > to {
		return nil, errors.New("from value > to")
	}

	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	return manager.ListAddress(from, to, extensionWord)
}

func (m WalletApi) NewMnemonicAndEntropyStore(passphrase string, language, extensionWord *string, mnemonicSize *int) (*NewStoreResponse, error) {
	lang := ""
	if language != nil {
		lang = *language
	}
	mnemonic, em, err := m.wallet.NewMnemonicAndEntropyStore(lang, passphrase, extensionWord, mnemonicSize)
	if err != nil {
		return nil, err
	}

	return &NewStoreResponse{
		Mnemonic:    mnemonic,
		PrimaryAddr: em.GetPrimaryAddr(),
		Filename:    em.GetEntropyStoreFile(),
	}, nil
}
func (m WalletApi) DeriveByFullPath(entropyStore string, fullpath string, extensionWord *string) (*DeriveResult, error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	_, key, e := manager.DeriveForFullPath(fullpath, extensionWord)
	if e != nil {
		return nil, e
	}

	address, err := key.Address()
	if err != nil {
		return nil, err
	}

	privateKey, err := key.PrivateKey()
	if err != nil {
		return nil, err
	}

	return &DeriveResult{
		Bip44Path:  fullpath,
		Address:    *address,
		PrivateKey: privateKey,
	}, nil
}

func (m WalletApi) DeriveByIndex(entropyStore string, index uint32, extensionWord *string) (*DeriveResult, error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	path, key, e := manager.DeriveForIndexPath(index, extensionWord)
	if e != nil {
		return nil, e
	}

	address, err := key.Address()
	if err != nil {
		return nil, err
	}

	privateKey, err := key.PrivateKey()
	if err != nil {
		return nil, err
	}

	return &DeriveResult{
		Bip44Path:  path,
		Address:    *address,
		PrivateKey: privateKey,
	}, nil
}

func (m WalletApi) RecoverEntropyStoreFromMnemonic(mnemonic string, newPassphrase string, language, extensionWord *string) (*NewStoreResponse, error) {

	lang := ""
	if language != nil {
		lang = *language
	}
	em, e := m.wallet.RecoverEntropyStoreFromMnemonic(mnemonic, lang, newPassphrase, extensionWord)
	if e != nil {
		return nil, e
	}
	return &NewStoreResponse{
		Mnemonic:    mnemonic,
		PrimaryAddr: em.GetPrimaryAddr(),
		Filename:    em.GetEntropyStoreFile(),
	}, nil
}

func (m WalletApi) IsAddrUnlocked(entropyStore string, addr types.Address, extensionWord *string) bool {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return false
	}
	return manager.IsAddrUnlocked(addr, extensionWord)
}

func (m WalletApi) IsUnlocked(entropyStore string) bool {
	return m.wallet.IsUnlocked(entropyStore)
}

func (m WalletApi) RefreshCache() {
	m.wallet.RefreshCache()
}

func (m WalletApi) Unlock(entropyStore string, passphrase string) error {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return e
	}
	err := manager.Unlock(passphrase)
	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return newerr
	}
	return nil
}

func (m WalletApi) Lock(entropyStore string) error {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return e
	}
	manager.Lock()
	return nil
}

func (m WalletApi) FindAddrWithPassphrase(entropyStore string, passphrase string, addr types.Address, extensionWord *string) (findResult *FindAddrResult, e error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	_, index, e := manager.FindAddrWithPassphrase(passphrase, addr, extensionWord)
	return &FindAddrResult{
		EntropyStoreFile: manager.GetEntropyStoreFile(),
		Index:            index,
	}, nil

}

func (m WalletApi) FindAddr(entropyStore string, addr types.Address, extensionWord *string) (findResult *FindAddrResult, e error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	_, index, e := manager.FindAddr(addr, extensionWord)
	return &FindAddrResult{
		EntropyStoreFile: manager.GetEntropyStoreFile(),
		Index:            index,
	}, nil
}

func (m WalletApi) AddEntropyStore(filename string) error {
	return m.wallet.AddEntropyStore(filename)
}

func (m WalletApi) CreateTxWithPassphrase(params CreateTransferTxParams) error {
	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok {
		return ErrStrToBigInt
	}
	var difficulty *big.Int = nil
	if params.Difficulty != nil {
		difficulty, ok = new(big.Int).SetString(*params.Difficulty, 10)
		if !ok {
			return ErrStrToBigInt
		}
	}

	msg := &generator.IncomingMessage{
		BlockType:      ledger.BlockTypeSendCall,
		AccountAddress: params.SelfAddr,
		ToAddress:      &params.ToAddr,
		TokenId:        &params.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Difficulty:     difficulty,
		Data:           params.Data,
	}

	_, fitestSnapshotBlockHash, err := generator.GetFittestGeneratorSnapshotHash(m.chain, &msg.AccountAddress, nil, true)
	if err != nil {
		return err
	}
	g, e := generator.NewGenerator(m.chain, fitestSnapshotBlockHash, nil, &params.SelfAddr)
	if e != nil {
		return e
	}

	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		manager, e := m.wallet.GetEntropyStoreManager(params.EntropystoreFile)
		if e != nil {
			return nil, nil, e
		}
		return manager.SignDataWithPassphrase(addr, params.Passphrase, data, params.Bip44Index, params.ExtensionWord)
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

func (m WalletApi) SignData(params *SignDataParams) (*HexSignedTuple, error) {
	if params == nil {
		return nil, errors.New("empty params")
	}

	msgbytes, err := hex.DecodeString(params.HexMsg)
	if err != nil {
		return nil, err
	}

	em, err := m.wallet.GetEntropyStoreManager(params.EntropystoreFile)
	if err != nil {
		return nil, err
	}

	if params.Passphrase != nil {
		signedData, pubkey, e := em.SignDataWithPassphrase(params.Address, *params.Passphrase, msgbytes, params.Bip44Index, params.ExtensionWord)
		if e != nil {
			return nil, e
		}
		return &HexSignedTuple{
			Message:    params.HexMsg,
			Pubkey:     hex.EncodeToString(pubkey),
			SignedData: hex.EncodeToString(signedData),
		}, nil
	} else {
		signedData, pubkey, e := em.SignData(params.Address, msgbytes, params.Bip44Index, params.ExtensionWord)
		if e != nil {
			return nil, e
		}
		return &HexSignedTuple{
			Message:    params.HexMsg,
			Pubkey:     hex.EncodeToString(pubkey),
			SignedData: hex.EncodeToString(signedData),
		}, nil
	}

}

func (m WalletApi) IsMayValidKeystoreFile(path string) IsMayValidKeystoreFileResponse {
	b, json, _ := entropystore.IsMayValidEntropystoreFile(path)
	if b {
		return IsMayValidKeystoreFileResponse{
			true, *json.PrimaryAddress,
		}
	}
	return IsMayValidKeystoreFileResponse{
		false, types.Address{},
	}
}

func (m WalletApi) GetDataDir() string {
	return m.wallet.GetDataDir()
}
