package handler

import (
	"bytes"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/access"
	"github.com/vitelabs/go-vite/log"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
	"time"
)

type AccountChain struct {
	vite Vite
	// Handle block
	acAccess *access.AccountChainAccess
	aAccess  *access.AccountAccess
	scAccess *access.SnapshotChainAccess
	uAccess  *access.UnconfirmedAccess
	tAccess  *access.TokenAccess
}

func NewAccountChain(vite Vite) *AccountChain {
	return &AccountChain{
		vite:     vite,
		acAccess: access.GetAccountChainAccess(),
		aAccess:  access.GetAccountAccess(),
		scAccess: access.GetSnapshotChainAccess(),
		uAccess:  access.GetUnconfirmedAccess(),
		tAccess:  access.GetTokenAccess(),
	}
}

// HandleBlockHash
func (ac *AccountChain) HandleGetBlocks(msg *protoTypes.GetAccountBlocksMsg, peer *protoTypes.Peer) error {
	go func() {
		blocks, err := ac.acAccess.GetBlocksFromOrigin(&msg.Origin, msg.Count, msg.Forward)
		if err != nil {
			log.Info(err.Error())
			return
		}

		// send out
		ac.vite.Pm().SendMsg(peer, &protoTypes.Msg{
			Code:    protoTypes.AccountBlocksMsgCode,
			Payload: blocks,
		})
	}()
	return nil
}

// HandleBlockHash
func (ac *AccountChain) HandleSendBlocks(msg *protoTypes.AccountBlocksMsg, peer *protoTypes.Peer) error {
	go func() {
		globalRWMutex.RLock()
		defer globalRWMutex.RUnlock()

		log.Info("AccountChain HandleSendBlocks: receive blocks from network")
		for _, block := range *msg {
			log.Info("AccountChain HandleSendBlocks: start process block " + block.Hash.String())
			if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
				// Discard the block.
				log.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because block.PublicKey or block.Hash or block.Signature is nil.")
				continue
			}
			// Verify hash
			computedHash, err := block.ComputeHash()
			if err != nil {
				// Discard the block.
				log.Info(err.Error())
				continue
			}

			if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
				// Discard the block.
				log.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because the computed hash is " + computedHash.String() + " and the block hash is " + block.Hash.String())
				continue
			}
			// Verify signature
			isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)

			if verifyErr != nil || !isVerified {
				// Discard the block.
				log.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because verify signature failed.")
				continue
			}

			// Write block
			writeErr := ac.acAccess.WriteBlock(block, nil)

			if writeErr != nil {

				switch writeErr.(type) {
				case *access.AcWriteError:
					err := writeErr.(*access.AcWriteError)
					if err.Code == access.WacPrevHashUncorrectErr {
						log.Info("AccountChain HandleSendBlocks: start download account chain.")
						errData := err.Data.(*ledger.AccountBlock)

						currentHeight := big.NewInt(0)
						if errData != nil {
							currentHeight = errData.Meta.Height
						}

						if block.Meta.Height.Cmp(currentHeight) <= 0 {
							return
						}
						// Download fragment
						count := &big.Int{}
						count.Sub(block.Meta.Height, currentHeight)
						ac.vite.Pm().SendMsg(peer, &protoTypes.Msg{
							Code: protoTypes.GetAccountBlocksMsgCode,
							Payload: &protoTypes.GetAccountBlocksMsg{
								Origin:  *errData.Hash,
								Forward: true,
								Count:   count.Uint64(),
							},
						})
						return
					}
				}

				log.Info(writeErr.Error())
				continue
			} else {
				log.Info("AccountChain HandleSendBlocks: write block " + block.Hash.String() + " success.")
			}
		}
	}()
	return nil
}

// AccAddr = account address
func (ac *AccountChain) GetAccountByAccAddr(addr *types.Address) (*ledger.AccountMeta, error) {
	return ac.aAccess.GetAccountMeta(addr)
}

// AccAddr = account address
func (ac *AccountChain) GetBlocksByAccAddr(addr *types.Address, index, num, count int) (ledger.AccountBlockList, error) {
	return ac.acAccess.GetBlockListByAccountAddress(index, num, count, addr)
}

func (ac *AccountChain) CreateTx(block *ledger.AccountBlock) error {
	return ac.CreateTxWithPassphrase(block, "")
}

func (ac *AccountChain) CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error {
	if !syncInfo.IsFirstSyncDone {
		log.Error("Sync unfinished, so can't handleSendBlocks.")
		return nil
	}
	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()

	accountMeta, err := ac.aAccess.GetAccountMeta(block.AccountAddress)

	if err != nil {
		return err
	}

	if accountMeta == nil {
		return errors.New("CreateTx failed, because account " + block.AccountAddress.String() + " is not existed.")
	}

	// Set prevHash
	latestBlock, err := ac.acAccess.GetLatestBlockByAccountAddress(block.AccountAddress)
	if err != nil {
		return err
	}

	if latestBlock != nil {
		block.PrevHash = latestBlock.Hash
	}

	// Set Snapshot Timestamp
	currentSnapshotBlock, err := ac.scAccess.GetLatestBlock()
	if err != nil {
		return err
	}

	block.SnapshotTimestamp = currentSnapshotBlock.Hash

	// Set Timestamp
	block.Timestamp = uint64(time.Now().Unix())

	// Set Pow params: Nounce、Difficulty
	block.Nounce = []byte{0, 0, 0, 0, 0}
	block.Difficulty = []byte{0, 0, 0, 0, 0}
	block.FAmount = big.NewInt(0)

	// Set PublicKey
	block.PublicKey = accountMeta.PublicKey

	writeErr := ac.acAccess.WriteBlock(block, func(accountBlock *ledger.AccountBlock) (*ledger.AccountBlock, error) {
		var signErr error
		if passphrase == "" {
			accountBlock.Signature, accountBlock.PublicKey, signErr =
				ac.vite.WalletManager().KeystoreManager.SignData(*block.AccountAddress, block.Hash.Bytes())

		} else {
			accountBlock.Signature, accountBlock.PublicKey, signErr =
				ac.vite.WalletManager().KeystoreManager.SignDataWithPassphrase(*block.AccountAddress, passphrase, block.Hash.Bytes())

		}

		return accountBlock, signErr
	})

	if err != nil {
		return writeErr
	}

	// Broadcast
	sendErr := ac.vite.Pm().SendMsg(nil, &protoTypes.Msg{
		Code:    protoTypes.AccountBlocksMsgCode,
		Payload: &protoTypes.AccountBlocksMsg{block},
	})

	if sendErr != nil {
		log.Info("CreateTx broadcast failed, error is " + sendErr.Error())
		return sendErr
	}
	return nil
}
