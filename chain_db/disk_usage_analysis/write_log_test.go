package disk_usage_analysis

import (
	"encoding/binary"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/journal"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/node"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func Test_write_log(t *testing.T) {
	//chainDb := newChainDb("testdata")

	journalFileReader, err := os.Open(path.Join(node.DefaultDataDir(), "ledger_test/testdata/ledger/005876.log"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = read(journalFileReader)
	if err != nil {
		t.Fatal(err)
	}

}

type keyType uint

const (
	keyTypeDel = keyType(0)
	keyTypeVal = keyType(1)

	keyMaxSeq = (uint64(1) << 56) - 1
)

type batchIndex struct {
	keyType            keyType
	keyPos, keyLen     int
	valuePos, valueLen int
}

func (index batchIndex) k(data []byte) []byte {
	return data[index.keyPos : index.keyPos+index.keyLen]
}

func (index batchIndex) v(data []byte) []byte {
	if index.valueLen != 0 {
		return data[index.valuePos : index.valuePos+index.valueLen]
	}
	return nil
}

func (index batchIndex) kv(data []byte) (key, value []byte) {
	return index.k(data), index.v(data)
}

const (
	batchHeaderLen = 8 + 4
)

type internalKey []byte

func ensureBuffer(b []byte, n int) []byte {
	if cap(b) < n {
		return make([]byte, n)
	}
	return b[:n]
}

func makeInternalKey(dst, ukey []byte, seq uint64, kt keyType) internalKey {
	if seq > keyMaxSeq {
		panic("leveldb: invalid sequence number")
	} else if kt > keyTypeVal {
		panic("leveldb: invalid type")
	}

	dst = ensureBuffer(dst, len(ukey)+8)
	copy(dst, ukey)
	binary.LittleEndian.PutUint64(dst[len(ukey):], (seq<<8)|uint64(kt))
	return internalKey(dst)
}

func decodeBatchHeader(data []byte) (seq uint64, batchLen int, err error) {
	if len(data) < batchHeaderLen {
		return 0, 0, errors.New("too short")
	}

	seq = binary.LittleEndian.Uint64(data)
	batchLen = int(binary.LittleEndian.Uint32(data[8:]))
	if batchLen < 0 {
		return 0, 0, errors.New("invalid records length")
	}
	return
}

func decodeBatch(data []byte, fn func(i int, index batchIndex) error) error {
	var index batchIndex
	for i, o := 0, 0; o < len(data); i++ {
		// Key type.
		index.keyType = keyType(data[o])
		if index.keyType > keyTypeVal {
			return errors.New(fmt.Sprintf("bad record: invalid type %#x", uint(index.keyType)))
		}
		o++

		// Key.
		x, n := binary.Uvarint(data[o:])
		o += n
		if n <= 0 || o+int(x) > len(data) {
			return errors.New("bad record: invalid key length")
		}
		index.keyPos = o
		index.keyLen = int(x)
		o += index.keyLen

		// Value.
		if index.keyType == keyTypeVal {
			x, n = binary.Uvarint(data[o:])
			o += n
			if n <= 0 || o+int(x) > len(data) {
				return errors.New("bad record: invalid value length")
			}
			index.valuePos = o
			index.valueLen = int(x)
			o += index.valueLen
		} else {
			index.valuePos = 0
			index.valueLen = 0
		}

		if err := fn(i, index); err != nil {
			return err
		}
	}
	return nil
}

type MemoryDb struct {
}

func (db *MemoryDb) Put(key []byte, value []byte) error {
	firstKey := key[0]
	switch firstKey {
	case database.DBKP_ACCOUNTBLOCKMETA:
		fmt.Println("===account block meta===")
		//fmt.Printf("key(%d): %+v \n value(%d): %+v\n", len(key), key, len(value), value)
		hash, err := types.BytesToHash(key[1:])
		if err != nil {
			return err
		}
		fmt.Printf("hash: %s\n", hash)
		abm := &ledger.AccountBlockMeta{}
		if err := abm.Deserialize(value); err != nil {
			return err
		}
		fmt.Printf("data: %+v\n", abm)

		fmt.Printf("===account block meta===\n\n")
	case database.DBKP_ACCOUNTBLOCK:
		fmt.Println("===account block===")
		//fmt.Printf("key(%d): %+v \n value(%d): %+v\n", len(key), key, len(value), value)
		hash, err := types.BytesToHash(key[17:])
		if err != nil {
			return err
		}
		fmt.Printf("hash: %s\n", hash)

		abm := &ledger.AccountBlock{}
		if err := abm.Deserialize(value); err != nil {
			return err
		}
		fmt.Printf("data: %+v\n", abm)

		fmt.Printf("===account block===\n\n")
	case database.DBKP_SNAPSHOTCONTENT:
		fmt.Println("===snapshot block content===")
		sbc := ledger.SnapshotContent{}
		if err := sbc.Deserialize(value); err != nil {
			return err
		}
		fmt.Println("data:")
		for addr, hashHeight := range sbc {
			fmt.Printf("%s: %d, %s\n", addr, hashHeight.Height, hashHeight.Hash)

		}
		fmt.Printf("===snapshot block content===\n\n")
	}

	return nil
}
func mockMdb() *MemoryDb {
	return &MemoryDb{}
}

func read(r io.Reader) ([]string, error) {
	//var ss []string
	journals := journal.NewReader(r, nil, true, true)
	mdb := mockMdb()
	for {
		j, err := journals.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(j)
		if err != nil {
			return nil, err
		}

		_, batchLen, err := decodeBatchHeader(data)
		if err != nil {
			return nil, err
		}

		data = data[batchHeaderLen:]
		//var ik []byte
		var decodedLen int
		err = decodeBatch(data, func(i int, index batchIndex) error {
			if i >= batchLen {
				return errors.New("invalid records length")
			}
			//ik = makeInternalKey(ik, index.k(data), seq+uint64(i), index.keyType)
			if err := mdb.Put(index.k(data), index.v(data)); err != nil {
				return err
			}
			decodedLen++
			return nil
		})
		if err == nil && decodedLen != batchLen {
			err = errors.New(fmt.Sprintf("invalid records length: %d vs %d", batchLen, decodedLen))
			return nil, err
		}

		//ss = append(ss, string(s))
	}
	return nil, nil
}
