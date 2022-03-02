# Introduction

First introduce the purpose of each directory:

- **.github** is used to store the configuration related to github. Currently the configuration of the github action is stored in it
- **bin** some scripts that may be used during development and deployment
- **client** implements the go version client for remotely calling gvite nodes
- **cmd** command line script, the entry of gvite is here cmd/gvite/main.go
- **common** some common method components and constant definitions
- **conf** stores the configuration file, conf/node_config.json is the default on the main network
- **contracts-vite** stores vite-related smart contracts, contracts-vite/contracts/ViteToken.sol is the source code of the VITE ERC20 Token deployed on the eth network
- **crypto** implementation of encryption algorithm and hash algorithm ed25519 and blake2b-512 are the main algorithms on the vite chain
- **docker** stores the dockerfile, which is used when compiling with docker
- **docs** go-vite related wiki storage, link documentation https://docs.vite.org/go-vite/
- **interfaces** The interface definitions of some modules are placed here to solve the golang circular dependency problem
- **ledger** contains ledger related implementations
- **log15** log framework used in log15 go-vite was copied since many things were changed at the beginning. The original project address: https://github.com/inconshreveable/log15
- **monitor** toolkit written for the convenience of data statistics during stress testing
- **net** network layer implementation, including node discovery, node interoperability, ledger broadcast and pull
- **node** is an upper package of ledger, mainly doing some combined functions
- **pow** locally implemented pow calculation and remote call pow-client to calculate pow
- **producer** triggers the entry of snapshot block and contract block production
- **rpc** underlying implementation of the three remote calling methods of websocket/http/ipc
- **rpcapi** interface exposed by each module
- **smart-contract** is similar to contracts-vite
- **tools** are similar to collection, some collections are implemented
- **version** stores the version number of each compilation, the make script will modify the content of version/buildversion
- **vm** the realization of virtual machine and the realization of built-in contract, there are a lot of virtual machine execution logic in it
- **vm_db** the storage interface required by the virtual machine during execution is the encapsulation of the virtual machine's access to the chain
- **wallet** implementation of wallet, private key mnemonic management, and signature and verification signature

Then there is the overall framework of gvite's implementation:

It is mainly divided into the following major modules:
1. **net** network interconnection
2. **chain** underlying storage
3. **vm** execution of the virtual machine
4. **consensus** is responsible for the consensus block producer
5. **pool** is responsible for fork selection
6. **verifier** wraps various verification logics and is a wrapper for the vite protocol
7. **onroad** is responsible for the production of the contract receive block
8. **generator** aggregates the process of create block, which will be called by modules such as onroad and api

# data flow

## The user sends a transfer transaction

- rpcapi/api/ledger_v2.go#SendRawTransaction 
  - rpc is the entry point for all access nodes
- ledger/pool/pool.go#AddDirectAccountBlock 
  - Insert the block directly into the ledger through the pool
- ledger/verifier/verifier.go#VerifyPoolAccountBlock 
  - call verifier to verify the validity of the block
- ledger/chain/insert.go#InsertAccountBlock 
  - calls the interface of chain to insert into the chain
- ledger/chain/index/insert.go#InsertAccountBlock 
  - maintains the hash relationship of blocks, such as the relationship between height and hash, the relationship between send-receive, etc.
  - involving storage files ledger/index/* and ledger/blocks
- ledger/chain/state/write.go#Write 
  - maintains the state changed by the block, such as balance, storage, etc., involving the storage file ledger/state

## Generate a receive block for a contract

- ledger/consensus/trigger.go#update 
  - generates an event that the contract starts to generate blocks and passes it to the downstream
- producer/producer.go#producerContract 
  - receives consensus layer messages and passes them to onroad
- ledger/onroad/manager.go#producerStartEventFunc 
  - onroad starts the coroutine internally, and processes the send block to be generated separately
- ledger/onroad/contract.go#ContractWorker.Start 
  - sorts all contracts according to the quota, and generates blocks in sequence
- ledger/onroad/taskprocessor.go#ContractTaskProcessor.work 
  - Take a contract address from the sorted queue and process it
- ledger/onroad/taskprocessor.go#ContractTaskProcessor.processOneAddress 
  - is processed for this contract
- ledger/generator/generator.go#GenerateWithOnRoad 
  - generates a receive block based on a send block
- vm/vm.go#VM.RunV2 
  - run vm logic
- ledger/onroad/access.go#insertBlockToPool 
  - insert the generated block into the pool
- ledger/pool/pool.go#AddDirectAccountBlock 
  - Insert the block directly into the ledger through the pool

The following process refers to the user sending a transfer transaction

## sbp generates a snapshot block

- ledger/consensus/trigger.go#update 
  - generates a snapshot block start event and passes it to the downstream
- producer/worker.go#produceSnapshot 
  - receives consensus layer messages and creates a separate coroutine for snapshot block generation
- producer/worker.go#randomSeed 
  - Calculate random number seed
- producer/tools.go#generateSnapshot
- ledger/chain/unconfirmed.go#GetContentNeedSnapshot 
  - Calculate which account blocks need snapshots
- producer/tools.go#insertSnapshot
- ledger/pool/pool.go#AddDirectSnapshotBlock 
  - insert snapshot block into ledger
- ledger/verifier/snapshot_verifier.go#VerifyReferred 
  - to verify the validity of the snapshot block
- ledger/chain/insert.go#InsertSnapshotBlock 
  - insert snapshot block into chain
- ledger/chain/insert.go#insertSnapshotBlock 
  - update indexDB, stateDB

# Architecture overview

The vite protocol is a DAG data structure protocol. The block is divided into two parts, the account block and the snapshot block.
If you make an analogy with ETH:
- send account block is similar to ETH transaction
- receive account block similar to ETH's receipt
- snapshot block is similar to ETH's block

The user signs the account block to change the state in the account, and the account block height of a single account is connected with this to form an account chain.
The height on the account block is equivalent to the nonce of each account in ETH, which represents the height of the account chain.
In terms of data structure, a big difference from ETH is that both the send block and the receive block occupy the height of the account chain, and the account block of vite exists in the form of a linked list, and each block refers to the hash of the previous block as its own prevHash. 
