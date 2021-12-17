package dex

import (
	"github.com/vitelabs/go-vite/v2/interfaces"
)

type levelDbBook struct {
	db       interfaces.VmDb
	marketId int32
	side     bool
	iterator interfaces.StorageIterator
}

func getMakerBook(db interfaces.VmDb, marketId int32, side bool) (book *levelDbBook, err error) {
	book = &levelDbBook{db: db, marketId: marketId, side: side}
	if book.iterator, err = db.NewStorageIterator(getBookPrefix(book)); err != nil {
		panic(err)
	}
	return
}

func (book *levelDbBook) nextOrder() (order *Order, ok bool) {
	if ok = book.iterator.Next(); !ok {
		if book.iterator.Error() != nil {
			panic(book.iterator.Error())
		}
		return
	}
	orderId := book.iterator.Key()
	orderData := book.iterator.Value()
	if len(orderId) != OrderIdBytesLength || len(orderData) == 0 {
		panic(IterateVmDbFailedErr)
	}
	order = &Order{}
	if err := order.DeSerializeCompact(orderData, orderId); err != nil {
		panic(err)
	}
	return
}

func (book *levelDbBook) release() {
	book.iterator.Release()
}

func getBookPrefix(book *levelDbBook) []byte {
	if book.side {
		return append(Uint32ToBytes(uint32(book.marketId))[1:], byte(int8(1)))
	} else {
		return append(Uint32ToBytes(uint32(book.marketId))[1:], byte(int8(0)))
	}
}
