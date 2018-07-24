package apis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log"
	"github.com/vitelabs/go-vite/wallet/keystore"
	"strconv"
	"time"
	"github.com/vitelabs/go-vite/vite"
)

type HexSignedTuple struct {
	Message    string `json:"Message"`
	SignedData string `json:"SignedData"`
	Pubkey     string `json:"Pubkey"`
}

type WalletApi interface {
	// list all address in keystore file, the reply string will split addresses with \n
	// example:
	// ["vite_15dac990004ae1cbf1af913091092f7c45205b88d905881d97",
	// "vite_48c5a659e37a9a462b96ff49ef3d30f10137c600417ce05cda"]
	ListAddress(v interface{}, reply *string) error

	// it will create a address and store in a dir
	// passphrase len must be 1, the reply string is hex-formed address
	NewAddress(passphrase []string, reply *string) error

	// return value is all the address with  Locked  or Unlocked state
	// example:
	// {"vite_15dac990004ae1cbf1af913091092f7c45205b88d905881d97":"Locked",
	// "vite_48c5a659e37a9a462b96ff49ef3d30f10137c600417ce05cda":"Unlocked"}
	Status(v interface{}, reply *string) error

	// hexAddress := unlockParams[0] passphrase := unlockParams[1] unlocktime := unlockParams[2]
	// unlocks the given address with the passphrase. The account stays unlocked for the duration of timeout (seconds)
	// if the timeout is <0 we will keep the unlock state until the program exit.
	UnLock(unlockParams []string, reply *string) error

	// you must pass an address into lockParams , if no error happened means lock success
	Lock(lockParams []string, reply *string) error

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1]
	// if the given address has not been unlocked it will return an ErrUnlocked
	SignData(signDataParams []string, reply *string) error

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1] passphrase := signDataParams[2]
	SignDataWithPassphrase(signDataParams []string, reply *string) error

	// if a keystore file name is changed it will read the file content
	// if the  content is legal the function will fix the filename into hex-formed address
	ReloadAndFixAddressFile(v interface{}, reply *string) error

	// hexprikey := hexkeypair[0] newPass := hexkeypair[1] the reply string is hex-formed address
	ImportPriv(hexkeypair []string, reply *string) error

	// hexaddr := extractPair[0] pass := extractPair[1] the return value is prikey in hex form
	ExportPriv(extractPair []string, reply *string) error

	// it reply string is false it must not be a valid keystore file
	// else only means that might be a keystore file
	IsMayValidKeystoreFile(path []string, reply *string) error

	// Get dir
	GetDataDir(v interface{}, reply *string) error
}

func NewWalletApi(vite *vite.Vite) WalletApi {
	return &WalletApiImpl{km: vite.WalletManager().KeystoreManager}
}

type WalletApiImpl struct {
	km *keystore.Manager
}

func (m WalletApiImpl) String() string {
	return "WalletApiImpl"
}

func (m WalletApiImpl) ListAddress(v interface{}, reply *string) error {
	log.Debug("ListAddress")
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
	log.Debug("GetDataDir")

	*reply = m.km.KeyStoreDir

	return nil
}

func (m *WalletApiImpl) NewAddress(passphrase []string, reply *string) error {
	log.Debug("NewAddress")
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
	log.Debug("Status")
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

	err = m.km.Unlock(addr, passphrase, time.Second*time.Duration(unlocktime))
	if err != nil {
		return err
	}

	*reply = "success"
	return nil
}

func (m *WalletApiImpl) Lock(hexaddr []string, reply *string) error {
	log.Debug("Lock")
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
	signedData, pubkey, err := m.km.SignData(addr, msgbytes)
	if err != nil {
		return err
	}

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return easyJsonReturn(t, reply)
}

func (m *WalletApiImpl) SignDataWithPassphrase(signDataParams []string, reply *string) error {
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
	signedData, pubkey, err := m.km.SignDataWithPassphrase(addr, passphrase, msgbytes)

	t := HexSignedTuple{
		Message:    hexMsg,
		Pubkey:     hex.EncodeToString(pubkey),
		SignedData: hex.EncodeToString(signedData),
	}

	return easyJsonReturn(t, reply)
}

func (m *WalletApiImpl) ReloadAndFixAddressFile(v interface{}, reply *string) error {
	log.Debug("ReloadAndFixAddressFile")
	m.km.ReloadAndFixAddressFile()
	*reply = "success"
	return nil
}

func (m *WalletApiImpl) ImportPriv(hexkeypair []string, reply *string) error {
	log.Debug("ImportPriv")
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
	s, err := m.km.ExportPriv(addr.Hex(), pass)
	*reply = s
	return nil
}

func (m *WalletApiImpl) IsMayValidKeystoreFile(path []string, reply *string) error {
	log.Debug("IsValidKeystoreFile")
	if len(path) != 1 {
		return fmt.Errorf("wrong params len %v. you should pass [0] path", len(path))
	}
	b, _ := keystore.IsMayValidKeystoreFile(path[0])
	if b {
		*reply = "true"
	} else {
		*reply = "false"
	}
	return nil
}
