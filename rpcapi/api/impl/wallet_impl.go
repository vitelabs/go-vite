package impl

import (
	"encoding/hex"
	"errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/rpcapi/api"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"time"
)

func NewWalletApi(vite *vite.Vite) api.WalletApi {
	return &WalletApiImpl{km: vite.WalletManager().KeystoreManager}
}

type WalletApiImpl struct {
	km *keystore.Manager
}

func (m WalletApiImpl) String() string {
	return "WalletApiImpl"
}

func (m WalletApiImpl) ListAddress() []types.Address {
	log.Info("ListAddress")
	return m.km.Addresses()

}

func (m *WalletApiImpl) NewAddress(passphrase string) (types.Address, error) {
	log.Info("NewAddress")
	key, err := m.km.StoreNewKey(passphrase)
	key.PrivateKey.Clear()
	return key.Address, err
}

func (m WalletApiImpl) Status() map[types.Address]string {
	log.Info("Status")
	s, _ := m.km.Status()
	return s
}

func (m *WalletApiImpl) UnLockAddress(addr types.Address, password string, duration *uint64) (bool, error) {
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
		newerr, _ := api.TryMakeConcernedError(err)
		return false, newerr
	}
	return true, nil
}

func (m *WalletApiImpl) LockAddress(addr types.Address) error {
	log.Info("Lock")
	err := m.km.Lock(addr)
	return err
}

func (m *WalletApiImpl) ReloadAndFixAddressFile() error {
	log.Info("ReloadAndFixAddressFile")
	m.km.ReloadAndFixAddressFile()
	return nil
}

func (m *WalletApiImpl) ImportPriv(privkey string, newpassword string) (types.Address, error) {
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

func (m *WalletApiImpl) ExportPriv(address types.Address, password string) (string, error) {
	log.Info("ExportPriv")
	s, err := m.km.ExportPriv(address.Hex(), password)

	if err != nil {
		newerr, _ := api.TryMakeConcernedError(err)
		return "", newerr

	}
	return s, nil
}

func (m *WalletApiImpl) SignData(addr types.Address, hexMsg string) (api.HexSignedTuple, error) {
	log.Info("SignData")

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return api.HexSignedTuple{}, err
	}
	signedData, pubkey, err := m.km.SignData(addr, msgbytes)
	if err != nil {
		return api.HexSignedTuple{}, err
	}

	t := api.HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return t, nil
}

func (m *WalletApiImpl) SignDataWithPassphrase(addr types.Address, hexMsg string, password string) (api.HexSignedTuple, error) {
	log.Info("SignDataWithPassphrase")

	msgbytes, err := hex.DecodeString(hexMsg)
	if err != nil {
		return api.HexSignedTuple{}, err
	}
	signedData, pubkey, err := m.km.SignDataWithPassphrase(addr, password, msgbytes)
	if err != nil {
		newerr, _ := api.TryMakeConcernedError(err)
		return api.HexSignedTuple{}, newerr
	}

	t := api.HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return t, nil
}

func (m *WalletApiImpl) IsMayValidKeystoreFile(path string) types.Address {
	log.Info("IsValidKeystoreFile")
	b, addr, _ := keystore.IsMayValidKeystoreFile(path)
	if b && addr != nil {
		return *addr
	}
	return types.Address{}
}

func (m WalletApiImpl) GetDataDir() string {
	log.Info("GetDataDir")
	return m.km.KeyStoreDir
}
