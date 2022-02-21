---
order: 1
parent:
    title: Smart Contract
---

# Introduction

Vite is an asynchronous blockchain built on a DAG ledger. 

Vite smart contracts can be written in Solidity++, a programming language that extends Solidity with asynchronous semantics while maintains major compatibility.

## Asynchronous Message Calls

Message calls in Ethereum are fully synchronous. All message calls in a call stack are executed sequentially in one transaction, they either completed at the same time or all fail.

Vite adopts an asynchronous architecture. A message call on Vite is separated into a *request* and a *response*. Each message call explicitly yields an on-chain *request transaction* (aka *send transaction*) when initiated and an on-chain *response transaction* (aka *receive transaction*) when executed.

A request transaction indicates the call is successfully initiated and wait for the called contract to accept. A response transaction indicates the call is accepted and executed by the called contract.

There is no data to return immediately when the call is sent. When the request is accepted by the called contract, it will trigger a new request transaction as a *callback* to pass the execution result back to the caller.

## Costs

Creating new contract consumes VITE tokens. In Pre-Mainnet, a base fee of 10 VITE is required for creating a contract.

A more accurate fee model will be applied on mainnet in the future.

Contract execution consumes quota instead of VITE tokens. The request transaction and the response transaction consume the quota of caller and the called contract respectively.

A contract can obtain quota by staking VITE tokens. If the contract runs out of quota, it will stop accepting requests until the quota is restored. Therefore, **contract deployer should ensure 'enough' VITE tokens have been staked for the contract**.

If the quota of a contract is insufficient for generating a response transaction, the execution will consume up the remaining quota and generate a Panic exception. If the requested transaction includes a token transfer, the token will be returned to the original account. A panic due to insufficient quota will cause the contract suspended for 75 snapshots (about 75 seconds).

## Smart Contract Language

The asynchonous extension to Solidity is called Solidity++.

### Messages and Callbacks
In the legacy versions of Solidity++, such as Solidity++ 0.4.3, developers can declare *messages* through `message` keyword, declare *callbacks* via `onMessage` keyword and send messages to another contract by using `send` statement. 

A *Request Transaction* is generated and appended to Vite's ledger when Vite VM executes a `send` statement. 

Vite VM will execute the code of the `onMessage` declared in the called contract ***asynchronously***, then generate a *Response Transaction* linked to the request transaction, and append it to Vite's ledger.

### Function Calls
Since Solidity++ 0.8.0, a fully Solidity compatible syntax is introduced and the low-level asynchronous syntax from 0.4.3 is deprecated.

Developers can declare a `function` instead of `onMessage` as a callback to receive requests to the contract.

And use an external function call statement rather than a `send` statement to call another contract asynchronously.

### Promise and Async/Await
In Solidity++ 0.8.1, we will introduce a modern approach to asynchronous functions using `promise` and `async`/`await`.

To those familiar with mordern programming languages such as javascript, the `async`/`await` provides a more elegant and readable way to organise asynchronous function calls, and implement more reliable and concise control flows.

## Vite Virtual Machine

Vite VM is an asynchonous EVM which is partially compatible with EVM. Most EVM instructions maintain the original semantics in Vite.

However, due to the asynchonous nature of Vite and the different ledger and consensus, some instructions need to be modified or extended.

See [Vite Instruction Set](./instructions.html) for details.

## Develop and Debug

See [Debugging Smart Contracts](./debug.html)
