package api

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"time"
)

type HexSignedTuple struct {
	Message    string `json:"message"`
	SignedData string `json:"signedData"`
	Pubkey     string `json:"pubkey"`
}

type IsMayValidKeystoreFileResponse struct {
	Maybe      bool
	MayAddress types.Address
}

func NewWalletApi(vite *vite.Vite) *WalletApi {
	return &WalletApi{km: vite.WalletManager().KeystoreManager}
}

type WalletApi struct {
	km *keystore.Manager
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

	err := m.km.Unlock(addr, password, d)

	if err != nil {
		newerr, _ := TryMakeConcernedError(err)
		return false, newerr
	}
	return true, nil
}

func (m *WalletApi) LockAddress(addr types.Address) error {
	log.Info("Lock")
	err := m.km.Lock(addr)
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
	b, addr, _ := keystore.IsMayValidKeystoreFile(path)
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
