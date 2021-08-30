---
order: 1
---

# Introduction to Solidity++ 0.8

The design goal of Solidity++ 0.8 is to be compatible with the syntax of Solidity as much as possible, so that developers don’t have to learn different languages to implement the same thing.

We assume that you are already familiar with Solidity, if not, please read [Solidity documentation](https://docs.soliditylang.org/) before starting.

::: tip Notice
The Solidity++ 0.8 is still under development, you can try a nightly build version for development or test, but do **NOT** deploy contracts to the mainnet until a stable version is released. Please report an issue [here](https://github.com/vitelabs/soliditypp/issues) if you find a bug.
:::

## Get Started
Let us begin with a basic example that sets the value of a variable and exposes it for frontend to access. 

### Hello World Example
```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract SimpleStorage {
    uint storedData;

    function set(uint x) external async {
        storedData = x;
    }

    function get() offchain view returns (uint) {
        return storedData;
    }
}
```

As above, the way of declaring a contract is syntactically similar to Solidity.

The first line tells you that the source code is licensed under the GPL version 3.0.

The next line specifies that the source code is written for Solidity++ version 0.8.0, or a newer version of the language up to, but not including version 0.9.0. This is to ensure that the contract is not compilable with a new (breaking) compiler version, where it could behave differently.
The only difference with Solidity is to replace `solidity` with `soliditypp`.

There are two interface functions in the **SimpleStorage** contract. The difference with Solidity is that the function `set` is declared as `async` and the function `get` is declared as `offchain`.

An *async function* does not execute or return result in the request transaction but in a subsequent response transaction.

An *offchain function* is used to retrieve data of a contract from a frontend. The codes of offchain functions are not stored in the blockchain ledger but in an off-chain frontend. The frontend must pass the compiled binary code along with the parameters each time it is called.

::: tip Notice
An offchain function can **NOT** be called in a contract.
:::

## Solidity Compatibility

Solidity++ 0.8 is compatible with almost all syntax of Solidity 0.8.0 except for the differences mentioned in this article.

All features related to Ethereum are replaced with that related to Vite.

The `new` keyword is disabled in this version. In other words, it is impossible to deploy a new contract from a contract at runtime.

In addition, `delegatecall`, `selfdestruct` and `ecrecover` are disabled in this version.

## Modifiers for Functions and State Variables

Modifiers can be used to declare the behaviour of functions and state variables. Both Solidity and Solidity++ have a set of built-in language-level modifiers, and also support user-defined modifiers.

### Modifiers in Solidity

The built-in modifiers in Solidity can be divided into two categaries: **Visibility** and **State Mutability**.

- **Visibility** defines the rules of how we access functions and state variables from different scopes.
- **State Mutability** defines the rules of how a function can mutate the EVM state.

Valid values of **Visibility** and **State Mutability** in Solidity are shown in the following table:

| Visibility 	| State Mutability 	|
|------------	|------------------	|
| public     	| pure             	|
| external   	| view             	|
| internal   	| payable          	|
| private    	| nopayable        	|

### Modifiers in Solidity++

In Solidity++, an additional value `offchain` is added to **Visibility** category.

In addition, a new category **Execution Behavior** is added.

- **Execution Behavior** defines how a function will be executed in Vite VM, in a synchronous or asynchronous manner.

Valid values of **Visibility**, **State Mutability** and **Execution Behavior** in Solidity++ are shown in the following table:

| Visibility 	| State Mutability 	| Execution Behavior 	|
|------------	|------------------	|--------------------	|
| public     	| pure             	| sync               	|
| external   	| view             	| async              	|
| internal   	| payable          	|                    	|
| private    	| nopayable        	|                    	|
| offchain   	|                  	|                    	|

## Visibility

There are four types of visibility for functions and state variables in Solidity: `external`, `public`, `internal` or `private`. 

An additional function visibility `offchain` is added in Solidity++. It means that those functions can only be accessed offchain. That is to say, they can only be called through a frontend, but not in another contract.

In Solidity, functions have to be specified as being `public`, `external`, `internal` or `private`. For state variables, `external` is not possible.

In Solidity++, functions have to be specified as being `external`, `internal`, `private` or `offchain`, `public` is not possible. For state variables, `external` or `offchain` is not possible.

The valid types of visibility for functions and state variables in Solidity and Solidity++ are shown in the following table:


|          | function in Solidity | variable in Solidity | function in Solidity++ | variable in Solidity++ |
|----------|----------------------|----------------------|------------------------|------------------------|
| public   | Possible             | Possible             | -                      | Possible               |
| external | Possible             | -                    | Possible               | -                      |
| internal | Possible             | Possible             | Possible               | Possible               |
| private  | Possible             | Possible             | Possible               | Possible               |
| offchain | -                    | -                    | Possible               | -                      |

In general, offchain functions have the following characteristics:

* As a *readonly* function, it is only used to retrieve the state but cannot modify it, so it should be declared as `view` state mutability.
* The compiled binary code of offchain functions is not stored on-chain, so it is not allowed to access transaction related information, such as `msg.amount` or `msg.tokenid` in an offchain function.
* It is not allowed to call an external function in an offchain function. 
* The `require` or `revert` function cannot be called in an offchain function.


## Asynchronous Functions

Solidity++ knows two kinds of function calls, asynchronous ones is not blocked while waiting for the called function to finish and synchonous ones do.

An asynchronous function is declared with the `async` keyword. It can be called by the other contract, both in synchronous and asynchronous functions.

Although it can have return values, the caller cannot get the return value immediately after the call, unless the `await` keyword is used.

The following code shows how a contract call another contract:

```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract A {
    uint public data;

    function set(uint a) external async {
        data = a;
    }
}

contract B {
    A a;
    event LogMessage(string s);
    
    constructor (address payable addr) {
        a = A(addr);
    }

    function f(uint data) external async {
        // call function set() of contract A
        a.set(data * 2);
        // do anything after calling a.set(), the code will not be blocked here
        emit LogMessage("a.set() is called");
    }
}
```

When the `a.set(data * 2)` statement is executed successfully, a Vite *request transaction* is initiated. The code of the function `set` is not executed at this moment, but executed when the corresponding *response transaction* is initiated.

In general, asynchronous functions have the following characteristics:

* The interface functions of a contract must be declared as `external async`.
* The `public` functions of a contract are no longer supported in Solidity++.
* The `internal` or `private` functions of a contract must be synchronous.

## Await Operator

To get the return value of the asynchronous function, you need to add the `await` operator before a function call.

The `await` operator is used to wait for an asynchronous function to return.

The following code shows the usage of `await`:

```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract A {
    uint public data;

    function set(uint a) external async {
        data = a;
    }

    function get() external async returns (uint) {
        return data;
    }
}

contract B {
    A a;
    event LogMessage(string s, uint d);
    
    constructor (address payable addr) {
        a = A(addr);
    }

    function f(uint data) external async {
        // call function set() of contract A
        await a.set(data * 2);
        // the code will be blocked here util a.set() returns
        emit LogMessage("a.set() is called", data);
        // call function get() of contract A
        uint result = await a.get();
        // the code will be blocked here util a.get() returns
        emit LogMessage("a.get() is called", result);
    }
}
```

When the `await a.get()` statement is executed, the contract B stops after initiating a Vite *request transaction*. When the corresponding *response transaction* is initiated, the code of the function `get` is executed and a *request transaction* from A is initiated to send return data back to B.

The Solidity++ compiler generates callback function entries for each await expression. When contract B discovers the returned *request transaction* from A, it will unpack the *calldata* to parse the return arguments and the callback *function selector* (4-bytes hash), then jump into the code after the await expression. At this time, the return data of `a.get()` is ready for the following assignment statement, and the rest of the function code can be executed.

::: tip Notice
The `await` operator is only valid in an asynchronous function.
:::

## Getter Functions

Same as Solidity, the Solidity++ compiler automatically creates getter functions for all public state variables. 

For the contract given below, the compiler will generate a function called `data` that does not take any arguments and returns a `uint`, the value of the state variable `data`.

```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract C {
    uint public data = 42;

    // Getter function generated by the compiler
    /* 
    function data() offchain view returns (uint) {
        return data;
    }
    */
}
```
::: tip Notice
The auto-generated getter functions have `offchain` visibility. It can **NOT** be called in a contract.
:::

If you have a public state variable of array type, then you can only retrieve single elements of the array via the auto-generated getter function. You can use arguments to specify which individual element to return, for example `myArray(0)`. 

If you want to return an entire array in one call, then you need to write a custom offchain function, for example:

```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract arrayExample {
    // public state variable
    uint[] public myArray;

    // Getter function generated by the compiler
    /*
    function myArray(uint i) offchain view returns (uint) {
        return myArray[i];
    }
    */

    // function that returns entire array
    function getArray() offchain view returns (uint[] memory) {
        return myArray;
    }
}
```

Now you can use getArray() to retrieve the entire array, instead of myArray(i), which returns a single element per call.

## Call Options
When calling functions of other contracts, you can specify the amount of a specific token sent with the call with the call options `{token: "tti_564954455820434f494e69b5", amount: 1e18}`.

```javascript
// SPDX-License-Identifier: GPL-3.0
pragma soliditypp >=0.8.0 <0.9.0;

contract InfoFeed {
    function info() external async payable returns (uint ret) {
        return 42;
    }
}

contract Consumer {
    InfoFeed feed;

    constructor payable (address payable addr) {
        feed = InfoFeed(addr);
    }

    function callFeed() external async {
        // send 1 VX with the call
        feed.info{token: "tti_564954455820434f494e69b5", amount: 1e18}();
    }
}
```

::: tip Notice
The `gas`,`salt` and `value` keys in a call option are not supported in Solidity++.
:::

## Units and Globally Available Variables in Solidity++

### VITE Units

In Solidity, a literal number can take a suffix of `wei`, `gwei` or `ether` to specify a subdenomination of Ether, where Ether numbers without a postfix are assumed to be `wei`.

Similarly, in Solidity++, a literal number can take a suffix of `attov` and `vite` to specify a subdenomination of VITE, where VITE numbers without a postfix are assumed to be `attov`.

```javascript
1 vite = 1e18 attov
```

### Magic Variables and Functions

#### Block and Transaction Properties

Below magic variables and functions has been removed from Solidity:

```javascript
tx.gasprice;
block.coinbase;
block.difficulty;
block.gaslimit;
blockhash(param);
gasleft();
msg.gas;
selfdestruct(_owner);
suicide(_addr);
```

Below magic variables and functions has been added in Solidity++:

```javascript
bytes32 b1 = fromhash();
uint height = accountheight();
bytes32 b2 = prevhash();
uint64 random = random64();
uint64 random = nextrandom();
```

* `fromhash()` returns request transaction's hash
* `accountheight()` returns latest block height of an account
* `prevhash()` returns latest block hash of an account
* `random64()` returns a random number in `uint64`. This function will return the same random number in one transaction
* `nextrandom()` returns a random number in `uint64`. This function can be called multiple times to obtain different random numbers in one transaction

Functions `address` and `tokenId` are redefined in Solidity++:

```
tokenId token01 = tokenId("tti_2445f6e5cde8c2c70e446c83");
tokenId token02 = "tti_2445f6e5cde8c2c70e446c83";
address addr01 = address("vite_0102030405060708090807060504030201020304eddd83748e");
address addr02 = "vite_0102030405060708090807060504030201020304eddd83748e";
```

Get transfer value in Solidity:
```
msg.value
```

Get transfer value in Solidity++(`value` has been changed to `amount`):
```
msg.amount;
```

Get transfer token id(Solidity doesn't have the feature):
```
msg.tokenid;
```

#### Members of Address Types
Get account balance in Solidity:

```
address.balance
```

Get account balance in Solidity++ where a parameter of tokenId is required:
```
address.balance(_tokenId)
```

In Solidity, send ETH to an address:

```
address(_addr).transfer(_amount);
```

In Solidity++, send some amount of a specific token to an address:

```
address(_addr).transfer(_tokenId, _amount);
```

#### Cryptographic Functions

A hash function of `blake2b` is introduced in Solidity++.
```
blake2b(bytes memory) returns (bytes32)
```
* Compute the Blake2b hash of the input.

## Contribution

The Solidity++ is open sourced [here](https://github.com/vitelabs/soliditypp). 

We welcome contributions from anyone interested, including reporting issues, fixing bus, commit new features and improving or translating the documentations.