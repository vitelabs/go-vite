package handler

import (
	"bytes"
	"errors"
	"github.com/inconshreveable/log15"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/ledger/access"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
	"strconv"
	"time"
)

var acLog = log15.New("module", "ledger/handler/account_chain")

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
			acLog.Info(err.Error())
			return
		}

		// send out
		acLog.Info("AccountChain.HandleGetBlocks: send " + strconv.Itoa(len(blocks)) + " blocks.")
		ac.vite.Pm().SendMsg(peer, &protoTypes.Msg{
			Code:    protoTypes.AccountBlocksMsgCode,
			Payload: &blocks,
		})
	}()
	return nil
}

// HandleBlockHash
func (ac *AccountChain) HandleSendBlocks(msg *protoTypes.AccountBlocksMsg, peer *protoTypes.Peer) error {
	go func() {
		globalRWMutex.RLock()
		defer globalRWMutex.RUnlock()
		acLog.Info("AccountChain HandleSendBlocks: receive " + strconv.Itoa(len(*msg)) + " blocks from network")

		for _, block := range *msg {
			if block.PublicKey == nil || block.Hash == nil || block.Signature == nil {
				// Discard the block.
				acLog.Info("AccountChain HandleSendBlocks: discard block, because block.PublicKey or block.Hash or block.Signature is nil.")
				continue
			}
			// Verify hash
			computedHash, err := block.ComputeHash()

			if err != nil {
				// Discard the block.
				acLog.Info(err.Error())
				continue
			}

			if !bytes.Equal(computedHash.Bytes(), block.Hash.Bytes()) {
				// Discard the block.
				acLog.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because the computed hash is " + computedHash.String() + " and the block hash is " + block.Hash.String())
				continue
			}
			// Verify signature
			isVerified, verifyErr := crypto.VerifySig(block.PublicKey, block.Hash.Bytes(), block.Signature)

			if verifyErr != nil || !isVerified {
				// Discard the block.
				acLog.Info("AccountChain HandleSendBlocks: discard block " + block.Hash.String() + ", because verify signature failed.")
				continue
			}

			// Write block
			acLog.Info("AccountChain HandleSendBlocks: try write a block from network")
			writeErr := ac.acAccess.WriteBlock(block, nil)

			if writeErr != nil {
				acLog.Info("AccountChain HandleSendBlocks: Write error. Error is " + writeErr.Error())
				switch writeErr.(type) {
				case *access.AcWriteError:
					err := writeErr.(*access.AcWriteError)
					if err.Code == access.WacHigherErr {
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
						if count.Cmp(big.NewInt(1)) <= 0 {
							return
						}

						count.Add(count, big.NewInt(1))

						acLog.Info("AccountChain HandleSendBlocks: start download account chain. Current height is " +
							currentHeight.String() + ", and target height is " + block.Meta.Height.String())
						acLog.Info(err.Error())

						// Download accountblocks
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
			}
		}
		acLog.Info("AccountChain HandleSendBlocks: write " + strconv.Itoa(len(*msg)) + " blocks finish.")
	}()
	return nil
}

// AccAddr = account address
func (ac *AccountChain) GetAccountByAccAddr(addr *types.Address) (*ledger.AccountMeta, error) {
	return ac.aAccess.GetAccountMeta(addr)
}

// AccAddr = account address
func (ac *AccountChain) GetBlocksByAccAddr(addr *types.Address, index, num, count int) (ledger.AccountBlockList, *types.GetError) {
	return ac.acAccess.GetBlockListByAccountAddress(index, num, count, addr)
}

func (ac *AccountChain) CreateTx(block *ledger.AccountBlock) error {
	return ac.CreateTxWithPassphrase(block, "")
}

func (ac *AccountChain) CreateTxWithPassphrase(block *ledger.AccountBlock, passphrase string) error {
	syncInfo := ac.vite.Ledger().Sc().GetFirstSyncInfo()
	if !syncInfo.IsFirstSyncDone {
		acLog.Error("Sync unfinished, so can't create transaction.")
		return errors.New("sync unfinished, so can't create transaction")
	}

	globalRWMutex.RLock()
	defer globalRWMutex.RUnlock()

	accountMeta, err := ac.aAccess.GetAccountMeta(block.AccountAddress)

	if block.IsSendBlock() {
		if err != nil || accountMeta == nil {
			err := errors.New("CreateTx failed, because account " + block.AccountAddress.String() + " doesn't found.")
			acLog.Info(err.Error())
			return err
		}
	} else {
		if err != nil && err != leveldb.ErrNotFound {
			err := errors.New("AccountChain CreateTx: get account meta failed, error is " + err.Error())
			acLog.Info(err.Error())
			return err
		}
	}

	acLog.Info("AccountChain CreateTx: get account meta success.")

	// Set prevHash
	latestBlock, err := ac.acAccess.GetLatestBlockByAccountAddress(block.AccountAddress)

	if err != nil {
		return err
	}

	if latestBlock != nil {
		block.PrevHash = latestBlock.Hash
	}
	acLog.Info("AccountChain CreateTx: get latestBlock success.")

	// Set Snapshot Timestamp
	currentSnapshotBlock, err := ac.scAccess.GetLatestBlock()
	if err != nil {
		return err
	}

	acLog.Info("AccountChain CreateTx: get currentSnapshotBlock success.")
	block.SnapshotTimestamp = currentSnapshotBlock.Hash

	// Set Timestamp
	block.Timestamp = uint64(time.Now().Unix())

	// Set Pow params: Nounce、Difficulty
	block.Nounce = []byte{0, 0, 0, 0, 0}
	block.Difficulty = []byte{0, 0, 0, 0, 0}
	block.FAmount = big.NewInt(0)

	// Set PublicKey
	if accountMeta != nil {
		block.PublicKey = accountMeta.PublicKey
	}

	acLog.Info("AccountChain CreateTx: start write block.")
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

	if writeErr != nil {
		acLog.Info("AccountChain CreateTx: write block failed, error is " + writeErr.Error())
		return writeErr.(*access.AcWriteError).Err
	}

	acLog.Info("AccountChain CreateTx: write block success.")

	// Broadcast
	sendErr := ac.vite.Pm().SendMsg(nil, &protoTypes.Msg{
		Code:    protoTypes.AccountBlocksMsgCode,
		Payload: &protoTypes.AccountBlocksMsg{block},
	})

	acLog.Info("AccountChain CreateTx: broadcast to network.")

	if sendErr != nil {
		acLog.Info("CreateTx broadcast failed, error is " + sendErr.Error())
		return sendErr
	}
	return nil
}
