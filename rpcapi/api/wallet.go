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
	Index      uint32        `json:"index"`
	Address    types.Address `json:"address"`
	PrivateKey []byte        `json:"privateKey"`
}

type CreateTransferTxParms struct {
	EntropystoreFile *string           `json:"entropystoreFile,omitempty"`
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

func (m WalletApi) ListEntropyStoreAddresses(entropyStore string, from, to uint32) ([]types.Address, error) {
	if from > to {
		return nil, errors.New("from value > to")
	}

	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	return manager.ListAddress(from, to)
}

func (m WalletApi) NewMnemonicAndEntropyStore(passphrase string) (*NewStoreResponse, error) {
	mnemonic, em, err := m.wallet.NewMnemonicAndEntropyStore(passphrase)
	if err != nil {
		return nil, err
	}

	return &NewStoreResponse{
		Mnemonic:    mnemonic,
		PrimaryAddr: em.GetPrimaryAddr(),
		Filename:    em.GetEntropyStoreFile(),
	}, nil
}

func (m WalletApi) DeriveForIndexPath(entropyStore string, index uint32) (*DeriveResult, error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	_, key, e := manager.DeriveForIndexPath(index)
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
		Index:      index,
		Address:    *address,
		PrivateKey: privateKey,
	}, nil
}

func (m WalletApi) RecoverEntropyStoreFromMnemonic(mnemonic string, newPassphrase string) (*NewStoreResponse, error) {
	em, e := m.wallet.RecoverEntropyStoreFromMnemonic(mnemonic, newPassphrase)
	if e != nil {
		return nil, e
	}
	return &NewStoreResponse{
		Mnemonic:    mnemonic,
		PrimaryAddr: em.GetPrimaryAddr(),
		Filename:    em.GetEntropyStoreFile(),
	}, nil
}

func (m WalletApi) GlobalCheckAddrUnlocked(addr types.Address) bool {
	return m.wallet.GlobalCheckAddrUnlock(addr)
}

func (m WalletApi) IsAddrUnlocked(entropyStore string, addr types.Address) bool {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return false
	}
	return manager.IsAddrUnlocked(addr)
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

func (m WalletApi) FindAddrWithPassphrase(entropyStore string, passphrase string, addr types.Address) (findResult *FindAddrResult, e error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	_, index, e := manager.FindAddrWithPassword(passphrase, addr)
	return &FindAddrResult{
		EntropyStoreFile: manager.GetEntropyStoreFile(),
		Index:            index,
	}, nil

}

func (m WalletApi) FindAddr(entropyStore string, addr types.Address) (findResult *FindAddrResult, e error) {
	manager, e := m.wallet.GetEntropyStoreManager(entropyStore)
	if e != nil {
		return nil, e
	}
	_, index, e := manager.FindAddr(addr)
	return &FindAddrResult{
		EntropyStoreFile: manager.GetEntropyStoreFile(),
		Index:            index,
	}, nil
}

func (m WalletApi) GlobalFindAddr(addr types.Address) (findResult *FindAddrResult, e error) {
	path, _, index, e := m.wallet.GlobalFindAddr(addr)
	if e != nil {
		return nil, e
	}
	return &FindAddrResult{
		EntropyStoreFile: path,
		Index:            index,
	}, nil
}

func (m WalletApi) GlobalFindAddrWithPassphrase(addr types.Address, passphrase string) (findResult *FindAddrResult, e error) {
	path, _, index, e := m.wallet.GlobalFindAddrWithPassphrase(addr, passphrase)
	if e != nil {
		return nil, e
	}
	return &FindAddrResult{
		EntropyStoreFile: path,
		Index:            index,
	}, nil
}

func (m WalletApi) AddEntropyStore(filename string) error {
	return m.wallet.AddEntropyStore(filename)
}

func (m WalletApi) SignData(addr types.Address, hexMsg string) (*HexSignedTuple, error) {

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return nil, err
	}
	_, key, _, e := m.wallet.GlobalFindAddr(addr)
	if e != nil {
		return nil, e
	}

	signedData, pubkey, err := key.SignData(msgbytes)
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

	fitestSnapshotBlockHash, err := generator.GetFitestGeneratorSnapshotHash(m.chain, nil)
	if err != nil {
		return err
	}
	g, e := generator.NewGenerator(m.chain, fitestSnapshotBlockHash, nil, &params.SelfAddr)
	if e != nil {
		return e
	}

	result, e := g.GenerateWithMessage(msg, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		if params.EntropystoreFile != nil {
			manager, e := m.wallet.GetEntropyStoreManager(*params.EntropystoreFile)
			if e != nil {
				return nil, nil, e
			}
			return manager.SignDataWithPassphrase(addr, params.Passphrase, data)
		}

		_, key, _, e := m.wallet.GlobalFindAddrWithPassphrase(addr, params.Passphrase)
		if e != nil {
			return nil, nil, e
		}
		return key.SignData(data)
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

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return nil, err
	}
	_, key, _, e := m.wallet.GlobalFindAddrWithPassphrase(addr, password)
	if e != nil {
		return nil, e
	}
	signedData, pubkey, err := key.SignData(msgbytes)
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
	return m.wallet.GetDataDir()
}
