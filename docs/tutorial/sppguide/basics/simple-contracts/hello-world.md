---
order: 2
---
# Hello World

<!-- links I removed that I want to put back
 (for more details see [contract deployment and consensus groups](link))
 (listened for)
 (filter blockchain events)
-->

Let’s begin with a basic example contract that redirects payments to a chosen address and logs a friendly message.

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/helloworld.solidity

Let's go through each line in detail.

**`1`** The first line indicates to the compiler that the source code is written for Solidity++ version 0.4.3 or newer, up to but not including version 0.5.0. Note that Solidity++ is still in the early stages of development, and minor version changes (e.g. from 0.4.x to 0.5.x) are not guaranteed to be compatible.

**`3`** This line begins the declaration of a *contract* with the name `HelloWorld`. Contracts in Solidity++ are similar to classes from other languages. They contain a collection of stored data (its *state*), and executable code for manipulating that data (its *functions* or *methods*). Contracts can be *deployed* to the network, after which both compiled code and data are stored on the blockchain. Each deployed contract is given a unique address on the blockchain, such that even identical code can be deployed multiple times while still creating distinct contracts, each with their own data and address.

**`4`** The next line `event MyLog(address indexed addr, string log);` declares an *event* called `MyLog`. Events are used for logging data directly to the blockchain and are used in combination with the `emit` keyword. The event *declaration* specifies the data field names and types for an event, while *emitting* the event allow these fields to be populated with data and stored on the blockchain. While data logged in the form of emitted events **cannot** be accessed by a contract, events can be listened for by Vite clients such as web applications, allowing for responsive blockchain applications.

Events can contain a variety of data types. In this case we chose for our event structure to contain an `address` and a `string`. Addresses store a Vite address which uniquely identify either an account in a wallet, or a deployed contract. Note that the `indexed` modifier allows for Vite clients to efficienty filter blockchain events, which is important for developing dApps. Logging strings in events can be convenient for testing and debugging as shown here, though they generally are not as practical for deployed contracts.

**`6`** This line `onMessage sayHello(address dest) payable {` begins the declaration of a *message listener*, one of the most important function types in Solidity++. Message listeners will execute when the contract receives a corresponding *message*, which can be sent from any Vite address, even from another contract.

For a message listener to execute, the message must indicate the message listener name (here it is `sayHello`) as well as contain the appropriate arguments (`sayHello` requires a single address `addr` as an argument).

Within a message listener, contracts have access to the special `msg` global variable. This contains information about the message that triggered the contract, such as the address that sent the message (`msg.sender`), the type of token sent with the message (`msg.tokenid`), the amount of tokens (`msg.amount`) as well as a complete copy of the data package (`msg.data`).

**`7`** The code `dest.transfer(msg.tokenid, msg.amount);` initiates a transfer from the contract to the address stored in `dest` of all the tokens that were initially sent in the message that triggered `sayHello`.

**`8`** Finally the code `emit MyLog(msg.dest, "Hello! Have some Vite!");` stores both the address of the recipient of our funds and a message in the blockchain.

