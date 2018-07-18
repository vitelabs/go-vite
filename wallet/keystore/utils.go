package keystore

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// it it return false it must not be a valid keystore file
// if it return a true it only means that might be true
func IsMayValidKeystoreFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	// out keystore file size is about 500 so if a file is very large it must not be a keystore file
	if fi.Size() > 2*1024 {
		return false, nil
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return false, err
	}
	_, _, _, _, _, err = parseJson(b)
	if err != nil {
		return false, err
	}
	return true, nil
}

func readAndFixAddressFile(path string) (*types.Address, *encryptedKeyJSON) {
	buf := new(bufio.Reader)
	keyJSON := encryptedKeyJSON{}

	fd, err := os.Open(path)
	if err != nil {
		log.Trace("Can not to open ", "path", path, "err", err)
		return nil, nil
	}
	defer fd.Close()
	buf.Reset(fd)
	keyJSON.HexAddress = ""
	err = json.NewDecoder(buf).Decode(&keyJSON)
	if err != nil {
		log.Trace("Decode keystore file failed ", "path", path, "err", err)
		return nil, nil
	}
	addr, err := types.HexToAddress(keyJSON.HexAddress)
	if err != nil {
		log.Trace("Address is invalid ", "path", path, "err", err)
		return nil, nil
	}

	// fix the file name
	standFileName := fullKeyFileName(filepath.Dir(path), addr)
	if standFileName != fd.Name() {
		os.Rename(fd.Name(), standFileName)
	}
	return &addr, &keyJSON

}

func fullKeyFileName(keysDirPath string, keyAddr types.Address) string {
	return filepath.Join(keysDirPath, keyAddr.Hex())
}

func addressFromKeyPath(keyfile string) (types.Address, error) {
	_, filename := filepath.Split(keyfile)
	return types.HexToAddress(filename)
}

func fullKeyFileNameV0(keysDirPath string, keyAddr types.Address) string {
	return filepath.Join(keysDirPath, "/v-i-t-e-"+hex.EncodeToString(keyAddr[:]))
}

func addressFromKeyPathV0(keyfile string) (types.Address, error) {
	_, filename := filepath.Split(keyfile)
	if !strings.HasPrefix(filename, "v-i-t-e-") {
		return types.Address{}, fmt.Errorf("not valid key file name %v", keyfile)
	}
	b, err := hex.DecodeString(filename[len("v-i-t-e-"):])
	if err != nil {
		return types.Address{}, fmt.Errorf("not valid key file name %v error %v", keyfile, err)
	}
	if len(b) != types.AddressSize {
		return types.Address{}, fmt.Errorf("not valid key file name %v error %v", keyfile)
	}

	a, err := types.BytesToAddress(b)
	if err != nil {
		return types.Address{}, fmt.Errorf("not valid key file name %v error %v", keyfile, err)
	}
	return a, nil
}
