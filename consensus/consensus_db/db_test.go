package consensus_db

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestNewConsensusDBByDir(t *testing.T) {
	db := NewConsensusDBByDir("/Users/jie/Library/GVite/testdata/consensus")
	//key, err := database.EncodeKey(INDEX_ElectionResult)
	//if err != nil {
	//	panic(err)
	//}
	//iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	iter := db.db.NewIterator(nil, nil)
	i := uint64(0)
	for ; iter.Next(); i++ {
		bytes := iter.Key()
		//value := iter.Value()
		hash, err := types.BytesToHash(bytes[1:])
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
		fmt.Println(hash, len(iter.Value()), string(iter.Value()))
	}

	hash := types.HexToHashPanic("0bc0f2af1d2d2583259e0c4931857b9a1249eb03bd1a1085197a63555489d79e")

	bytes, e := db.GetElectionResultByHash(hash)
	if e != nil {
		panic(e)
	}

	fmt.Println(string(bytes))

	var dd []types.Address

	err := json.Unmarshal(bytes, &dd)

	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", dd)
}

func TestPasrse(t *testing.T) {
	addr1, err := types.HexToAddress("vite_29a3ec96d1e4a52f50e3119ed9945de09bef1d74a772ee60ff")
	if err != nil {
		panic(err)
	}
	addr2, err := types.HexToAddress("vite_0d97a04d61dbbf238e9b6050c9d29d2b6f97986859e7856ec9")
	if err != nil {
		panic(err)
	}
	var dd []types.Address

	dd = append(dd, addr1)
	dd = append(dd, addr2)
	arr := AddrArr(dd)
	result := arr.Bytes()
	bytes, _ := json.Marshal(&dd)

	fmt.Println(len(result), len(bytes), " byte len")
	var rst AddrArr
	rerrr, _ := rst.SetBytes(result)
	fmt.Printf("%+v\n", rerrr)
}
