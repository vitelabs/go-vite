package api

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

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1]
	// if the given address has not been unlocked it will return an ErrUnlocked
	// SignData(signDataParams []string, reply *string) error

	// hexAddress := signDataParams[0] hexMsg := signDataParams[1] passphrase := signDataParams[2]
	// SignDataWithPassphrase(signDataParams []string, reply *string) error
}
