package chain_index

//func (iDB *IndexDB) DeleteTo(endLocation *chain_file_manager.Location) error {
//
//	batch := iDB.store.NewBatch()
//	iter := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountIdPrefixKey()))
//	defer iter.Release()
//
//	// delete account blocks
//	for iter.Next() {
//		value := iter.Value()
//
//		addr, err := types.BytesToAddress(value)
//		if err != nil {
//			return err
//		}
//
//		heightIter := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountBlockHeightPrefixKey(&addr)))
//
//		iterOk := heightIter.Last()
//		for iterOk {
//			valueBytes := heightIter.Value()
//
//			location := chain_utils.DeserializeLocation(valueBytes[types.HashSize:])
//			if location.Compare(endLocation) < 0 {
//				break
//			}
//
//			height := chain_utils.BytesToUint64(heightIter.Key()[1+types.AddressSize:])
//
//			hash, err := types.BytesToHash(valueBytes[:types.HashSize])
//			if err != nil {
//				heightIter.Release()
//				return err
//			}
//
//			// rollback
//			iDB.deleteAccountBlock(batch, addr, hash, height)
//
//			iterOk = heightIter.Prev()
//		}
//
//		if err := heightIter.Error(); err != nil && err != leveldb.ErrNotFound {
//			heightIter.Release()
//			return err
//		}
//
//		heightIter.Release()
//	}
//
//	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
//		return err
//	}
//
//	snapshotIter := iDB.store.NewIterator(util.BytesPrefix([]byte{chain_utils.SnapshotBlockHeightKeyPrefix}))
//	defer snapshotIter.Release()
//
//	// delete snapshot blocks
//	for snapshotIter.Next() {
//		value := snapshotIter.Value()
//
//		location := chain_utils.DeserializeLocation(value[types.HashSize:])
//		if location.Compare(endLocation) < 0 {
//			break
//		}
//
//		height := chain_utils.BytesToUint64(iter.Key()[1:])
//		hash, err := types.BytesToHash(value[:types.HashSize])
//		if err != nil {
//			return err
//		}
//
//		// rollback
//		iDB.deleteSnapshotBlock(hash, height)
//	}
//
//	if err := snapshotIter.Error(); err != nil && err != leveldb.ErrNotFound {
//		return err
//	}
//
//	return nil
//}
//
//func (iDB *IndexDB) deleteSnapshotBlock(batch *leveldb.Batch, hash types.Hash, height uint64) {
//	iDB.deleteSnapshotBlockHeight(batch, height)
//	iDB.deleteSnapshotBlockHash(batch, hash)
//}
//
//func (iDB *IndexDB) deleteAccountBlock(batch *leveldb.Batch, addr types.Address, hash types.Hash, height uint64) {
//	iDB.deleteAccountBlockHash(batch, hash)
//
//	iDB.deleteAccountBlockHeight(batch, addr, height)
//
//	iDB.deleteReceive(batch, hash)
//
//	iDB.deleteOnRoad(batch, hash)
//
//	iDB.deleteConfirmHeight(batch, addr, height)
//}
