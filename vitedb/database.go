package vitedb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"math/big"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"encoding/hex"
)

type DataBase struct {
	filename string

	Leveldb *leveldb.DB
}

const (
	DB_BLOCK = "vite_leveldb_database/block"
)


var ldbDataBaseCache = map[string]* DataBase{}

type viteComparer struct {}

func (*viteComparer) Name() string {
	return "vite.cmp.v1"
}

func (*viteComparer) Separator(dst, a, b []byte) []byte {
	return comparer.DefaultComparer.Separator(dst, a, b)
}

func (*viteComparer) Successor (dst, b []byte) []byte {
	return comparer.DefaultComparer.Successor(dst, b)
}

func GetBigIntBytesList (key []byte) [][]byte {
	var temp, tempKey []byte
	var bigIntBytesList [][]byte
	keyMaxIndex := len(key) - 1
	for index, oneByte := range key {
		tempLength := len(temp)
		if oneByte == DBK_UNDERLINE[0] || index == keyMaxIndex {
			if tempLength == 1 {
				var bigIntBytes []byte
				hex.Decode(bigIntBytes, tempKey)
				bigIntBytesList = append(bigIntBytesList, bigIntBytes)

				temp = nil
				tempKey = nil
			} else {
				temp = append(temp, oneByte)
			}
		} else if tempLength == 1 {
			tempKey = append(tempKey, oneByte)
		}
	}

	return bigIntBytesList
}

func cmpTwoBigInt (a []byte, b[]byte) int {
	aBigInt := &big.Int{}
	bBigInt := &big.Int{}

	aBigInt.SetBytes(a)
	bBigInt.SetBytes(b)
	return aBigInt.Cmp(bBigInt)
}

func (* viteComparer) Compare (a, b []byte) int {
	aBigIntBytesList, bBigIntBytesList:=  GetBigIntBytesList(a), GetBigIntBytesList(b)
	for index, aBigIntBytes := range aBigIntBytesList {
		result := cmpTwoBigInt(aBigIntBytes, bBigIntBytesList[index])
		if result != 0 {
			return result
		}
	}

	if aBigIntBytesList == nil {
		return comparer.DefaultComparer.Compare(a, b)
	}

	return 0
}


func GetLDBDataBase (file string) ( *DataBase, error ){
	if _, ok := ldbDataBaseCache[file]; !ok {
		cmp := new(viteComparer)
		options := &opt.Options {
			Comparer: cmp,
		}
		db, err := leveldb.OpenFile(file, options)
		if err != nil {
			log.Println(err)
			return  nil, err
		}

		dataBase := &DataBase{
			filename: file,
			Leveldb: db,
		}

		ldbDataBaseCache[file] = dataBase
	}

	return ldbDataBaseCache[file], nil
}

