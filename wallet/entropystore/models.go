package entropystore

const (
	storeVersion = 2
)

type versionAware struct {
	OldVersion *int `json:"seedstoreversion"`
	Version    *int `json:"version"`
}

type entropyJSONV1 struct {
	FlagData  string     `json:"flagData"`
	Crypto    cryptoJSON `json:"crypto"`
	Version   int        `json:"version"`
	Timestamp int64      `json:"timestamp"`
}

type entropyJSON struct {
	PrimaryAddress string     `json:"primaryAddress"`
	Crypto         cryptoJSON `json:"crypto"`
	Version        int        `json:"seedstoreversion"`
	Timestamp      int64      `json:"timestamp"`
}

type cryptoJSON struct {
	CipherName   string       `json:"ciphername"`
	CipherText   string       `json:"ciphertext"`
	Nonce        string       `json:"nonce"`
	KDF          string       `json:"kdf"`
	ScryptParams scryptParams `json:"scryptparams"`
}

type scryptParams struct {
	N      int    `json:"n"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	KeyLen int    `json:"keylen"`
	Salt   string `json:"salt"`
}
