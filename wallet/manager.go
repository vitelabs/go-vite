package wallet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"strconv"
	"time"
)

type Manager struct {
	KeystoreManager *keystore.Manager
	JsonApi         JsonApi
}

func NewManager(walletdir string) *Manager {
	km := keystore.NewManager(walletdir)
	km.Init()
	return &Manager{
		KeystoreManager: km,
		JsonApi:         &walletApiImpl{KeystoreManager: km},
	}
}

type walletApiImpl struct {
	KeystoreManager *keystore.Manager
}

func (m walletApiImpl) ListAddress(v interface{}, reply *string) error {
	log.Debug("ListAddress")
	*reply = types.Addresses(m.KeystoreManager.Addresses()).String()
	return nil
}

func (m *walletApiImpl) NewAddress(passphrase []string, reply *string) error {
	log.Debug("NewAddress")
	if len(passphrase) != 1 {
		return fmt.Errorf("wrong params len %v. you should pass [0] passphrase, ", len(passphrase))
	}
	key, err := m.KeystoreManager.StoreNewKey(passphrase[0])
	if err != nil {
		return err
	}
	key.PrivateKey.Clear()
	*reply = key.Address.Hex()
	return nil
}

func (m walletApiImpl) Status(v interface{}, reply *string) error {
	log.Debug("Status")
	s, err := m.KeystoreManager.Status()
	if err != nil {
		return err
	}
	*reply = s
	return nil
}

func (m *walletApiImpl) UnLock(unlockParams []string, reply *string) error {
	log.Debug("UnLock")
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

	err = m.KeystoreManager.Unlock(addr, passphrase, time.Second*time.Duration(unlocktime))
	if err != nil {
		return err
	}

	*reply = "success"
	return nil
}

func (m *walletApiImpl) Lock(hexaddr []string, reply *string) error {
	log.Debug("Lock")
	if len(hexaddr) != 1 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddr, ", len(hexaddr))
	}
	addr, err := types.HexToAddress(hexaddr[0])
	if err != nil {
		return err
	}
	m.KeystoreManager.Lock(addr)
	return nil
}
func (m *walletApiImpl) SignData(signDataParams []string, reply *string) error {
	log.Debug("SignData")
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
	signedData, pubkey, err := m.KeystoreManager.SignData(addr, msgbytes)
	if err != nil {
		return err
	}

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	j, err := json.Marshal(t)
	if err != nil {
		return err
	}
	*reply = string(j)
	return nil
}

func (m *walletApiImpl) SignDataWithPassphrase(signDataParams []string, reply *string) error {
	log.Debug("SignDataWithPassphrase")
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
	println("passphrase " + passphrase)
	msgbytes, err := hex.DecodeString(signDataParams[2])
	if err != nil {
		return fmt.Errorf("wrong hex message %v", err)
	}
	signedData, pubkey, err := m.KeystoreManager.SignDataWithPassphrase(addr, passphrase, msgbytes)

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	j, err := json.Marshal(t)
	if err != nil {
		return err
	}
	*reply = string(j)
	return nil
}

func (m *walletApiImpl) ReloadAndFixAddressFile(v interface{}, reply *string) error {
	log.Debug("ReloadAndFixAddressFile")
	m.KeystoreManager.ReloadAndFixAddressFile()
	*reply = "success"
	return nil
}

func (m *walletApiImpl) ImportPriv(hexkeypair []string, reply *string) error {
	log.Debug("ImportPriv")
	if len(hexkeypair) != 2 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexhexprikey,"+
			" [1]newPass", len(hexkeypair))
	}
	hexprikey := hexkeypair[0]
	newPass := hexkeypair[1]
	key, err := m.KeystoreManager.ImportPriv(hexprikey, newPass)
	if err != nil {
		return err
	}
	key.PrivateKey.Clear()
	*reply = key.Address.Hex()
	return nil
}

func (m *walletApiImpl) ExportPriv(extractPair []string, reply *string) error {
	log.Debug("ExportPriv")
	if len(extractPair) != 2 {
		return fmt.Errorf("wrong params len %v. you should pass [0] hexaddr,"+
			" [1]address releated passphrase", len(extractPair))
	}
	addr, err := types.HexToAddress(extractPair[0])
	if err != nil {
		return err
	}
	pass := extractPair[1]
	s, err := m.KeystoreManager.ExportPriv(addr.Hex(), pass)
	*reply = s
	return nil
}
