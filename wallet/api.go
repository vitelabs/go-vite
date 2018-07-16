package wallet

type HexSignedTuple struct {
	Message    string `json:"Message"`
	SignedData string `json:"SignedData"`
	Pubkey     string `json:"Pubkey"`
}

type ExternalAPI interface {

	ListAddress(v interface{}, reply *string) error

	// it will create a address and store in a dir and return address in hex form
	NewAddress(pwd string, reply *string) error

	// return value is all the address with  Locked  or Unlocked state
	// example
	// "vite_af136fb4cbd8804b8e40c64683f463555aa204b9db78965416 Locked
	//  vite_af136fb4cbd8804b8e40c64683f463555aa204b9db78965416 Unlocked"
	Status(v interface{}, reply *string) error

	// hexAddress := unlockParams[0] passphrase := unlockParams[1] unlocktime := unlockParams[2]
	// unlocks the given address with the passphrase. The account stays unlocked
	// for the duration of timeout (seconds)
	// if the timeout is <0 we will keep the unlock state until the program exit.
	UnLock(unlockParams []string, reply *string) error

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1]
	// if the given address has not been unlocked it will return an ErrUnlocked
	SignData(signDataParams []string, reply *string) error

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1] passphrase := signDataParams[2]
	SignDataWithPassphrase(signDataParams []string, reply *string) error

	// if a keystore file name is changed it will read the file content
	// if the  content is legal the function will fix the filename into hexaddress
	ReloadAndFixAddressFile(v interface{}, reply *string) error

	//hexprikey := hexkeypair[0] newPass := hexkeypair[1] the return value is address
	ImportPriv(hexkeypair []string, reply *string) error

	// hexaddr := extractPair[0] pass := extractPair[1] the return value is prikey in hex form
	ExportPriv(extractPair []string, reply *string) error
}
