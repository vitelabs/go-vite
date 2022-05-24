package derivation

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveFromPath(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	key, err := DeriveForPath(VitePrimaryAccountPath, seed)
	if err != nil {
		panic(err)
	}

	s, address, e := key.StringPair()
	if e != nil {
		fmt.Println(e)
	}

	fmt.Println(VitePrimaryAccountPath, s, " "+address)
}

func TestDeriveMultipleKeys(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")

	for i := 0; i < 10; i++ {
		path := fmt.Sprintf(ViteAccountPathFormat, i)
		key, err := DeriveForPath(path, seed)
		if err != nil {
			panic(err)
		}

		s, address, e := key.StringPair()
		if e != nil {
			panic(err)
		}

		fmt.Println(path, s, address)
	}
}

func TestDeriveMultipleKeysFaster(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	mainKey, err := DeriveForPath(ViteAccountPrefix, seed)
	if err != nil {
		panic(err)
	}

	for i := uint32(0); i < 10; i++ {
		key, err := mainKey.Derive(FirstHardenedIndex + i)
		if err != nil {
			panic(err)
		}

		s, address, err := key.StringPair()
		if err != nil {
			panic(err)
		}

		fmt.Println(fmt.Sprintf(ViteAccountPathFormat, i), s, address)
	}
}

func BenchmarkDerive(b *testing.B) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")

	for i := 0; i < b.N; i++ {
		_, err := DeriveForPath(VitePrimaryAccountPath, seed)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkDeriveFast(b *testing.B) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	mainKey, err := DeriveForPath(ViteAccountPrefix, seed)
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := mainKey.Derive(FirstHardenedIndex)
		if err != nil {
			panic(err)
		}
	}
}

func TestIsValidPath(t *testing.T) {
	TestDeriveMultipleKeysFaster(t)
	assert.True(t, isValidPath("m/0'"))
	assert.True(t, isValidPath("m/0'/100'"))
	assert.True(t, isValidPath("m/0'/100'/200'"))
	assert.True(t, isValidPath("m/0'/100'/200'/300'"))

	assert.False(t, isValidPath("dummy"))
	assert.False(t, isValidPath("m"))                           // Master key only
	assert.False(t, isValidPath("m/0"))                         // Missing '
	assert.False(t, isValidPath("m/0'/"))                       // Trailing slash
	assert.False(t, isValidPath("m/0'/893478327492379497823'")) // Overflow
}

// https://github.com/satoshilabs/slips/blob/master/slip-0010.md#test-vector-1-for-ed25519
func TestDeriveVector1(t *testing.T) {
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	assert.NoError(t, err)

	key, err := NewMasterKey(seed)
	assert.NoError(t, err)
	assert.Equal(t, "963d056ba6d197ef5a46d4965ed699cc35e47fc0841d0b5da3e4a819fbe1e2a1", hex.EncodeToString(key.Key))
	assert.Equal(t, "9d7e39189b93135c9b9bc29131d7fc1ed01a2f96165bbae036cd5baf871e3f23", hex.EncodeToString(key.ChainCode))
	publicKey, err := key.PublicKey()
	assert.NoError(t, err)
	assert.Equal(t, "00c39ad5f7156fd43eab9daa32a1230bcb75e4b4387f0afb3d5cd5159a9339b7ad", hex.EncodeToString(append([]byte{0x0}, publicKey...)))

	tests := []struct {
		Path       string
		ChainCode  string
		PrivateKey string
		PublicKey  string
	}{
		{
			Path:       "m/0'",
			ChainCode:  "c09508830a25ddf11c85411a3722937829cae8b338bbb8d2ecdfd0cd53c8011b",
			PrivateKey: "bc632e40074ab1d1dd7fd2463e9837218852a00a998ac43e1f4907f67834e77e",
			PublicKey:  "00ca3f8440fdd050a9af004538a6e5e2d329138aeaf25aa72119403717569edcab",
		},
		{
			Path:       "m/0'/1'",
			ChainCode:  "f90c07c845c94be14db0615de107f281f7de7a6af333d6d6ad4c8cd07fe76f0e",
			PrivateKey: "5351f75816232b0d0799a1d2c5e2d90b3fa26fd0452c2214f046d4e92096f07e",
			PublicKey:  "008c4093af763b1d7c161151a12796be0285806ea46c7fa06ea31bcf718d531059",
		},
		{
			Path:       "m/0'/1'/2'",
			ChainCode:  "52f1714edb01c49dc9393dba4b82b6c14789bdcdb79b1a4cd188f8d82673e871",
			PrivateKey: "32d9c872085e9fedf7af3eec9f2dc478c3e5dccdfcfb7ad13b0ba44a6e89a55d",
			PublicKey:  "00851d89ef83518ec04832a6d56e37c7f5f937c207ca123154e8f4fc663f2baeac",
		},
		{
			Path:       "m/0'/1'/2'/2'",
			ChainCode:  "46a360b9633485a7d4a82847fc94524a65ff84d599784d1fd45a6f994d04a923",
			PrivateKey: "48f318a03f3f9873c759fc8adeceb01b35c50825522fe486cec5471b30347cdf",
			PublicKey:  "0024251409a38e214b7b161cc3ae267a8175500af8b7c2827d403c480a5dce3478",
		},
		{
			Path:       "m/0'/1'/2'/2'/1000000000'",
			ChainCode:  "6a6242b2466565d478a8bfa54b65dc402ff79433f091f57960c67e1d1dc9ccbe",
			PrivateKey: "eaaf7e3ea8e2dedf08bf872932c8c57ad5ee0c15282ed2cbd6095842fd429d36",
			PublicKey:  "00d8df4d8cfd365c3da2b6cb7135397c3f2ee7786d823516123bec9458120bc107",
		},
	}

	for _, test := range tests {
		key, err = DeriveForPath(test.Path, seed)
		assert.NoError(t, err)
		assert.Equal(t, test.PrivateKey, hex.EncodeToString(key.Key))
		assert.Equal(t, test.ChainCode, hex.EncodeToString(key.ChainCode))
		publicKey, err := key.PublicKey()
		assert.NoError(t, err)
		assert.Equal(t, test.PublicKey, hex.EncodeToString(append([]byte{0x0}, publicKey...)))
	}
}