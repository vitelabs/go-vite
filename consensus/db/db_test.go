package consensus_db

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/go-errors/errors"

	"github.com/magiconair/properties/assert"

	"github.com/syndtr/goleveldb/leveldb"

	"github.com/vitelabs/go-vite/common/types"
)

func prepareConsensusDB() *ConsensusDB {
	clearConsensusDB(nil)
	d, err := leveldb.OpenFile("testdata-consensus", nil)
	if err != nil {
		panic(err)
	}

	db := NewConsensusDB(d)
	return db
}

func clearConsensusDB(db *ConsensusDB) {
	if db != nil {
		db.db.Close()
	}
	os.RemoveAll("testdata-consensus")
}

func TestNewConsensusDB(t *testing.T) {
	db := prepareConsensusDB()
	defer clearConsensusDB(db)
}

func TestConsensusDB_Point(t *testing.T) {
	db := prepareConsensusDB()
	defer clearConsensusDB(db)

	// get empty
	p1, err := db.GetPointByHeight(INDEX_Point_DAY, 1)
	if err != nil {
		panic(err)
	}
	if p1 != nil {
		panic(errors.New("fail"))
	}

	// store point
	h1 := types.HexToHashPanic("f3b9187d69e0749e28f9c8172fd6b7b468cbe89acb14f8038bbbb6a402d738ae")
	h2 := types.HexToHashPanic("d7251a9d1da157dcfd20a729fc368f1dfc85a55e4057df229fbb1bacb3405385")
	infos := make(map[types.Address]*Content)
	addr1, _ := types.HexToAddress("vite_91dc0c38d104c7915d3a6c4381a40c360edd871c34ac255bb2")
	addr2, _ := types.HexToAddress("vite_d6851aaf8966f4550bde3d64582f7da9c6ab8b18a289823b95")
	infos[addr1] = &Content{3, 2}
	infos[addr2] = &Content{9, 7}
	p := &point{PreHash: h1, Hash: h2, Sbps: infos}

	err = db.StorePointByHeight(INDEX_Point_DAY, 1, p)
	if err != nil {
		panic(err)
	}

	// get point
	p1, err = db.GetPointByHeight(INDEX_Point_DAY, 1)
	if err != nil {
		panic(err)
	}

	if p1 == nil {
		panic(p1)
	}
	assert.Equal(t, p1.PreHash, h1)
	assert.Equal(t, p1.Hash, h2)
	for k, v := range p1.Sbps {
		assert.Equal(t, v, infos[k])
	}
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
