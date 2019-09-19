package api

import (
	"errors"

	"github.com/vitelabs/go-vite/common/types"
)

func (m WalletApi) GetEntropyFilesInStandardDir() ([]string, error) {
	return m.wallet.ListEntropyFilesInStandardDir()
}

func (m WalletApi) GetAllEntropyFiles() []string {
	return m.wallet.ListAllEntropyFiles()
}

func (m WalletApi) ExportMnemonic(entropyFile string, passphrase string) (string, error) {
	return m.wallet.ExtractMnemonic(entropyFile, passphrase)
}

func (m WalletApi) Unlock(entropyFile string, passphrase string) error {
	manager, e := m.wallet.GetEntropyStoreManager(entropyFile)
	if e != nil {
		return e
	}
	err := manager.Unlock(passphrase)
	if err != nil {
		return err
	}
	return nil
}

func (m WalletApi) Lock(entropyFile string) error {
	manager, e := m.wallet.GetEntropyStoreManager(entropyFile)
	if e != nil {
		return e
	}
	manager.Lock()
	return nil
}

func (m WalletApi) DeriveAddressesByIndexRange(entropyFile string, startIndex, endIndex uint32) ([]types.Address, error) {
	if startIndex > endIndex {
		return nil, errors.New("from value > to")
	}

	manager, e := m.wallet.GetEntropyStoreManager(entropyFile)
	if e != nil {
		return nil, e
	}
	return manager.ListAddress(startIndex, endIndex)
}

type CreateEntropyFileResponse struct {
	Mnemonics      string        `json:"mnemonics"`
	PrimaryAddress types.Address `json:"primaryAddress"`
	FilePath       string        `json:"filePath"`
}

func (m WalletApi) CreateEntropyFile(passphrase string) (*CreateEntropyFileResponse, error) {
	mnemonic, em, err := m.wallet.NewMnemonicAndEntropyStore(passphrase)
	if err != nil {
		return nil, err
	}

	return &CreateEntropyFileResponse{
		Mnemonics:      mnemonic,
		PrimaryAddress: em.GetPrimaryAddr(),
		FilePath:       em.GetEntropyStoreFile(),
	}, nil
}

func (m WalletApi) DeriveAddressByIndex(entropyFile string, index uint32) (*DeriveResult, error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyFile)
	if e != nil {
		return nil, e
	}
	path, key, e := manager.DeriveForIndexPath(index)
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

func (m WalletApi) DeriveAddressByPath(entropyFile string, bip44Path string) (*DeriveResult, error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyFile)
	if e != nil {
		return nil, e
	}
	_, key, e := manager.DeriveForFullPath(bip44Path)
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
		Bip44Path:  bip44Path,
		Address:    *address,
		PrivateKey: privateKey,
	}, nil
}

func (m WalletApi) RecoverEntropyFile(mnemonics string, passphrase string) (*CreateEntropyFileResponse, error) {
	em, e := m.wallet.RecoverEntropyStoreFromMnemonic(mnemonics, passphrase)
	if e != nil {
		return nil, e
	}
	return &CreateEntropyFileResponse{
		Mnemonics:      mnemonics,
		PrimaryAddress: em.GetPrimaryAddr(),
		FilePath:       em.GetEntropyStoreFile(),
	}, nil
}

func (m WalletApi) IsUnlocked(entropyFile string) bool {
	return m.wallet.IsUnlocked(entropyFile)
}

type FindAddrResponse struct {
	EntropyFile string `json:"entropyFile"`
	Index       uint32 `json:"index"`
}

func (m WalletApi) FindAddressInEntropyFile(entropyFile string, address types.Address) (findResult *FindAddrResponse, e error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyFile)
	if e != nil {
		return nil, e
	}
	_, index, e := manager.FindAddr(address)
	return &FindAddrResponse{
		EntropyFile: manager.GetEntropyStoreFile(),
		Index:       index,
	}, nil
}

func (m WalletApi) FindAddress(address types.Address) (findResult *FindAddrResponse, e error) {
	path, _, index, e := m.wallet.GlobalFindAddr(address)
	if e != nil {
		return nil, e
	}
	return &FindAddrResponse{
		EntropyFile: path,
		Index:       index,
	}, nil
}

type CreateTransactionParms struct {
	EntropyFile *string           `json:"entropyFile,omitempty"`
	Address     types.Address     `json:"address"`
	ToAddress   types.Address     `json:"toAddress"`
	TokenId     types.TokenTypeId `json:"tokenId"`
	Passphrase  string            `json:"passphrase"`
	Amount      string            `json:"amount"`
	Data        []byte            `json:"data,omitempty"`
	Difficulty  *string           `json:"difficulty,omitempty"`
}

func (m WalletApi) CreateTransaction(params CreateTransactionParms) (*types.Hash, error) {
	return m.CreateTxWithPassphrase(CreateTransferTxParms{
		EntropystoreFile: params.EntropyFile,
		SelfAddr:         params.Address,
		ToAddr:           params.ToAddress,
		TokenTypeId:      params.TokenId,
		Passphrase:       params.Passphrase,
		Amount:           params.Amount,
		Data:             params.Data,
		Difficulty:       params.Difficulty,
	})
}
