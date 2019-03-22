package chain

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm_context"
)

type blackBlock struct {
	log    log15.Logger
	chain  *chain
	isOpen bool
}

func NewBlackBlock(chain *chain, isOpen bool) *blackBlock {
	bb := &blackBlock{
		chain:  chain,
		log:    log15.New("module", "blackBlockLog"),
		isOpen: isOpen,
	}

	if bb.isOpen {
		bb.log.SetHandler(log15.LvlFilterHandler(log15.LvlInfo, log15.Must.FileHandler(filepath.Join(chain.globalCfg.RunLogDir(), "vite_black_block.log"), log15.TerminalFormat())))
	}
	return bb
}

func (bb *blackBlock) recordNeedSnapshotCache() {
	if !bb.isOpen {
		return
	}
	content := bb.chain.GetNeedSnapshotContent()
	for addr, hashHeight := range content {
		bb.log.Info(fmt.Sprintf("%s: %d %s", addr, hashHeight.Height, hashHeight.Hash), "method", "recordNeedSnapshotCache")
	}
}

func (bb *blackBlock) InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) {
	if !bb.isOpen {
		return
	}
	for _, vmAccountBlock := range vmAccountBlocks {
		bb.log.Info(fmt.Sprintf("%+v", vmAccountBlock.AccountBlock), "method", "InsertAccountBlocks")
	}
	bb.recordNeedSnapshotCache()
}

func (bb *blackBlock) InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) {
	if !bb.isOpen {
		return
	}
	for _, snapshotBlock := range snapshotBlocks {
		content := snapshotBlock.SnapshotContent
		var contentString []byte
		if len(content) > 0 {
			contentString, _ = json.Marshal(content)
		}

		bb.log.Info(fmt.Sprintf("height: %d, prevHash: %s, hash: %s, content: %s", snapshotBlock.Height, snapshotBlock.PrevHash, snapshotBlock.Hash, contentString),
			"method", "InsertSnapshotBlocks")
	}
	bb.recordNeedSnapshotCache()
}

func (bb *blackBlock) DeleteSnapshotBlock(snapshotBlocks []*ledger.SnapshotBlock, subLedger map[types.Address][]*ledger.AccountBlock) {
	if !bb.isOpen {
		return
	}

	for _, snapshotBlock := range snapshotBlocks {
		content := snapshotBlock.SnapshotContent
		var contentString []byte
		if len(content) > 0 {
			contentString, _ = json.Marshal(content)
		}
		bb.log.Info(fmt.Sprintf("height: %d, prevHash: %s, hash: %s, content: %s", snapshotBlock.Height, snapshotBlock.PrevHash, snapshotBlock.Hash, contentString),
			"method", "DeleteSnapshotBlock.snapshotBlock")
	}

	for _, blocks := range subLedger {
		for _, block := range blocks {
			bb.log.Info(fmt.Sprintf("%+v", block), "method", "DeleteSnapshotBlock.accountBlock")
		}
	}

	bb.recordNeedSnapshotCache()

	if !bb.chain.CheckNeedSnapshotCache(bb.chain.GetNeedSnapshotContent()) {
		bb.log.Error("check failed!!!")

		unconfirmedSubLedger, getSubLedgerErr := bb.chain.GetUnConfirmedSubLedger()
		if getSubLedgerErr != nil {
			bb.log.Crit("getUnConfirmedSubLedger failed, error is "+getSubLedgerErr.Error(), "method", "printCorrectCacheMap")
		}
		for addr, blocks := range unconfirmedSubLedger {
			bb.log.Info(fmt.Sprintf("%s: %d %s", addr, blocks[len(blocks)-1].Height, blocks[len(blocks)-1].Hash), "method", "DeleteSnapshotBlock.correctNeedSnapshotCache")
		}
	}
	bb.log.Info("DeleteSnapshotBlock finish")
}

func (bb *blackBlock) DeleteAccountBlock(subLedger map[types.Address][]*ledger.AccountBlock) {
	if !bb.isOpen {
		return
	}
	for addr, blocks := range subLedger {
		bb.log.Info(addr.String())
		for _, block := range blocks {
			bb.log.Info(fmt.Sprintf("%+v", block), "method", "DeleteInvalidAccountBlocks")
		}
	}
	bb.recordNeedSnapshotCache()
}
