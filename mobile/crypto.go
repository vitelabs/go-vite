package mobile

import "github.com/vitelabs/go-vite/crypto"

func BlakeHash256(data ...[]byte) []byte {
	return crypto.Hash256(data...)
}
