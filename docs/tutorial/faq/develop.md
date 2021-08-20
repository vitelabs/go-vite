# Troubleshooting & FAQ - Development

## Dev Node
See [Run a Local Dev Node](../dappguide/testnode.html)

You can also set up a dev node by installing the VSCode Soliditypp extension. The extension will install a local dev node automatically.

## RPC Error Code
See [Common RPC Errors](../../api/rpc/#common-rpc-errors)

## Tips of Sending Transaction

1. Check the mandatory block fields such as `previousHash`, `blockType`, `accountAddress`, `toAddress`, `sendBlockHash`, `amount`, `tokenId` and `data`;
2. Estimate the quota consumption. If the account has sufficient quota, create the account block and send;
3. Otherwise, provide a correct PoW nonce (obtained by calling getPoWDifficulty API) in the account block and send the block. Remember, you should not send the transaction by PoW if network congestion occurs. In this case, wait for a while and check the network status again. 

Related APIs：
- [Quota Consumption / Available Quota / Network Status / PoW Difficulty](../../api/rpc/ledger_v2.html#ledger_getPoWDifficulty)
- [PoW Nonce](../../api/rpc/util.html#util_getPoWNonce)
- [Sending Transaction](../../api/rpc/ledger_v2.html#ledger_sendRawTransaction)

:::tip Best Practice
Multiple transactions of the same account must be strictly ordered. An account chain is formed according to the dependency of `hash` and `previousHash`. If there are two different blocks with the same `previousHash`, only one block of them will be snapshotted, and the other block will be discarded. 

Therefore, when sending and receiving transactions in batches, in order to have the maximum efficiency and the minimum error rate, it is best to use one thread for the same account to send and receive transactions, and use the `previousHash` field to specify the sending order.
:::

Common errors when sending transaction through `ledger_sendRawTransaction`:

* lack of difficulty field

Empty `difficulty` field

* check pow nonce failed

Fields of `difficulty` and `nonce` don't match

* tokenId doesn’t exist

Wrong `tokenId`

* verify hash failed

Wrong block hash

* verify signature failed

Wrong block signature

* receive's AccountAddress doesn't match the sendToAddress

The `sendBlockHash` in response block doesn't link to a correct unreceived transaction.

* Inconsistent execution results in vm, err:xxx

Field xxx does not match the result in the VM. Usually this is caused by incorrect `toAddress` when deploying a contract or wrong `data` field when calling contracts.

* abi: method not found

The called contract method does not exist. Check the method name and signature in the `data` field. 

* invalid method param

Wrong parameter used when calling the contract. Usually this is caused by incorrect `amount`, `tokenId` or parameters encoded in `data` field. 

* contract not exists

The contract specified in `toAddress` field of the request transaction does not exist.

* The node time is inaccurate, quite different from the time of latest snapshot block. 

The last snapshot block on the node is out of date. Check the sync status of the node. 

* calc PoW twice referring to one snapshot block

PoW used twice in the same snapshot block. Try to send the transaction again in the next snapshot block. 

* generator_vm panic error

Panic incurred in VM, usually caused by a recent rollback on the node. Try to send the transaction again. 

* block is already received successfully

Cannot receive a request transaction (specified by `sendBlockHash`) that has been received.

* fail to find receive's send in verifyProducerLegality；failed to find the recvBlock's fromBlock

The request block corresponding to `sendBlockHash` in the response transaction is not found. This is usually caused by a rollback. Make sure the request transaction is not reverted and retry. 

* verify prevBlock failed, incorrect use of prevHash or fork happened

The transaction's `previousHash` or `height` cannot match the previous block on the account chain. Use the `hash` and `height` of the latest block to rebuild the transaction. 

* insufficient balance for transfer

Insufficient balance to finish the transaction.

* out of quota

Insufficient quota to finish the transaction. You should follow the below steps.
1. If the Vite network is in congestion, try to send the transaction later;
2. If you have stakes, and the quota received from the stake can cover the consumption, wait for a while and send the transaction again;
3. If you don't have stake or the stake is insufficient to send the transaction, stake first, or send the transaction through PoW.

## Stake for Quota

The available quota of the account relies on how much VITE has been staked and the actual use of quota in the past 74 snapshot blocks. Refer to [Quota Consumption Rules](../rule/quota.html#quota-consumption-rules) to know how quota works on Vite. 

There are two ways to know the staking amount:
* Refer to [Quota Calculation](../rule/quota.html#quota-calculation) and make the estimation
* See [Get Minimum Required Stake Amount](../../api/rpc/contract_v2.html#contract-getrequiredstakeamount)

For example, account A needs to send two transfers (having no comment) in 75 seconds, consuming 21000*2 quota, according to the above table, the required staking amount is 267 VITE.

## Auto-Receive Transactions

Some Vite SDKs, like vite.js provides the API to receive transactions for an account automatically. This is recommended. 

For advanced usage, poll [Unreceived Transaction List](../../api/rpc/ledger_v2.html#ledger_getUnreceivedBlocksByAddress) or subscribe to [Unreceived Transaction Event](../../api/rpc/subscribe_v2.html#subscribe_createUnreceivedBlockSubscriptionByAddress) of an account to get unreceived transactions, then create corresponding response blocks and [Send Response Transaction](../../api/rpc/ledger_v2.html#ledger_sendRawTransaction).

## Event Subscription Not Working

Follow the following steps to investigate:
* Check if the connecting node is fully synced; 
* Check if `subscribe` is configured in `PublicModules` of node_config.json and `"SubscribeEnabled":true` is set;
* Check if the subscription is correct and the event does incur. For example, to check new transaction events on given account, you can call [GetAccountBlocks API](../../api/rpc/ledger_v2.html#ledger_getAccountBlocksByAddress). Assuming account A sends a transaction to account B, you should subscribe to new transaction events on account A, or new unreceived transaction events on account B. 

## How to Know an Address is Contract

This is defined in [VEP 16: Specification of Address Generation](../../vep/vep-16.html). Check it out.

## Random Numbers in Solidity++

Two methods are provided in Solidity++ to get randoms:
```
// return a random. the returning value keeps the same during one call
uint64 random = random64();
// return a random. this methods returns different random numbers each time
uint64 random = nextrandom();
```

Refer to [VEP 12: The Implementation of Random Numbers in Vite](../../vep/vep-12.html) for the details.

:::tip Note
The `randomDegree` parameter when deploying the contract specifies the degree of confidence. The bigger `randomDegree`, the more secure the random number generated, and the slower the contract producing response. Choose your `randomDegree` according to your business need in the contract. 
:::

## Decode Calling Parameter and Event

Use `--abi` option to get the ABI definition when compiling the contract. An ABI consists of constructor, methods, events (aka vmlog), etc. Given the ABI definition and the data field of the transaction, you can decode the calling parameters; given the ABI definition and the vmlog field of the transaction, you can decode the event.

The ABI definition of built-in contracts are specified in [Built-in Contracts](../../api/rpc/contract_v2.html#call-built-in-contract)

Vite SDKs such as vite.js and vitej provide convenient APIs for encode and decode. 

## Get Contract State

When a contract is running, there are two methods to check the states:
* Decode event to get the state value in the VS-Code soliditypp extension or through vite.js SDK;
* Set `"VMDebug":true` in node_config.json to print running log in {DataDir}/maindata/vmlog/interpreter.log, including each instruction, and `stack`, `memory`, `storage` states.

We also provide two methods to check the contract states offline:
* Call the Solidity++ getter method through [Call Offline Method API](../../api/rpc/contract_v2.html#contract-calloffchainmethod);
* Call [Get Contract Storage API](../../api/rpc/contract_v2.html#contract_getContractStorage) and parse the result.

## Estimate Quota Consumption of Contract

The quota spent when receiving a calling request is defined in [Quota Consumption Table of Transactions](../../vep/vep-17.html#quota-consumption-table-of-transactions) and [Quota Consumption Table of VM Instructions](../../vep/vep-17.html#quota-consumption-table-of-vm-instructions).

Below is a simple example to explain how quota is spent for a contract response transaction:

When a contract method is called, if it is non-payable, the contract will first check whether the transfer value of the request transaction is 0. `341561001057600080FD` is the binary code for checking the transfer amount.

Let's translate the code into the OPCODEs that are understandable:
```
34 CALLVALUE - Get the transfer value from the request transaction and put on the top of the stack, consuming 1 unit of quota
15 ISZERO -  Check if the transfer value on the top of the stack is 0. If yes, push 1 on the stack, otherwise push 0. This operation costs 1 unit of quota
610010 PUSH2 0x0010 - Push 0x0010 onto the stack, consuming 1 unit of quota
57 JUMPI - If the second element on the stack is not 0, jump to 0x0010 to execute, consuming 4 units of quota
6000 PUSH1 0x00 - Push 0x00 onto the stack, consuming 1 unit of quota
80 DUP1 - Copy the top element and push it into the stack, consuming 1 unit of quota
FD REVERT - Pop the top two elements from the stack and return, consuming zero quota
```

The total quota consumption is 21000+1+1+1+4+1+1+0 = 21009

## How to Get vmlog

Event (aka vmlog) is not saved on the node by default. You must explicitly turn it on in node_config.json on the node that provides RPC API service, as following
* `"VmLogAll":true` -  Save all vmlog
* `"VmLogWhiteList":["vite_d789431f1d820506c83fd539a0ae9863d6961382f67341a8b5","vite_000000000000000000000000000000000000000595292d996d"]` - Save the vmlog of the specified contracts
