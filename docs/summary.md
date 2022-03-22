# go-vite summary

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

## Appendix <a name="appendix"></a>

### Setup goplantuml <a name="goplantuml"></a>

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