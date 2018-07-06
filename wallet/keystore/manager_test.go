package keystore

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"testing"
)

const (
	DummySignData = "123456123456123456123456123456"
	DummyPwd      = "123456"
)

func TestStoreAndExtractNewKey(t *testing.T) {

	ks := KeyStorePassphrase{keysDirPath: common.DefaultDataDir()}
	kp := NewManager(&DefaultKeyConfig)

	key1, addr1, err := kp.StoreNewKey(DummyPwd)
	if err != nil {
		t.Fatal(err)
	}

	println("Encrypt finish")

	key2, err := ks.ExtractKey(addr1, DummyPwd)
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
	status, _ := kp.Status()
	println(status)
	for _, v := range kp.ListAddress() {
		println(v.Hex())
		outdata, err := kp.SignDataWithPassphrase(v, DummyPwd, []byte(DummySignData))
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

func TestManager_ImportPriv(t *testing.T) {
	kp := NewManager(&DefaultKeyConfig)
	kp.Init()
	hexPri, err := kp.ExportPriv("vite_452cab46cf322a85e4390984e43f69d6f32dd8352bf46cd870", DummyPwd)
	if err != nil {
		t.Fatal(err)
	}

	kp.ImportPriv(hexPri, "654321")

	hexpri1, err := kp.ExportPriv("vite_452cab46cf322a85e4390984e43f69d6f32dd8352bf46cd870", "654321")
	if err != nil {
		t.Fatal(err)
	}
	if hexPri != hexpri1 {
		t.Fatalf("1: %v != 2: %v", hexpri1, hexPri)
	}
	kp.ImportPriv(hexpri1, DummyPwd)
}

func TestManager_Import(t *testing.T) {
	kp := NewManager(&DefaultKeyConfig)
	kp.Init()
	hexaddr := "vite_452cab46cf322a85e4390984e43f69d6f32dd8352bf46cd870"
	addr, _ := types.HexToAddress(hexaddr)

	_, key0, err := kp.ExtractKey(addr, DummyPwd)
	if err != nil {
		t.Fatal(err)
	}
	json, err := kp.Export(hexaddr, DummyPwd, "654321")
	if err != nil {
		t.Fatal(err)
	}

	kp.Import(json, "654321", "123123")
	_, key1, err := kp.ExtractKey(addr, "123123")
	if err != nil {
		t.Fatal(err)
	}
	if key1.PrivateKey.Hex() != key0.PrivateKey.Hex() {
		t.Fatalf("1: %v != 2: %v", key1.PrivateKey.Hex(), key0.PrivateKey.Hex())
	}

	json1, err := kp.Export(hexaddr, "123123", "123111")
	if err != nil {
		t.Fatal(err)
	}
	kp.Import(json1, "123111", DummyPwd)

}
