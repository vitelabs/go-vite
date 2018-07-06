package vitedb

import (
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
	"errors"
	"strings"
	"encoding/hex"
	"bytes"
)

// DBK = database key, DBKP = database key prefix
var (
	DBK_DOT = []byte(".")
	DBK_UNDERLINE = []byte("_")

	DBKP_ACCOUNTID_INDEX = "j"

	DBKP_ACCOUNTBLOCKMETA = "b"

	DBKP_ACCOUNTBLOCK = "c"

	DBKP_SNAPSHOTBLOCKHASH = "d"

	DBKP_SNAPSHOTBLOCK = "e"

	DBKP_TOKENNAME_INDEX = "f"

	DBKP_TOKENSYMBOL_INDEX = "g"

	DBKP_TOKENID_INDEX = "h"

	DBKP_SNAPSHOTTIMESTAMP_INDEX = "i"

	DBKP_ACCOUNTMETA = "a"
)


func createKey (keyPartionList... interface{}) ([]byte, error){
	key := []byte{}
	len := len(keyPartionList)

	// Temporary: converting ascii code of hex string to bytes, takes up twice as much space,
	// to avoid dot ascii code appear that are separator of leveldb key.
	for index, keyPartion := range keyPartionList {
		var bytes []byte

		switch keyPartion.(type) {
		case []byte:
			hex.Encode(bytes, keyPartion.([]byte))

		case string:
			keyPartionString := keyPartion.(string)
			if strings.Contains(keyPartionString, ".") {
				return nil, errors.New("CreateKey failed. Key must not contains dot(\".\")")
			}
			bytes = []byte(keyPartionString)

		case *big.Int:
			hex.Encode(bytes, keyPartion.(*big.Int).Bytes())
			bytes = append(DBK_UNDERLINE, bytes...)
		default:
			return nil, errors.New("CreateKey failed. Key must be big.Int or string type")
		}

		key = append(key, bytes...)
		if index < len - 1 {
			key = append(key, DBK_DOT...)
		}
	}

	return key, nil
}

func deserializeKey(key []byte) [][]byte  {
	bytesList := bytes.Split(key, DBK_DOT)
	var parsedBytesList [][]byte
	for i := 1; i < len(bytesList); i++ {
		bytes := bytesList[i]
		if bytes[0] == DBK_UNDERLINE[0] {
			// big.Int
			bytes = bytes[1:]
		}

		var parsedBytes []byte
		hex.Decode(parsedBytes, bytes)

		parsedBytesList = append(parsedBytesList, parsedBytes)
	}
	return parsedBytesList
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