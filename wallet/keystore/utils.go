package keystore

import (
	"github.com/vitelabs/go-vite/common/types"
	"path/filepath"
	"encoding/hex"
	"strings"
	"fmt"
	"bufio"
	"os"
	"encoding/json"
	"github.com/vitelabs/go-vite/log"
)

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

func readAndFixAddressFile(path string) *types.Address {
	buf := new(bufio.Reader)
	key := encryptedKeyJSON{}

	fd, err := os.Open(path)
	if err != nil {
		log.Trace("Can not to open ", "path", path, "err", err)
		return nil
	}
	defer fd.Close()
	buf.Reset(fd)
	key.HexAddress = ""
	err = json.NewDecoder(buf).Decode(&key)
	if err != nil {
		log.Trace("Decode keystore file failed ", "path", path, "err", err)
		return nil
	}
	addr, err := types.HexToAddress(key.HexAddress)
	if err != nil {
		log.Trace("Address is invalid ", "path", path, "err", err)
		return nil
	}
	os.Rename(fd.Name(), fullKeyFileName(filepath.Dir(path), addr))
	return &addr

}