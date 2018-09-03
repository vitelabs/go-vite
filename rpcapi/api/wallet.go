package api

import (
	"github.com/vitelabs/go-vite/common/types"
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
	ListAddress() []types.Address

	// it will create a address and store in a dir
	// passphrase len must be 1, the reply string is hex-formed address
	NewAddress(passphrase string) (types.Address, error)

	// return value is all the address with  Locked  or Unlocked state
	// example:
	// {"vite_15dac990004ae1cbf1af913091092f7c45205b88d905881d97":"Locked",
	// "vite_48c5a659e37a9a462b96ff49ef3d30f10137c600417ce05cda":"Unlocked"}
	Status() map[types.Address]string

	// hexAddress := unlockParams[0] passphrase := unlockParams[1] unlocktime := unlockParams[2]
	// unlocks the given address with the passphrase. The account stays unlocked for the duration of timeout (seconds)
	// if the timeout is <0 we will keep the unlock state until the program exit.
	UnlockAddress(addr types.Address, password string, duration *uint64) (bool, error)

	// you must pass an address into lockParams , if no error happened means lock success
	LockAddress(addr types.Address) error

	// if a keystore file name is changed it will read the file content
	// if the  content is legal the function will fix the filename into hex-formed address
	ReloadAndFixAddressFile() error

	// hexprikey := hexkeypair[0] newPass := hexkeypair[1] the reply string is hex-formed address
	ImportPriv(privkey string, password string) (types.Address, error)

	// hexaddr := extractPair[0] pass := extractPair[1] the return value is prikey in hex form
	ExportPriv(address types.Address, password string) (string, error)

	// it reply string is false it must not be a valid keystore file
	// else only means that might be a keystore file
	IsMayValidKeystoreFile(path string) types.Address

	// Get dir
	GetDataDir() string

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1]
	// if the given address has not been unlocked it will return an ErrUnlocked
	SignData(address types.Address, hexMsg string) (HexSignedTuple, error)

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1] passphrase := signDataParams[2]
	SignDataWithPassphrase(address types.Address, hexMsg string, password string) (HexSignedTuple, error)
}
