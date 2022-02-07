---
order: 1
---

# Migrating from Solidity

## Solidity Compatibility

Solidity++ 0.8 is compatible with almost all syntax of Solidity except for the differences mentioned in this article.

All features related to Ethereum are replaced with that related to Vite.

The `new` keyword is disabled in this version. In other words, it is impossible to deploy a new contract from a contract at runtime.

In addition, `selfdestruct` and `ecrecover` are disabled in this version.


## Types
The `address` type is redefined to be compatible with Vite, and a new type `vitetoken` representing Vite Native Token Id is introduced in Solidity++.

```javascript
// address type
address addr1 = address("vite_0102030405060708090807060504030201020304eddd83748e");
address addr2 = "vite_0102030405060708090807060504030201020304eddd83748e";
address payable addr3 = payable("vite_0102030405060708090807060504030201020304eddd83748e"); 
// vitetoken type
vitetoken token1 = vitetoken("tti_2445f6e5cde8c2c70e446c83");
vitetoken token2 = "tti_2445f6e5cde8c2c70e446c83";
```

## Call Options
When calling functions of other contracts, you can specify the amount of a specific token sent with the call with the call options `{token: "tti_564954455820434f494e69b5", value: 1e18}`.

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
        feed.info{token: "tti_564954455820434f494e69b5", value: 1e18}();
    }
}
```

If the `token` call option is omited, the *VITE* token is sent by default.

The following code:
```javascript
feed.info{value: 1e18}();
```
equivalents to:
```javascript
feed.info{token: "tti_5649544520544f4b454e6e40", value: 1e18}();
```

::: tip Notice
The `gas`,`salt` keys in a call option are not supported in Solidity++.
:::

## Units

In Solidity, a literal number can take a suffix of `wei`, `gwei` or `ether` to specify a subdenomination of Ether, where Ether numbers without a postfix are assumed to be `wei`.

Similarly, in Solidity++, a literal number can take a suffix of `attov` and `vite` to specify a subdenomination of VITE, where VITE numbers without a postfix are assumed to be `attov`.

```javascript
1 vite = 1e18 attov
```

## Magic Variables and Functions

### Block and Transaction Properties

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


Get transfer value (ether only) in Solidity:
```javascript
uint amount = msg.value
```

Get transfer value and token id in Solidity++:
```javascript
uint amount = msg.value;
vitetoken token = msg.token;
```

### Members of Address Types

Get account balance in Solidity:

```javascript
address.balance
```

Get account balance in Solidity++, balance is a function with a single parameter of `vitetoken`:
```javascript
address.balance("tti_564954455820434f494e69b5");
```

In Solidity, send ETH to an address:

```javascript
payable(_addr).transfer(_amount);
```

In Solidity++, send some amount of a specific token to an address:

```javascript
payable(_addr).transfer(_tokenId, _amount);
```

### Cryptographic Functions

A hash function of `blake2b` is introduced in Solidity++.
```javascript
blake2b(bytes memory) returns (bytes32)
```
* Compute the Blake2b hash of the input.



