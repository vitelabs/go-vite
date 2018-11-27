package entropystore

const (
	cryptoStoreVersion = 1
)

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
