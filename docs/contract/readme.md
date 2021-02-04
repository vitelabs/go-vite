---
order: 1
parent:
    title: Smart Contract
---

# Introduction

Vite is a high-performance decentralized application platform built on asynchronous message-driven architecture. 
The smart contracts in Vite are written in Solidity++, a programming language that extends Ethereum Solidity by adding asynchronous semantics while maintains major compatibility.
Smart contracts in Vite won't share states but communicate with each other via messaging.
User is able to write, compile smart contracts and deploy in Vite network now.

## What is Asynchronous Smart Contract

Cross-contract calls in Ethereum are represented as function calls, or internal transactions. This set of calls are either completed at the same time or all fail. Obviously, this kind of atomic ACID semantic could become a performance bottleneck in system. 
To tackle the issue, Vite adopts an asynchronous, message-driven architecture on the basis of known solution of centralized Internet technology.  Smart contracts in Vite communicate with each other via sending messages instead of sharing states.

Similar to common transfer, a contract call is separated into a contract request transaction and a contract response, representing as transaction blocks appended into the account chains of requester and responder of a contract call respectively.

The manner how these transactions are written to the ledger and how they are confirmed are also asynchronous. A "snapshotted" contract request transaction means the contract call is successfully initiated. A "snapshotted" contract response transaction indicates the contract call is complete.

There is no return value in asynchronous contract. Execution result should be returned to the caller by callback, which actually yields a new transaction after current execution process has completed.

## Who is Responsible for Executing Smart Contract

When smart contract is created, the owner should designate a delegated consensus group. The delegated nodes in the group will execute the smart contract and generate transaction block using DPoS algorithm.

A smart contract can only designate one delegated consensus group, which can't be changed afterwards. A delegated consensus group can serve multiple smart contracts.

A built-in delegated consensus group, aka Global Consensus Group, is provided to serve smart contracts that haven't designated delegated consensus group. The block producers of Global Consensus Group are same with those of Snapshot Consensus Group but having different block producing order.

## Priority of Contract Response Transaction

If multiple contracts have designated the same delegated consensus group, they could be prioritized according to how much quota they have. The higher quota the contract possesses, with the higher priority the transaction of the contract could be handled by consensus group. 
Since prioritization of contract response transactions is not specified in Vite's protocol, each delegated consensus group can define prioritization rules on its own.

FIFO(First In First Out) rule must be guaranteed when multiple messages are sent to a contract from the same account. In other words, the request transaction having lower block height is always accepted by the contract in ahead of that having higher height. FIFO does not apply to scenario where messages are sent out from different accounts.

If multiple accounts happen to send messages to a contract simultaneously, the delegated consensus group will choose a random processing order. However, the consensus group can define an algorithm for prioritization purpose on its own. 

## Cost of Contract

### Fees for Creating Contract

Creating new contract consumes VITE. In Pre-Mainnet, a destruction of 10 VITE is required for creating a contract.

### Quota in Contract

The quota for contract creation request transaction is supplied by contract creator, while the quota consumed in contract creation response transaction comes from the destruction of VITE. In Pre-Mainnet, destroying 10 VITE during smart contract creation will receive quota up to 47.62 `UTPS`.

Similar to contract creation, contract execution consumes quota as well. The contract request transaction and the contract response transaction consume the quota of transaction initiator and contract account respectively.

In Pre-Mainnet, contract account can only obtain quota by staking. If a contract does not have sufficient quota, no transaction of this contract will be performed. Therefore, **contract provider should always ensure 'enough' VITE tokens have been staked for the contract**.

For complex contract, the quota of contract account may be insufficient for generating response transaction. In this case, the response transaction will consume up the quota and fail in the end by generating a "failed" response block on the contract chain. And, if the request has carried transfer amount, the tokens will be returned to original account after transaction fails. In Pre-Mainnet, a "failed" contract response due to insufficient quota will cause the contract locked for 75 snapshot blocks. In other words, no transaction for the contract will be processed during the time until it is unlocked.

## Smart Contract Language

Ethereum provides Solidity, a Turing-complete programming language for developing smart contracts. To support asynchronous semantics, Vite extends Solidity and defines a set of syntax for message communication. The extended Solidity in Vite is called Solidity++.

Solidity++ supports most of Solidity's syntax, but will no longer support synchronous function calls between contracts. Developers can define messages through `message` keyword and define message handlers via `onMessage` keyword to enable cross-contract communication. Messages in Solidity++ are compiled into `CALL` instructions. As a result, a request transaction is generated and appended to Vite's ledger, which plays a key role as message middleware for asynchronous communication between contracts, ensuring reliable storage of messages and preventing duplication.

## Virtual Machine

At present, Ethereum has a large community of developers, who have contributed abundant smart contracts based on Solidity and EVM environment. 
Since Vite's VM is partially compatible with EVM, most EVM instructions can maintain the original semantics in Vite. However, due to different ledger structure and transaction definition, the semantics of some EVM instructions need to be redefined.

### Instruction Set

See [Vite Instruction Set](./instructions.html)

## Contract Debugging

See [Debugging Smart Contracts](./debug.html)
