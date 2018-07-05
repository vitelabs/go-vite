package keystore

import (
	"github.com/vitelabs/go-vite/common"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"testing"
	"encoding/hex"
)

const DummySignData = "123456123456123456123456123456"

func TestStoreAndExtractNewKey(t *testing.T) {

	ks := KeyStorePassphrase{keysDirPath: common.DefaultDataDir()}
	kp := NewManager(&DefaultKeyConfig)

	key1, addr1, err := kp.StoreNewKey("123456")
	if err != nil {
		t.Fatal(err)
	}

	println("Encrypt finish")

	key2, err := ks.ExtractKey(addr1, "123456")
	if err != nil {
		t.Fatal(err)
	}
	println("decrypt finish")
	println(key1.PrivateKey.Hex())
	println(key2.PrivateKey.Hex())

	if key1.PrivateKey.Hex() != key2.PrivateKey.Hex() {
		t.Fatalf("miss PrivateKey content")
	}

}

func TestSignAndVerfify(t *testing.T) {
	kp := NewManager(&DefaultKeyConfig)
	kp.Init()
	for _, v := range kp.ListAddress() {
		println(v.Hex())
		outdata, err := kp.SignDataWithPassphrase(v, "123456", []byte(DummySignData))
		if err != nil {
			t.Fatal(err)
		}
		println(hex.EncodeToString(outdata))
		_, ek := readAndFixAddressFile(fullKeyFileName(kp.keyConfig.KeyStoreDir, v))
		pub, err := ed25519.HexToPublicKey(ek.HexPubKey)
		if err != nil {
			t.Fatal(err)
		}
		if !vcrypto.VerifySig(&pub, []byte(DummySignData), outdata) {
			t.Fatal("Verify wrong")
		}
	}
}
