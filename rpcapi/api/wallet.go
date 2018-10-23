package api

import (
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet/seedstore"
)

type HexSignedTuple struct {
	Message    string `json:"message"`
	SignedData string `json:"signedData"`
	Pubkey     string `json:"pubkey"`
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
		km:    vite.WalletManager().SeedStoreManagers,
		chain: vite.Chain(),
		pool:  vite.Pool(),
	}
}

type WalletApi struct {
	km    *seedstore.Manager
	chain chain.Chain
	pool  pool.Writer
}

func (m WalletApi) String() string {
	return "WalletApi"
}

func (m WalletApi) ListAddress() []types.Address {
	log.Info("ListAddress")
	return m.km.Addresses()

}

func (m *WalletApi) NewAddress(passphrase string) (types.Address, error) {
	log.Info("NewAddress")
	key, err := m.km.StoreNewKey(passphrase)
	key.PrivateKey.Clear()
	return key.Address, err
}

func (m WalletApi) Status() map[types.Address]string {
	log.Info("Status")
	s, _ := m.km.Status()
	return s
}

func (m *WalletApi) UnlockAddress(addr types.Address, password string, duration *uint64) (bool, error) {
	log.Info("UnLock")

	const max = uint64(time.Duration(math.MaxInt64) / time.Second)
	var d time.Duration
	if duration == nil {
		d = 300 * time.Second
	} else if *duration > max {
		return false, errors.New("unlock duration too large")
	} else {
		d = time.Duration(*duration) * time.Second
	}

	err := m.km.UnlockAddress(addr, password, d)

	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return false, newerr
	}
	return true, nil
}

func (m *WalletApi) LockAddress(addr types.Address) error {
	log.Info("Lock")
	err := m.km.LockAddress(addr)
	return err
}

func (m *WalletApi) ReloadAndFixAddressFile() error {
	log.Info("ReloadAndFixAddressFile")
	m.km.ReloadAndFixAddressFile()
	return nil
}

func (m *WalletApi) ImportPriv(privkey string, newpassword string) (types.Address, error) {
	log.Info("ImportPriv")

	key, err := m.km.ImportPriv(privkey, newpassword)
	if err != nil {
		return types.Address{}, err
	}
	key.PrivateKey.Clear()
	newpassword = ""
	privkey = ""
	return key.Address, nil
}

func (m *WalletApi) ExportPriv(address types.Address, password string) (string, error) {
	log.Info("ExportPriv")
	s, err := m.km.ExportPriv(address.Hex(), password)

	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return "", newerr

	}
	return s, nil
}

func (m *WalletApi) SignData(addr types.Address, hexMsg string) (HexSignedTuple, error) {
	log.Info("SignData")

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return HexSignedTuple{}, err
	}
	signedData, pubkey, err := m.km.SignData(addr, msgbytes)
	if err != nil {
		return HexSignedTuple{}, err
	}

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return t, nil
}

func (m *WalletApi) CreateTxWithPassphrase(params CreateTransferTxParms) error {
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
		return m.km.SignDataWithPassphrase(addr, params.Passphrase, data)
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

func (m *WalletApi) SignDataWithPassphrase(addr types.Address, hexMsg string, password string) (HexSignedTuple, error) {
	log.Info("SignDataWithPassphrase")

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return HexSignedTuple{}, err
	}
	signedData, pubkey, err := m.km.SignDataWithPassphrase(addr, password, msgbytes)
	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return HexSignedTuple{}, newerr
	}

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return t, nil
}

func (m *WalletApi) IsMayValidKeystoreFile(path string) IsMayValidKeystoreFileResponse {
	log.Info("IsValidKeystoreFile")
	b, addr, _ := seedstore.IsMayValidSeedstoreFile(path)
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
	return m.km.KeyStoreDir
}
