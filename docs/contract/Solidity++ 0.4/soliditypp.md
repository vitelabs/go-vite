---
order: 1
---

# Introduction to Solidity++ 0.4

Solidity++ retains most syntax of Solidity. However, Solidity++ is still a different language. The difference mainly lies in asynchronous design.

## Syntax Removed in Solidity++

Below syntax has been removed from Solidity

```
tx.gasprice;
block.coinbase;
block.difficulty;
block.gaslimit;
blockhash(param);
gasleft();
msg.gas;
selfdestruct(_owner);
suicide(_addr);
address(_addr).send(_amount);
```

Syntax related to `ecrecover` and `ripemd160` has been disabled

`DELEGATECALL` is not available in Solidity++ at this moment

Syntax of inline assembly in Solidity is temporarily unavailable


## Syntax Added/Modified in Solidity++

Below syntax has been added in Solidity++

```
bytes32 b1 = fromhash();
uint height = accountheight();
bytes32 b2 = prevhash();
uint64 random = random64();
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

Obtain transfer value in Solidity:
```
msg.value
```

Obtain transfer value in Solidity++(`value` has been changed to `amount`):
```
msg.amount;
```

Obtain transfer token id(Solidity doesn't have the feature):
```
msg.tokenid;
```

Obtain account balance in Solidity:

```
address.balance
```

Obtain account balance in Solidity++(a parameter of tokenId is required):
```
address.balance(_tokenId)
```

In Solidity, send ETH to an address:

```
address(_addr).transfer(_amount);
```

In Solidity++, send specific token to an address:

```
address(_addr).transfer(_tokenId, _amount);
```

Unit of ETH in Solidity: wei/szabo/finney/ether

1 ether = 1000 finney, 1 finney = 1000 szabo, 1 szabo = 1000000000000 wei

Unit of VITE in Solidity++: attov/vite

1 vite = 1000000000000000000 attov

In Solidity++, all syntax related to `sha256` or `sha3` is replaced by `blake2b`

## Asynchronous Syntax in Solidity++

Unlike Solidity, in-contract communication in Solidity++ is not done through function calls, but by message passing

In Solidity++, 

* `public` functions cannot be accessed from outside of the contract
* `function` cannot be declared as `external`. Instead, `function` can only be declared as `public`, `private` or `internal`
* `public` static variable is not visible from outside of the contract

An example

```
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
    uint total;
    message add(uint a, uint b);

    onMessage invoker(address addr, uint a, uint b) {
       send(addr, add(a, b));
    }

    onMessage sum(uint result) {
        if (result > 10) {
           total += result;
       }
    }
}
```

* `message`: keyword, declaring a message, including message name and passed-in parameter. In above example, `message sum(uint result)` declares message "sum" accepting a `uint` parameter.

* `onMessage`: keyword, declaring a message listener, including message name, passed-in parameter and business logic that handles the message. In above example, `onMessage add(uint a, uint b)` declares message listener "add" accepting two `uint` parameters.

* `send`: keyword, sending a message, accepting two parameters including message receiving address and message body


Message can be declared in contract and can only be sent out from the exact contract who declares it. If a message should be processed by another contract, the name and passed-in parameters of the message rely on how message listener is defined in the other contract.

In other words, if contract A would like to send a message to contract B, A must define the message in a format that complies to the listener defined in contract B.

Message listener is defined in contract. A message listener specifies a certain type of message that can be processed by the contract. Unrecognized messages will not be handled but discarded.

:::warning
Message listeners cannot be called directly as normal functions.
:::

In above example,

Contract A defines a message listener `add(uint a, uint b)` while contract B defines `sum(uint result)`, indicating the messages that can be processed by contract A or contract B.

Here contract B sends message to contract A. For this purpose, contract B must declare "add" message which complies to `add(uint a, uint b)` defined in contract A. 
Similarly, contract A should declare "sum" message matching message listener `sum(uint result)` of contract B, since contract A will send such a message to contract B in return.

## `getter` in Solidity++

In Solidity++, because interactions between contracts are carried out through asynchronous messaging, the `public` fields cannot be accessed directly from outside of the contract. 
To address this problem, Solidity++ provides a solution.

```
pragma soliditypp ^0.4.3;
contract A {

    uint magic = 0;
   
    getter getMagic() returns(uint256) {
        return magic;
    }

    getter getMagicAdd(uint256 m) returns(uint256) {
        return calculate(m);
    }
    
    function calculate(uint256 m) public view returns(uint256) {
        return m + magic;
    }
}
```

As shown in above example, a "getter" method `getMagic()` is defined to access public field `magic`. In general, "getter" in Solidity++ has the following characteristics:

* Keyword `getter` is used to declare such a method.
* Method name, input parameters and return value are required. As query method, it is only used to obtain the specific field's value and cannot modify it.
* The compiled code of "getter" is not stored on chain, so it is not allowed to access transaction related information, such as `msg.amount` or `msg.tokenid`, within "getter".
* "getter" cannot send transactions or send messages. It cannot call `require` or `revert` either.
* "getter" can call functions, which should be defined as `view` type.

## Example of Solidity++ Contract

Below contract defines a batch transfer which accepts a list of addresses and amounts and then transfers specified amount to specified address in the list

```
// Declare the contract is written in soliditypp 0.4.3. Backwards compatibility is guaranteed for generating the same compiling result.
pragma soliditypp ^0.4.3;
 

// Define contract A
contract A {
     // Define a message listener. Every external method should be defined as message listener in Solidity++
     // Define listener name and parameters. Visibility is not necessary. Message listener has no return value
     // Messsage listener "transfer" is defined with a passed-in parameter of a uint array, in format of address in odd element and amount in even
     onMessage transfer(uint[] calldata body) payable {
         // Check if the parameter length is even because each address should have corresponding amount
         require(body.length%2==0);
         uint256 totalAmount = 0;
         for(uint i = 0; i < body.length; i=i+2) {
             uint addr = body[i];
             uint amount = body[i+1];
             totalAmount = totalAmount + amount;
             require(totalAmount >= amount);
             if(amount > 0) {
                // Transfer to address
                address(addr).transfer(msg.tokenid, amount);
             }
         }
         require(totalAmount == msg.amount);
     }
}
```
