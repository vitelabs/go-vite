package config

type Producer struct {
	Producer         bool   `json:"Producer"`
	Coinbase         string `json:"Coinbase"`
	EntropyStorePath string `json:"EntropyStorePath"`
}
