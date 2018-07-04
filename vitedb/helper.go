package vitedb

import (
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
	"errors"
	"strings"
	"encoding/hex"
)

// DBK = database key, DBKP = database key prefix
var (
	DBK_DOT = []byte(".")

	DBKP_ACCOUNTID_INDEX = []byte("j")

	DBKP_ACCOUNTBLOCKMETA = []byte("b")

	DBKP_ACCOUNTBLOCK = []byte("c")

	DBKP_SNAPSHOTBLOCKHASH = []byte("d")

	DBKP_SNAPSHOTBLOCK = []byte("e")

	DBKP_TOKENNAME_INDEX = []byte("f")

	DBKP_TOKENSYMBOL_INDEX = []byte("g")

	DBKP_TOKENID_INDEX = []byte("h")

	DBKP_SNAPSHOTTIMESTAMP_INDEX = []byte("i")

	DBKP_ACCOUNTMETA = []byte("a")
)


func createKey (keyPartionList... interface{}) ([]byte, error){
	key := []byte{}
	len := len(keyPartionList)

	// Temporary: converting ascii code of hex string to bytes, takes up twice as much space,
	// to avoid dot ascii code appear that are separator of leveldb key.
	for index, keyPartion := range keyPartionList {
		var bytes []byte

		switch keyPartion.(type) {
		case string:
			keyPartionString := keyPartion.(string)
			if strings.Contains(keyPartionString, ".") {
				return nil, errors.New("createKey failed. Key must not contains dot(\".\")")
			}
			bytes = []byte(keyPartionString)

		case *big.Int:
			hex.Encode(bytes, keyPartion.(*big.Int).Bytes())
			bytes = append(DBK_DOT, bytes...)
		default:
			return nil, errors.New("createKey failed. Key must be big.Int or string type")
		}

		key = append(key, bytes...)
		if index < len - 1 {
			key = append(key, DBK_DOT...)
		}
	}

	return key, nil
}

type batchContext struct {
	Batch *leveldb.Batch
	db *leveldb.DB
	isWrite bool
}

func (batchContext) New (batch *leveldb.Batch, db *leveldb.DB) *batchContext {
	isWrite := false
	if batch == nil {
		batch = new(leveldb.Batch)
		isWrite = true
	}


	return &batchContext {
		Batch: batch,
		db: db,
		isWrite: isWrite,
	}
}

func (ctx *batchContext) Quit () error{
	if (ctx.isWrite) {
		return ctx.db.Write(ctx.Batch, nil)
	}
	return nil
}


func batchWrite (batch *leveldb.Batch, db * leveldb.DB, writeFunc func(*batchContext) error) error {
	ctx := batchContext{}.New(batch, db)
	if err := writeFunc(ctx); err != nil {
		return err
	}

	return ctx.Quit()

}