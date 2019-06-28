package net

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

var errHashBlocked = errors.New("the block hash is invalid")

var defaultInvalidHashes []types.Hash

func init() {
	hash, _ := types.HexToHash("b07c664782e2e0b439191d0bba5d9a97bdbc640ab1008569bafb1b836fc0a034")
	defaultInvalidHashes = append(defaultInvalidHashes, hash)
}

type InvalidHash struct {
	HashList []string `json:"hashList"`
}

const invalidHashFile = "interHash.json"

func loadCustomInvalidHashes() (ret []types.Hash) {
	content, err := ioutil.ReadFile(invalidHashFile)

	if err != nil {
		return nil
	}

	hs := new(InvalidHash)
	err = json.Unmarshal(content, hs)
	if err != nil {
		panic(err)
	}

	for _, hstr := range hs.HashList {
		var h types.Hash
		h, err = types.HexToHash(hstr)
		if err != nil {
			panic(err)
		}

		ret = append(ret, h)
	}

	return
}

type verifier struct {
	v Verifier
	m map[types.Hash]struct{}
}

func newVerifier(v Verifier) Verifier {
	var wv = &verifier{
		v: v,
		m: make(map[types.Hash]struct{}),
	}

	for _, hash := range defaultInvalidHashes {
		wv.m[hash] = struct{}{}
	}
	for _, hash := range loadCustomInvalidHashes() {
		wv.m[hash] = struct{}{}
	}

	return wv
}

func (v *verifier) VerifyNetSb(block *ledger.SnapshotBlock) error {
	if _, ok := v.m[block.Hash]; ok {
		return errHashBlocked
	}
	return v.v.VerifyNetSb(block)
}

func (v *verifier) VerifyNetAb(block *ledger.AccountBlock) error {
	if _, ok := v.m[block.Hash]; ok {
		return errHashBlocked
	}
	return v.v.VerifyNetAb(block)
}
