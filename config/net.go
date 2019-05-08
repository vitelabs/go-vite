package config

import "github.com/vitelabs/go-vite/crypto/ed25519"

type Net struct {
	Single            bool   `json:"Single"`
	FileListenAddress string `json:"FileListenAddress"`
	ForwardStrategy   string `json:"ForwardStrategy"`
	MinePrivateKey    ed25519.PrivateKey
	P2PPrivateKey     ed25519.PrivateKey
}
