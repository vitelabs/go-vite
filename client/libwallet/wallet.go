package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"unsafe"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	ed255192 "github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/entropystore"
)

func main() {
}

type GoResult struct {
	Error string      `json:"error"`
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
}

func successResult() string {
	return `{"Code":0}`
}

func CString(str string) *C.char {
	return C.CString(str)
}

func GoString(str *C.char) string {
	return C.GoString(str)
}

func successResultWithData(data interface{}) string {
	result := GoResult{Code: 0, Data: data}
	bytes, err := json.Marshal(result)
	if err != nil {
		return failResult(errors.New("error when json data result"))
	}
	return string(bytes)
}

func failResult(err error) string {
	return `{"Error":"` + err.Error() + `","Code":1}`
}

var instance *wallet.Manager

//export InitWallet
func InitWallet(dataDir *C.char, maxSearchIndex int, useLightScrypt bool) *C.char {
	dataDirStr := GoString(dataDir)

	tmp := wallet.New(&wallet.Config{
		DataDir:        dataDirStr,
		MaxSearchIndex: uint32(maxSearchIndex),
		UseLightScrypt: useLightScrypt,
	})

	err := tmp.StartWallet()
	if err != nil {
		return CString(failResult(err))
	}

	instance = tmp

	return CString(successResult())
}

//export ListAllEntropyFiles
func ListAllEntropyFiles() *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}
	files := tmp.ListAllEntropyFiles()
	return CString(successResultWithData(files))
}

//export Unlock
func Unlock(entropyStore, passphrase *C.char) *C.char {
	entropyStoreStr := GoString(entropyStore)
	passphraseStr := GoString(passphrase)

	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	err := tmp.Unlock(entropyStoreStr, passphraseStr)
	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResult())
}

//export IsUnlocked
func IsUnlocked(entropyStore *C.char) *C.char {
	entropyStoreStr := GoString(entropyStore)

	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	isUnlocked := tmp.IsUnlocked(entropyStoreStr)
	return CString(successResultWithData(isUnlocked))
}

//export Lock
func Lock(entropyStore *C.char) *C.char {
	entropyStoreStr := GoString(entropyStore)

	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	err := tmp.Lock(entropyStoreStr)
	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResult())
}

//export AddEntropyStore
func AddEntropyStore(entropyStore *C.char) *C.char {
	entropyStoreStr := GoString(entropyStore)

	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}
	err := tmp.AddEntropyStore(entropyStoreStr)

	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResult())
}

//export RemoveEntropyStore
func RemoveEntropyStore(entropyStore *C.char) *C.char {
	entropyStoreStr := GoString(entropyStore)

	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}
	tmp.RemoveEntropyStore(entropyStoreStr)
	return CString(successResult())
}

//export RecoverEntropyStoreFromMnemonic
func RecoverEntropyStoreFromMnemonic(mnemonic, newPassphrase, language, extensionWord *C.char) *C.char {
	mnemonicStr := GoString(mnemonic)
	newPassphraseStr := GoString(newPassphrase)
	languageStr := GoString(language)
	extensionWordStr := GoString(extensionWord)

	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	tmpExtensionWordStr := &extensionWordStr
	if len(extensionWordStr) == 0 {
		tmpExtensionWordStr = nil
	}

	em, err := tmp.RecoverEntropyStoreFromMnemonic(mnemonicStr, languageStr, newPassphraseStr, tmpExtensionWordStr)
	if err != nil {
		return CString(failResult(err))
	}

	return CString(successResultWithData(em.GetPrimaryAddr().String()))
}

type EntropyResult struct {
	Mnemonic     string `json:"mnemonic"`
	EntropyStore string `json:"entropyStore"`
}

//export NewMnemonicAndEntropyStore
func NewMnemonicAndEntropyStore(passphrase, language, extensionWord *C.char, mnemonicSize int) *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	passphraseStr := GoString(passphrase)
	languageStr := GoString(language)
	extensionWordStr := GoString(extensionWord)

	tmpExtensionWordStr := &extensionWordStr
	if extensionWordStr == "" {
		tmpExtensionWordStr = nil
	}

	mnemonic, em, err := tmp.NewMnemonicAndEntropyStore(languageStr, passphraseStr, tmpExtensionWordStr, &mnemonicSize)

	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResultWithData(EntropyResult{
		Mnemonic:     mnemonic,
		EntropyStore: em.GetPrimaryAddr().String(),
	}))
}

type DerivationResult struct {
	Path       string `json:"path"`
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
}

//export DeriveByFullPath
func DeriveByFullPath(entropyStore, fullpath, extensionWord *C.char) *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	entropyStoreStr := GoString(entropyStore)
	fullPathStr := GoString(fullpath)
	extensionWordStr := GoString(extensionWord)

	tmpExtensionWordStr := &extensionWordStr
	if extensionWordStr == "" {
		tmpExtensionWordStr = nil
	}

	manager, err := tmp.GetEntropyStoreManager(entropyStoreStr)
	if err != nil {
		return CString(failResult(err))
	}

	fpath, key, err := manager.DeriveForFullPath(fullPathStr, tmpExtensionWordStr)
	if err != nil {
		return CString(failResult(err))
	}
	addr, err := key.Address()
	if err != nil {
		return CString(failResult(err))
	}
	keys, err := key.PrivateKey()
	if err != nil {
		return CString(failResult(err))
	}

	return CString(successResultWithData(DerivationResult{
		Path:       fpath,
		Address:    addr.String(),
		PrivateKey: keys.Hex(),
	}))
}

