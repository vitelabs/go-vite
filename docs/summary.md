# go-vite summary

<div align="center">
    <img src="https://github.com/vitelabs/doc.vite.org/blob/master/docs/.vuepress/public/logo_black.svg" alt="Logo" width='300px' height='auto'/>
</div>

<br />

[Vite](https://vite.org) is a next-generation Reactive Blockchain that adopts a _message-driven, asynchronous architecture and a DAG-based ledger_.
The goal for Vite's design is to _provide a reliable public platform for industrial dApps_, with features of ultra-high throughput and scalability.

As a result of the underlying data structure of the protocol being a directed acyclic graph (DAG), the block is divided into two parts, the account block and the snapshot block. If you make an analogy with ETH:

* send account block is similar to ETH's transaction
* receive account block similar to ETH's receipt
* snapshot block is similar to ETH's block

The user signs the account block to change the state in the account, and the account block height of a single account is connected with this to form an account chain. The height on the account block is equivalent to the nonce of each account in ETH, which represents the height of the account chain. In terms of data structure, a big difference from ETH is that both the send block and the receive block change the height of the account chain, and the account block of Vite exists in the form of a linked list, and each block refers to the hash of the previous block.

Table of Content:
* [Directories & packages](#directories)
  * [client](#directories_client)
  * [cmd](#directories_cmd)
  * [common](#directories_common)
  * [crypto](#directories_crypto)
  * [interfaces](#directories_interfaces)
  * [ledger](#directories_ledger)
  * [log15](#directories_log15)
  * [monitor](#directories_monitor)
  * [net](#directories_net)
  * [node](#directories_node)
  * [pow](#directories_pow)
  * [producer](#directories_producer)
  * [rpc](#directories_rpc)
  * [rpcapi](#directories_rpcapi)
  * [tools](#directories_tools)
  * [vm_db](#directories_vm_db)
  * [vm](#directories_vm)
  * [wallet](#directories_wallet)
* [Data flow examples](#examples)
  * [User sends a transfer transaction](#examples_1)
  * [Generate a receive block for a contract](#examples_2)
  * [SBP generates a snapshot block](#examples_3)
* [Appendix](#appendix)
  * [Setup goplantuml](#goplantuml)
  * [Other visualization tools](#visualization)

## Directories & packages <a name="directories"></a>

### client <a name="directories_client"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/client.png" alt="client">
</p>

[[Full client diagram]](/docs/images/summary_diagrams/puml/client_full.png?raw=true)

implements the go version client for remotely calling gvite nodes

### cmd <a name="directories_cmd"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/cmd.png" alt="cmd">
</p>

[[Full cmd diagram]](/docs/images/summary_diagrams/puml/cmd_full.png?raw=true)

command line script, the entry of gvite is here cmd/gvite/main.go

### common <a name="directories_common"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/common.png" alt="common">
</p>

[[Full common diagram]](/docs/images/summary_diagrams/puml/common_full.png?raw=true)

some common method components and constant definitions

### crypto <a name="directories_crypto"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/crypto.png" alt="crypto">
</p>

[[Full crypto diagram]](/docs/images/summary_diagrams/puml/crypto_full.png?raw=true)

implementation of encryption algorithm and hash algorithm ed25519 and blake2b-512 are the main algorithms on the vite chain

### interfaces <a name="directories_interfaces"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/interfaces.png" alt="interfaces">
</p>

[[Full interfaces diagram]](/docs/images/summary_diagrams/puml/interfaces_full.png?raw=true)

the interface definitions of some modules are placed here to solve the golang circular dependency problem

### ledger <a name="directories_ledger"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/ledger.png" alt="ledger">
</p>

[[Full ledger diagram]](/docs/images/summary_diagrams/puml/ledger_full.png?raw=true)

contains ledger related implementations

### log15 <a name="directories_log15"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/log15.png" alt="log15">
</p>

[[Full log15 diagram]](/docs/images/summary_diagrams/puml/log15_full.png?raw=true)

log framework used in log15 go-vite was copied since many things were changed at the beginning. The original project address: https://github.com/inconshreveable/log15

### monitor <a name="directories_monitor"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/monitor.png" alt="monitor">
</p>

[[Full monitor diagram]](/docs/images/summary_diagrams/puml/monitor_full.png?raw=true)

toolkit written for the convenience of data statistics during stress testing

### net <a name="directories_net"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/net.png" alt="net">
</p>

[[Full net diagram]](/docs/images/summary_diagrams/puml/net_full.png?raw=true)

network layer implementation, including node discovery, node interoperability, ledger broadcast and pull

### node <a name="directories_node"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/node.png" alt="node">
</p>

[[Full node diagram]](/docs/images/summary_diagrams/puml/node_full.png?raw=true)

an upper package of ledger, mainly doing some combined functions

### pow <a name="directories_pow"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/pow.png" alt="pow">
</p>

[[Full pow diagram]](/docs/images/summary_diagrams/puml/pow_full.png?raw=true)

locally implemented pow calculation and remote call pow-client to calculate pow

### producer <a name="directories_producer"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/producer.png" alt="producer">
</p>

[[Full producer diagram]](/docs/images/summary_diagrams/puml/producer_full.png?raw=true)

triggers the entry of snapshot block and contract block production

### rpc <a name="directories_rpc"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/rpc.png" alt="rpc">
</p>

[[Full rpc diagram]](/docs/images/summary_diagrams/puml/rpc_full.png?raw=true)

underlying implementation of the three remote calling methods of websocket/http/ipc

### rpcapi <a name="directories_rpcapi"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/rpcapi.png" alt="rpcapi">
</p>

[[Full rpcapi diagram]](/docs/images/summary_diagrams/puml/rpcapi_full.png?raw=true)

interface exposed by each module

### tools <a name="directories_tools"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/tools.png" alt="tools">
</p>

[[Full tools diagram]](/docs/images/summary_diagrams/puml/tools_full.png?raw=true)

similar to collection, some collections are implemented

### vm_db <a name="directories_vm_db"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/vm_db.png" alt="vm_db">
</p>

[[Full vm_db diagram]](/docs/images/summary_diagrams/puml/vm_db_full.png?raw=true)

the storage interface required by the virtual machine during execution is the encapsulation of the virtual machine's access to the chain

### vm <a name="directories_vm"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/vm.png" alt="vm">
</p>

[[Full vm diagram]](/docs/images/summary_diagrams/puml/vm_full.png?raw=true)

the realization of virtual machine and the realization of built-in contract, there are a lot of virtual machine execution logic in it

### wallet <a name="directories_wallet"></a>

<p align="center">
  <img src="/docs/images/summary_diagrams/puml/wallet.png" alt="wallet">
</p>

[[Full wallet diagram]](/docs/images/summary_diagrams/puml/wallet_full.png?raw=true)

implementation of wallet, private key mnemonic management, and signature and verification signature

## Data flow examples <a name="examples"></a>

### User sends a transfer transaction <a name="examples_1"></a>

TODO: create diagram

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

### Generate a receive block for a contract <a name="examples_2"></a>

TODO: create diagram

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

### SBP generates a snapshot block <a name="examples_2"></a>

TODO: create diagram

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

## Appendix <a name="appendix"></a>

### Setup goplantuml <a name="goplantuml"></a>

[GoPlantUML](https://github.com/jfeliu007/goplantuml) is a PlantUML Class Diagram Generator for golang projects. It can generate class diagram text compatible with plantuml with the information of all structures and interfaces as well as the relationship among them.

1. Install goplantuml
     - go install github.com/jfeliu007/goplantuml/cmd/goplantuml@v1.5.2 
2. Install Java Runtime Environment
     - sudo apt update
     - sudo apt install default-jre
3. Execute the script to generate all puml diagrams
     - ./docs/scripts/gen_diagrams.sh

### Other visualization tools <a name="visualization"></a>

#### GoCity

GoCity is an implementation of the Code City metaphor for visualizing source code. GoCity represents a Go program as a city, as follows:

* Folders are districts
* Files are buildings
* Structs are represented as buildings on the top of their files.

Structures Characteristics

* The Number of Lines of Source Code (LOC) represents the build color (high values makes the building dark)
* The Number of Variables (NOV) correlates to the building's base size.
* The Number of methods (NOM) correlates to the building height.

https://go-city.github.io/#/github.com/vitelabs/go-vite

<p align="center">
  <img src="/docs/images/summary_appendix/gocity.gif" alt="gocity">
</p>