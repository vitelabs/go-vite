package disk_usage_analysis

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/chain_db/database"
	"testing"
)

func Test_overview(t *testing.T) {
	chainDb := newChainDb("testdata")
	const (
		PRINT_PER_ACCOUNTBLOCK  = 0
		PRINT_PER_SNAPSHOTBLOCK = 0
	)

	var totalCount, totalKeyUsage, totalValueUsage uint64
	addTotalCountUsage := func(count uint64, keyUsage uint64, valueUsage uint64) {
		totalCount += count
		totalKeyUsage += keyUsage
		totalValueUsage += valueUsage
	}

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ACCOUNTID_INDEX, "DBKP_ACCOUNTID_INDEX", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ACCOUNT, "DBKP_ACCOUNT", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ACCOUNTBLOCKMETA, "DBKP_ACCOUNTBLOCKMETA", PRINT_PER_ACCOUNTBLOCK))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ACCOUNTBLOCK, "DBKP_ACCOUNTBLOCK", PRINT_PER_ACCOUNTBLOCK))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_SNAPSHOTBLOCKHASH, "DBKP_SNAPSHOTBLOCKHASH", PRINT_PER_SNAPSHOTBLOCK))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_SNAPSHOTBLOCK, "DBKP_SNAPSHOTBLOCK", PRINT_PER_SNAPSHOTBLOCK))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_SNAPSHOTCONTENT, "DBKP_SNAPSHOTCONTENT", PRINT_PER_SNAPSHOTBLOCK))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ONROADMETA, "DBKP_ONROADMETA", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ONROADRECEIVEERR, "DBKP_ONROADRECEIVEERR", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ACCOUNTBLOCK_COUNTER, "DBKP_ACCOUNTBLOCK_COUNTER", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_TRIE_NODE, "DBKP_TRIE_NODE", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_TRIE_REF_VALUE, "DBKP_TRIE_REF_VALUE", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_LOG_LIST, "DBKP_LOG_LIST", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_ADDR_GID, "DBKP_ADDR_GID", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_GID_ADDR, "DBKP_GID_ADDR", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_BLOCK_EVENT, "DBKP_BLOCK_EVENT", 0))

	addTotalCountUsage(countUsage(chainDb, database.DBKP_BE_SNAPSHOT, "DBKP_BE_SNAPSHOT", 0))

	fmt.Printf("[totalUsage] totalUsage: %d KB, keyUsage: %d KB, valueUsage: %d KB, count: %d\n", (totalKeyUsage+totalValueUsage)/1024, totalKeyUsage/1024, totalValueUsage/1024, totalCount)
}

func countUsage(chainDb *chain_db.ChainDb, prefix byte, name string, printInterval uint64) (uint64, uint64, uint64) {
	iter := chainDb.Db().NewIterator(util.BytesPrefix([]byte{prefix}), nil)
	defer iter.Release()
	var keyUsage uint64
	var valueUsage uint64
	var count uint64
	for iter.Next() {
		count++
		keyUsage += uint64(len(iter.Key()))
		valueUsage += uint64(len(iter.Value()))
		if printInterval > 0 && count%printInterval == 0 {
			fmt.Printf("[%s] keyUsage: %d KB, valueUsage: %d KB, count: %d\n", name, keyUsage/1024, valueUsage/1024, count)
		}
	}
	var keyPerUsage uint64
	var valuePerUsage uint64
	var perUsage uint64
	if count > 0 {
		keyPerUsage = keyUsage / count
		valuePerUsage = valueUsage / count
		perUsage = (keyUsage + valueUsage) / count
	}
	fmt.Printf("[%s] keyUsage: %d KB, valueUsage: %d KB, keyPerUsage: %d Byte, valuePerUsage: %d Byte, perUsage: %d Byte, count: %d\n", name, keyUsage/1024, valueUsage/1024, keyPerUsage, valuePerUsage, perUsage, count)
	return count, keyUsage, valueUsage
}