//export DeriveByIndex
func DeriveByIndex(entropyStore *C.char, index int, extensionWord *C.char) *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	entropyStoreStr := GoString(entropyStore)

	extensionWordStr := GoString(extensionWord)

	tmpExtensionWordStr := &extensionWordStr
	if extensionWordStr == "" {
		tmpExtensionWordStr = nil
	}

	manager, err := tmp.GetEntropyStoreManager(entropyStoreStr)
	if err != nil {
		return CString(failResult(err))
	}

	fpath, key, err := manager.DeriveForIndexPath(uint32(index), tmpExtensionWordStr)
	if err != nil {
		return CString(failResult(err))
	}
	addr, err := key.Address()
	if err != nil {
		return CString(failResult(err))
	}
	keys, err := key.PrivateKey()
	if err != nil {
		return CString(failResult(err))
	}

	return CString(successResultWithData(DerivationResult{
		Path:       fpath,
		Address:    addr.String(),
		PrivateKey: keys.Hex(),
	}))
}

//export ExtractMnemonic
func ExtractMnemonic(entropyStore, passphrase *C.char) *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	entropyStoreStr := GoString(entropyStore)
	passphraseStr := GoString(passphrase)
	mnemonic, err := tmp.ExtractMnemonic(entropyStoreStr, passphraseStr)
	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResultWithData(mnemonic))
}

//export GetDataDir
func GetDataDir() *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}
	return CString(successResultWithData(tmp.GetDataDir()))
}

//export EntropyStoreToAddress
func EntropyStoreToAddress(entropyStore *C.char) *C.char {
	tmp := instance
	if tmp == nil {
		return CString(failResult(errors.New("wallet should be init")))
	}

	entropyStoreStr := GoString(entropyStore)

	addrStr := entropyStoreStr
	if filepath.IsAbs(entropyStoreStr) {
		addrStr = filepath.Base(entropyStoreStr)
	}
	address, err := types.HexToAddress(addrStr)
	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResultWithData(address.Hex()))
}

//export Hash256
func Hash256(data *C.char) *C.char {
	dataBase64 := GoString(data)
	byt, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return CString(failResult(errors.New("base decode fail")))
	}
	return CString(successResultWithData(base64.StdEncoding.EncodeToString(crypto.Hash256(byt))))
}

//export Hash
func Hash(size int, data *C.char) *C.char {
	dataBase64 := GoString(data)
	byt, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return CString(failResult(errors.New("base decode fail")))
	}
	return CString(successResultWithData(base64.StdEncoding.EncodeToString(crypto.Hash(size, byt))))
}

type SignDataResult struct {
	PublicKey string `json:"publicKey"`
	Data      string `json:"data"`
	Signature string `json:"signature"`
}

//export SignData
func SignData(privHex *C.char, messageBase64 *C.char) *C.char {
	priKey, err := ed255192.HexToPrivateKey(GoString(privHex))
	if err != nil {
		return CString(failResult(err))
	}

	messageBase64Str := GoString(messageBase64)
	message, err := base64.StdEncoding.DecodeString(messageBase64Str)
	if err != nil {
		return CString(failResult(err))
	}
	var byt []byte = priKey
	var a ed25519.PrivateKey = byt
	signature := ed25519.Sign(a, message)

	return CString(successResultWithData(SignDataResult{
		PublicKey: base64.StdEncoding.EncodeToString(priKey.PubByte()),
		Data:      messageBase64Str,
		Signature: base64.StdEncoding.EncodeToString(signature),
	}))
}

//export VerifySignature
func VerifySignature(pub, message, signData *C.char) *C.char {
	pubByt, err := base64.StdEncoding.DecodeString(GoString(pub))
	if err != nil {
		return CString(failResult(err))
	}
	messageByt, err := base64.StdEncoding.DecodeString(GoString(message))
	if err != nil {
		return CString(failResult(err))
	}
	signDataByt, err := base64.StdEncoding.DecodeString(GoString(signData))
	if err != nil {
		return CString(failResult(err))
	}

	result, err := crypto.VerifySig(pubByt, messageByt, signDataByt)
	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResultWithData(result))
}

//export PubkeyToAddress
func PubkeyToAddress(pubBase64 *C.char) *C.char {
	pubByt, err := base64.StdEncoding.DecodeString(GoString(pubBase64))
	if err != nil {
		return CString(failResult(err))
	}
	address := types.PubkeyToAddress(pubByt)
	return CString(successResultWithData(address.Hex()))
}

//export TransformMnemonic
func TransformMnemonic(mnemonic, language, extensionWord *C.char) *C.char {
	extensionWordStr := GoString(extensionWord)
	extensionWordP := &extensionWordStr
	if extensionWordStr == "" {
		extensionWordP = nil
	}
	entropyprofile, e := entropystore.MnemonicToEntropy(GoString(mnemonic), GoString(language), extensionWordP != nil, extensionWordP)
	if e != nil {
		return CString(failResult(e))
	}
	return CString(successResultWithData(entropyprofile.PrimaryAddress.Hex()))
}

//export RandomMnemonic
func RandomMnemonic(language *C.char, mnemonicSize int) *C.char {
	mnemonic, err := entropystore.NewMnemonic(GoString(language), &mnemonicSize)
	if err != nil {
		return CString(failResult(err))
	}
	return CString(successResultWithData(mnemonic))
}

//export ComputeHashForAccountBlock
func ComputeHashForAccountBlock(block *C.char) *C.char {
	blockStr := GoString(block)

	accBlock := ledger.AccountBlock{}
	err := json.Unmarshal([]byte(blockStr), &accBlock)
	if err != nil {
		return CString(failResult(err))
	}
	hash := accBlock.ComputeHash()
	return CString(successResultWithData(hash.Hex()))
}

//export Hello
func Hello(name *C.char) *C.char {
	return CString("Hello " + GoString(name) + " !")
}

//export FreeCchar
func FreeCchar(c *C.char) {
	C.free(unsafe.Pointer(c))
}
