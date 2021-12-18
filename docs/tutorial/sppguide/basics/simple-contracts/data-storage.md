---
order: 3
---
# Data Storage Contracts
  
Let’s look at some simple examples on how to store and access data through smart contracts.

## Simple Storage Example

Here's an example that sets the value of a variable and defines a **getter** function for accessing that variable.

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/datastorage1.solidity

The line `uint storedData;` declares a state variable (an unsigned 256 bit integer) which is stored on the block chain and avaliable to any functions within the contract.

`6-8` Defines a message listener allowing anyone to modify our state variable by calling the set function.

`11-13` Defines a **getter** function that can be used to access the value of `storedData`.

So far this seems relatively straightforward, but there is an important caveat: getter functions are off-chain, and can *not* be called by other contracts. They are conveniences for interfaces or front ends to collect needed information.

So how do contracts interact?
One of the challenges of Solidity++ is understanding the asynchronous design and how values can not be returned by functions. Rather than return values, a message can be passed to request a value, and upon receiving the message, a response message can be generated and sent back. 

## Contract Accessible Storage

Here we look at an example of basic communication between contracts. We set up two contracts which are intended to work with eachother.


<<< @/tutorial/sppguide/basics/simple-contracts/snippets/datastorage2.solidity

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/datastorage3.solidity

#### Message and Send

Here we are introduced to `message` and `send`, new features of Solidity++.

A `message` is a data structure containing the call signature that cooresponds to a message listener. A message can then be populated with data and sent to an address using `send`. If there is a cooresponding message listener at the target address, that function will be called.

However, `send` is *non-blocking*, and the contract receiving a message may execute the function call at a later time. This does require some consideration.

#### How it works

In this example, the contract `ContractStorage` has the same `storedData` and `set` functions as our `SimpleStorage` contract did, but there is no `getter` function. Instead, there is a `requestData` function which will reply to the sender with a `replyData` message using `send`.

Meanwhile, the `Accesser` contract is designed to store a copy of the data contained in `ContractStorage`. A user calls the `updateCopyData` function with the address of the `ContractStorage` contract, to initiate the update process. `Accesser` will send a `requestData` message to `ContractStorage`, which will react to the request and respond with a `replyData` message, allowing `Accesser` to update it's local copy `copyData`.

