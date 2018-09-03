package keystore

import (
	"encoding/hex"
	"encoding/json"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	vcrypto "github.com/vitelabs/go-vite/crypto"
	"path/filepath"
	"runtime"
	"testing"
)

const (
	DummySignData = "123456123456123456123456123456"
	DummyPwd      = "123456"
)

func TestStoreAndExtractNewKey(t *testing.T) {

	dir := filepath.Join(common.GoViteTestDataDir(), "super")
	ks := keyStorePassphrase{keysDirPath: dir}

	kp := NewManager(dir)
	kp.Init()

	key1, err := kp.StoreNewKey(DummyPwd)
	if err != nil {
		t.Fatal(err)
	}

	println("Encrypt finish")

	key2, err := ks.ExtractKey(key1.Address, DummyPwd)
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

func TestSignAndVerify(t *testing.T) {
	kp := NewManager(common.GoViteTestDataDir())
	kp.Init()
	for _, v := range kp.Addresses() {
		println(v.Hex())
		outdata, pubkey, err := kp.SignDataWithPassphrase(v, DummyPwd, []byte(DummySignData))
		if err != nil {
			t.Fatal(err)
		}
		println("##" + hex.EncodeToString(outdata))
		readAndFixAddressFile(fullKeyFileName(common.GoViteTestDataDir(), v))
		if err != nil {
			t.Fatal(err)
		}

		ok, err := vcrypto.VerifySig(pubkey, []byte(DummySignData), outdata)

		if !ok || err != nil {
			t.Fatal("Verify wrong")
		}
	}
}

func TestManager_ImportPriv2(t *testing.T) {
	kp := NewManager(filepath.Join(common.DefaultDataDir(), "wallet"))
	kp.Init()
	hexPri := "ab565d7d8819a3548dbdae8561796ccb090692086ff7d5a47eb7b034497cabe73af9a47a11140c681c2b2a85a4ce987fab0692589b2ce233bf7e174bd430177a"
	key, e := kp.ImportPriv(hexPri, "123456")
	if e != nil {
		println(e.Error())
	} else {
		println(key.Address.String())
	}
}

func TestManager_ImportPriv(t *testing.T) {
	kp := NewManager(common.GoViteTestDataDir())
	kp.Init()
	hexPri, err := kp.ExportPriv("vite_af136fb4cbd8804b8e40c64683f463555aa204b9db78965416", DummyPwd)
	if err != nil {
		t.Fatal(err)
	}

	kp.ImportPriv(hexPri, "654321")

	hexpri1, err := kp.ExportPriv("vite_af136fb4cbd8804b8e40c64683f463555aa204b9db78965416", "654321")
	if err != nil {
		t.Fatal(err)
	}
	if hexPri != hexpri1 {
		t.Fatalf("1: %v != 2: %v", hexpri1, hexPri)
	}
	kp.ImportPriv(hexpri1, DummyPwd)
}

func TestManager_Import(t *testing.T) {
	kp := NewManager(common.GoViteTestDataDir())
	kp.Init()
	hexaddr := "vite_af136fb4cbd8804b8e40c64683f463555aa204b9db78965416"
	addr, _ := types.HexToAddress(hexaddr)

	key0, err := kp.ExtractKey(addr, DummyPwd)
	if err != nil {
		t.Fatal(err)
	}
	json, err := kp.Export(hexaddr, DummyPwd, "654321")
	if err != nil {
		t.Fatal(err)
	}

	kp.Import(json, "654321", "123123")
	key1, err := kp.ExtractKey(addr, "123123")
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

func TestDir(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)

	println(filename)
}

func TestManager_Status(t *testing.T) {
	m := make(map[string]string)
	//a0, _, _ := types.CreateAddress()
	//m[a0] = "lock"
	//a1, _, _ := types.CreateAddress()
	//m[a1] = "unlock"
	//bytes, e := json.Marshal(m)
	//if e != nil {
	//	t.Fatal(e)
	//}
	//fmt.Println(string(bytes))

	unmarshal := json.Unmarshal(
		[]byte(`{"vite_642b00ebfdc76c12fdd8f7272c174f8646d615cfc03c41aac7":"lock","vite_cf1411bcbb5aac657b4607ac3cfdeac843b22c2de6ae23685b":"unlock"}`),
		&m)
	if unmarshal != nil {
		t.Fatal(unmarshal)
	}
}

