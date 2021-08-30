---
order: 2
---

# Migrating from Solidity++ 0.4

Solidity++ 0.8 is syntactically closer to Solidity than Solidity++ 0.4, which is more friendly to developers from Solidity. 

Some counter-intuitive syntex and keywords are removed in this version.


## Solidity++ 0.8.0 Breaking Changes

* All Solidity breaking changes since v0.5.0 to v0.8.0 are applied to Solidity++ 0.8.

* `onMessage` keyword is deprecated since 0.8.0. Use function declarations instead (`function external async`).

* `message` keyword is deprecated since 0.8.0. Use function types instead.

* `send` keyword is deprecated since 0.8.0. Use function calls instead.

* `getter` keyword is deprecated since 0.8.0. Use offchain function instead (`function offchain`).

* Inline assembly and Yul are available in this version.

* `keccak256` and `sha256` are available in this version.

## How to Update Your Code

Let's start with an example:

```javascript
pragma soliditypp ^0.4.3;

contract A {
    message sum(uint result);
    
    onMessage add(uint a, uint b) {
        uint result = a + b;
        address sender = msg.sender;
        send(sender, sum(result));
   }
}

contract B {
    address addrA;
    uint total;
    message add(uint a, uint b);

    constructor (address addr) {
        addrA = addr;
    }

    onMessage invoke(uint a, uint b) {
        send(addrA, add(a, b));
    }

    onMessage sum(uint result) {
        total += result;
    }

    getter total() returns(uint) {
        return total;
    }
}
```

In the above code, contract A declares a message listener `add(uint a, uint b)`.

contract B declares `add` message which has the same signature to `add` message listener in contract A.

Contract B declares a message listener `invoke` as the entry to the contract. When `B.invoke()` is called, contract B sends a `add` message to contract A to initiate an asynchronous message call.

When contract A responds to the message call, it sends a `sum` message back to contract B to return data asynchronously.

Contract B also declares a message listener `sum(uint result)` as a *'callback function'* to handle the returned message from contract A.

Since Solidity++ 0.8.0, no explicit `onMessage` declaration as a callback is required. The compiler is smart enough to generate callbacks automatically. The contract code is simplified and optimized significantly by async/await syntactic sugar.

The migrated code is as follows:

```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract A {
    // the onMessage is replaced with an async function
    function add(uint a, uint b) external async returns(uint) {
        return a + b;
    }
}

contract B {
    A contractA;
    uint public total;  // an offchain getter function will be auto-generated

    constructor (address addr) {
        contractA = A(addr);
    }

    // the onMessage is replaced with an async function
    function invoke(uint a, uint b) external async {
        // use await operator to get return data
        total += await contractA.add(a, b);
    }
}
```



