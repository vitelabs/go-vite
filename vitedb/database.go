package vitedb

import (
	"encoding/hex"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"log"
	"math/big"
	"path/filepath"
)

type DataBase struct {
	filename string

	Leveldb *leveldb.DB
}

var ldbDataBaseCache = map[string]*DataBase{}

type viteComparer struct{}

func (*viteComparer) Name() string {
	return "vite.cmp.v1"
}

func (*viteComparer) Separator(dst, a, b []byte) []byte {
	return comparer.DefaultComparer.Separator(dst, a, b)
}

func (*viteComparer) Successor(dst, b []byte) []byte {
	return comparer.DefaultComparer.Successor(dst, b)
}

func GetBigInt(src []byte) *big.Int {

	bigIntBytes := make([]byte, hex.DecodedLen(len(src)))

	hex.Decode(bigIntBytes, src)

	bigInt := &big.Int{}
	bigInt.SetBytes(bigIntBytes)
	return bigInt
}

func (*viteComparer) Compare(a, b []byte) (result int) {

	lenA := len(a)
	lenB := len(b)

	aCurrentState := 0 // 0 means byte comparing, 1 means bigInt comparing
	bCurrentState := 0 // 0 means byte comparing, 1 means bigInt comparing

	aIndex := 0
	bIndex := 0

	var aBigIntBytes []byte
	var bBigIntBytes []byte

	for {
		if aCurrentState == 0 {
			if bCurrentState == 0 {
				if aIndex >= lenA && bIndex >= lenB {
					return 0
				} else if bIndex >= lenB {
					return 1
				} else if aIndex >= lenA {
					return -1
				}

				aByte := a[aIndex]
				bByte := b[bIndex]

				if aByte == DBK_BIGINT[0] {
					aCurrentState = 1
				}

				if bByte == DBK_BIGINT[0] {
					bCurrentState = 1
				}

				if aByte > bByte {
					return 1
				}

				if aByte < bByte {
					return -1
				}

				aIndex++
				bIndex++
			} else {
				return 1
			}
		} else {
			if bCurrentState == 0 {
				return -1
			} else {
				aIsEnd := aIndex == lenA
				bIsEnd := bIndex == lenB

				var aByte, bByte byte
				if !aIsEnd {
					aByte = a[aIndex]

					if aByte != DBK_DOT[0] &&
						aByte != DBK_DOT[0]+1 {
						aBigIntBytes = append(aBigIntBytes, aByte)
						aIndex++
					}
				}

				if !bIsEnd {
					bByte = b[bIndex]

					if bByte != DBK_DOT[0] &&
						bByte != DBK_DOT[0]+1 {
						bBigIntBytes = append(bBigIntBytes, bByte)
						bIndex++
					}
				}

				if (aIsEnd || aByte == DBK_DOT[0] || aByte == DBK_DOT[0]+1) &&
					(bIsEnd || bByte == DBK_DOT[0] || bByte == DBK_DOT[0]+1) {

					aBigInt := GetBigInt(aBigIntBytes)
					bBigInt := GetBigInt(bBigIntBytes)

					bigIntCmpResult := aBigInt.Cmp(bBigInt)
					if bigIntCmpResult != 0 {
						return bigIntCmpResult
					}

					aBigIntBytes = []byte{}
					bBigIntBytes = []byte{}

					aCurrentState = 0
					bCurrentState = 0

					continue
				}

			}
		}
	}

	return 0
}

var (
	DB_DIR    = ""
	DB_LEDGER = "ledger"
)

func SetDataDir(dataDir string) {
	DB_DIR = dataDir
}

func GetLDBDataBase(file string) (*DataBase, error) {
	if _, ok := ldbDataBaseCache[file]; !ok {
		cmp := new(viteComparer)
		options := &opt.Options{
			Comparer: cmp,
		}
		db, err := leveldb.OpenFile(filepath.Join(DB_DIR, file), options)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		dataBase := &DataBase{
			filename: file,
			Leveldb:  db,
		}

		ldbDataBaseCache[file] = dataBase
	}

	return ldbDataBaseCache[file], nil
}
