package vitedb

import (
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
)

// DBK = database key, DBKP = database key prefix
var (
	DBK_DOT = []byte(".")


	DBKP_ACCOUNTID_INDEX = []byte("j")

	DBKP_ACCOUNTBLOCKMETA = []byte("b")

	DBKP_ACCOUNTBLOCK = []byte("c")

	DBKP_SNAPSHOTBLOCK = []byte("e")

	DBKP_TOKENNAME_INDEX = []byte("f")

	DBKP_TOKENSYMBOL_INDEX = []byte("g")

	DBKP_TOKENID_INDEX = []byte("h")

	DBKP_SNAPSHOTTIMESTAMP_INDEX = []byte("i")
)


func createKey (keyPartionList... interface{}) []byte {
	key := []byte{}
	len := len(keyPartionList)
	for index, keyPartion := range keyPartionList {
		var bytes []byte

		switch keyPartion.(type) {
		case []byte:
			bytes = keyPartion.([]byte)

		case *big.Int:
			bytes = append(DBK_DOT, keyPartion.(*big.Int).Bytes()...)

		}

		key = append(key, bytes...)
		if index < len - 1 {
			key = append(key, DBK_DOT...)
		}
	}

	return key
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