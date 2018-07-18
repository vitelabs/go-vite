package handler

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"log"
	"github.com/vitelabs/go-vite/crypto"
	"errors"
	"time"
)

type AccountChain struct {
	vite Vite
	// Handle block
	acAccess *access.AccountChainAccess
	aAccess *access.AccountAccess
}

func NewAccountChain (vite Vite) (*AccountChain) {
	return &AccountChain{
		vite: vite,
		acAccess: access.GetAccountChainAccess(),
		aAccess: access.GetAccountAccess(),
	}
}

// HandleBlockHash
func (ac *AccountChain) HandleGetBlocks (msg *protocols.GetAccountBlocksMsg, peer *protocols.Peer) error {
	go func() {
		blocks, err := ac.acAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			log.Println(err)
			return
		}
		// send out
		ac.vite.Pm().SendMsg(peer, &protocols.Msg{
			Code: protocols.AccountBlocksMsgCode,
			Payload: blocks,
		})
	}()
	return nil
}

// HandleBlockHash
func (ac *AccountChain) HandleSendBlocks (msg protocols.AccountBlocksMsg, peer *protocols.Peer) error {
	go func() {
		for _, block := range msg {
			// Verify signature
			isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)

			if verifyErr != nil {
				log.Println(verifyErr)
				continue
			}

			if !isVerified {
				continue
			}

			// Write block
			writeErr := ac.acAccess.WriteBlock(block)
			if writeErr != nil {
				log.Println(writeErr)
				continue
			}
		}
	}()
	return nil
}


// AccAddr = account address
func (ac *AccountChain) GetAccountByAccAddr (addr *types.Address) (*ledger.Account){
	return nil
}

// AccAddr = account address
func (ac *AccountChain) GetBlocksByAccAddr (addr *types.Address, index, num, count int) (ledger.AccountBlockList){
	return nil
}

func (ac *AccountChain) CreateTx (addr *types.Address, block *ledger.AccountBlock) (error) {
	accountMeta, err := ac.aAccess.GetAccountMeta(addr)

	if err != nil {
		return err
	}

	if accountMeta == nil {
		return errors.New("CreateTx failed, because account " + addr.String() + " is not existed.")
	}

	block.AccountAddress = addr

	// Set Snapshot Timestamp
	block.Timestamp = uint64(time.Now().Unix())

	// Set Meta: Height, Status, AccountId

	// Set Timestamp


	// Set Pow params: Nounce、Difficulty

	// Set Balance or Amount

	// Set Hash
}

func (ac *AccountChain) CreateTxWithPassphrase (addr *types.Address, passphrase string, block *ledger.AccountBlock) {

}