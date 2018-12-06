package mobile

import (
	"github.com/vitelabs/go-vite/common/types"
)

type Address struct {
	address types.Address
}

func IsValidAddress(s string) bool {
	_, e := types.HexToAddress(s)
	return e == nil
}

func NewAddressFromByte(binary []byte) (address *Address, _ error) {
	addresses, e := types.BytesToAddress(binary)
	if e != nil {
		return nil, e
	}
	a := new(Address)
	a.address = addresses
	return a, nil
}

func NewAddressFromString(s string) (address *Address, _ error) {
	addresses, e := types.HexToAddress(s)
	if e != nil {
		return nil, e
	}
	a := new(Address)
	a.address = addresses
	return a, nil
}

func (a *Address) SetBytes(address []byte) error {
	return a.address.SetBytes(address)
}

func (a *Address) GetBytes() []byte {
	return a.address[:]
}

func (a *Address) SetHex(s string) error {
	addresses, e := types.HexToAddress(s)
	if e != nil {
		return e
	}
	a.address = addresses
	return nil
}

func (a *Address) GetHex() string {
	return a.address.Hex()
}

type TokenTypeId struct {
	tti types.TokenTypeId
}

func IsValidTokenTypeId(s string) bool {
	_, e := types.HexToTokenTypeId(s)
	return e == nil
}
func NewTokenTypeIdFromByte(binary []byte) (tti *TokenTypeId, _ error) {
	atti, e := types.BytesToTokenTypeId(binary)
	if e != nil {
		return nil, e
	}
	a := new(TokenTypeId)
	a.tti = atti
	return a, nil
}

func NewTokenTypeIdFromString(s string) (tti *TokenTypeId, _ error) {
	htti, e := types.HexToTokenTypeId(s)
	if e != nil {
		return nil, e
	}
	a := new(TokenTypeId)
	a.tti = htti
	return a, nil
}

func (t *TokenTypeId) SetBytes(tti []byte) error {
	return t.tti.SetBytes(tti)
}

func (t *TokenTypeId) GetBytes() []byte {
	return t.tti[:]
}

func (t *TokenTypeId) SetHex(s string) error {
	tti, e := types.HexToTokenTypeId(s)
	if e != nil {
		return e
	}
	t.tti = tti
	return nil
}

func (t *TokenTypeId) GetHex() string {
	return t.tti.Hex()
}
