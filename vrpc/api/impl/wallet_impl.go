package impl

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vrpc/api"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"strconv"
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

func (m WalletApiImpl) ListAddress(v interface{}, reply *string) error {
	log.Info("ListAddress")
	as := m.km.Addresses()
	s := make([]string, len(as))
	for i, v := range as {
		s[i] = v.String()
	}
	json, err := json.Marshal(s)
	if err != nil {
		return err
	}
	*reply = string(json)

	return nil
}

func (m WalletApiImpl) GetDataDir(v interface{}, reply *string) error {
	log.Info("GetDataDir")

	*reply = m.km.KeyStoreDir

	return nil
}

func (m *WalletApiImpl) NewAddress(passphrase []string, reply *string) error {
	log.Info("NewAddress")
	if len(passphrase) != 1 {
		return fmt.Errorf("wrong params len %v. you should pass [0] passphrase, ", len(passphrase))
	}
	key, err := m.km.StoreNewKey(passphrase[0])
	if err != nil {
		return err
	}
	key.PrivateKey.Clear()
	*reply = key.Address.Hex()
	return nil
}

func (m WalletApiImpl) Status(v interface{}, reply *string) error {
	log.Info("Status")
	s, err := m.km.Status()
	if err != nil {
		return err
	}
	stringMap := make(map[string]string)

	for k, v := range s {
		stringMap[k.String()] = v
	}
	return easyJsonReturn(stringMap, reply)
}

func (m *WalletApiImpl) UnLock(unlockParams []string, reply *string) error {
	log.Info("UnLock")
	if len(unlockParams) != 3 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddress, "+
			"[1] address releated passphrase passphrase, [3] unlocktime", len(unlockParams))
	}
	addr, err := types.HexToAddress(unlockParams[0])
	if err != nil {
		return err
	}

	passphrase := unlockParams[1]
	unlocktime, err := strconv.Atoi(unlockParams[2])
	if err != nil {
		return err
	}

	err = m.km.Unlock(addr, passphrase, time.Second*time.Duration(unlocktime))
	if err != nil {
		return tryMakeConcernedError(err, reply)
	}

	*reply = "success"
	return nil
}

func (m *WalletApiImpl) Lock(hexaddr []string, reply *string) error {
	log.Info("Lock")
	if len(hexaddr) != 1 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddr, ", len(hexaddr))
	}
	addr, err := types.HexToAddress(hexaddr[0])
	if err != nil {
		return err
	}
	m.km.Lock(addr)
	return nil
}

func (m *WalletApiImpl) SignData(signDataParams []string, reply *string) error {
	log.Info("SignData")
	if len(signDataParams) != 2 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddress,"+
			" [1]message ", len(signDataParams))
	}
	addr, err := types.HexToAddress(signDataParams[0])
	if err != nil {
		return err
	}
	hexMsg := signDataParams[1]
	msgbytes, err := hex.DecodeString(signDataParams[1])
	if err != nil {
		return fmt.Errorf("wrong hex message %v", err)
	}
	signedData, pubkey, err := m.km.SignData(addr, msgbytes)
	if err != nil {
		return err
	}

	t := api.HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return easyJsonReturn(t, reply)
}

func (m *WalletApiImpl) SignDataWithPassphrase(signDataParams []string, reply *string) error {
	log.Info("SignDataWithPassphrase")
	if len(signDataParams) != 3 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddress,"+
			" [1]message, [2] address releated passphrase", len(signDataParams))
	}
	addr, err := types.HexToAddress(signDataParams[0])
	if err != nil {
		return err
	}
	hexMsg := signDataParams[1]
	passphrase := signDataParams[2]
	msgbytes, err := hex.DecodeString(signDataParams[2])
	if err != nil {
		return fmt.Errorf("wrong hex message %v", err)
	}
	signedData, pubkey, err := m.km.SignDataWithPassphrase(addr, passphrase, msgbytes)

	t := api.HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return easyJsonReturn(t, reply)
}

func (m *WalletApiImpl) ReloadAndFixAddressFile(v interface{}, reply *string) error {
	log.Info("ReloadAndFixAddressFile")
	m.km.ReloadAndFixAddressFile()
	*reply = "success"
	return nil
}

func (m *WalletApiImpl) ImportPriv(hexkeypair []string, reply *string) error {
	log.Info("ImportPriv")
	if len(hexkeypair) != 2 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexhexprikey,"+
			" [1]newPass", len(hexkeypair))
	}
	hexprikey := hexkeypair[0]
	newPass := hexkeypair[1]
	key, err := m.km.ImportPriv(hexprikey, newPass)
	if err != nil {
		return err
	}
	key.PrivateKey.Clear()
	*reply = key.Address.Hex()
	return nil
}

func (m *WalletApiImpl) ExportPriv(extractPair []string, reply *string) error {
	log.Info("ExportPriv")
	if len(extractPair) != 2 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddr,"+
			" [1]address releated passphrase", len(extractPair))
	}
	addr, err := types.HexToAddress(extractPair[0])
	if err != nil {
		return err
	}
	pass := extractPair[1]
	s, err := m.km.ExportPriv(addr.Hex(), pass)
	*reply = s
	return nil
}

func (m *WalletApiImpl) IsMayValidKeystoreFile(path []string, reply *string) error {
	log.Info("IsValidKeystoreFile")
	if len(path) != 1 {
		return fmt.Errorf("wrong params len %v. you should pass [0] path", len(path))
	}
	b, addr, _ := keystore.IsMayValidKeystoreFile(path[0])
	if b && addr != nil {
		*reply = addr.String()
	} else {
		*reply = ""
	}
	return nil
}
