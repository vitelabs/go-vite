package vitedb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"fmt"
)

type DataBase struct {
	filename string
	db *leveldb.DB
}

const (
	DB_BLOCK = "vite_leveldb_database/block"
)



func NewDataBase (file string) ( *DataBase, error ){
	db, err := leveldb.OpenFile(file, nil)
	if err != nil {
		fmt.Println(err)
		return  nil, err
	}

	return &DataBase{
		filename: file,
		db: db,
	}, nil
}


var dataBaseCache = map[string]* DataBase{}

func GetDataBase (file string) (*DataBase) {
	if dataBase, ok := dataBaseCache[file]; ok {
		return dataBase
	}

	dataBase, _ := NewDataBase(file)
	dataBaseCache[file] = dataBase

	return dataBase
}

func (db *DataBase) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *DataBase) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *DataBase) Get(key []byte) ([]byte, error) {
	data, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return data, err
}

func (db *DataBase) Close () {
	err := db.db.Close()
	if err != nil {
		fmt.Println("Failed to close database, err: ", err)
	}
}


