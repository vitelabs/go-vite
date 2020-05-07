package db

type Database interface {
	Put(key []byte, val []byte)
	Get(key []byte)
	Del(key []byte)
}
