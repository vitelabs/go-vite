---
sidebarDepth: 4
---

# Mintage
:::tip Maintainer
[viteLiz](https://github.com/viteLiz)
:::

## Contract Specification
Built-in token issuance contract. Contract address is `vite_000000000000000000000000000000000000000595292d996d`

ABI：

```json
[
  // Issue new token
  {"type":"function","name":"Mint","inputs":[{"name":"isReIssuable","type":"bool"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"}]},
  // Re-issue. Mint additional tokens
  {"type":"function","name":"Issue","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"amount","type":"uint256"},{"name":"beneficial","type":"address"}]},
  // Burn
  {"type":"function","name":"Burn","inputs":[]},
  // Transfer ownership of re-issuable token
  {"type":"function","name":"TransferOwner","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"newOwner","type":"address"}]},
  // Change token type from re-issuable to non-reissuable
  {"type":"function","name":"ChangeTokenType","inputs":[{"name":"tokenId","type":"tokenId"}]},
  // Query token information
  {"type":"function","name":"GetTokenInfo","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"bid","type":"uint8"}]},
  // Callback function for token information query
  {"type":"callback","name":"GetTokenInfo","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"bid","type":"uint8"},{"name":"exist","type":"bool"},{"name":"decimals","type":"uint8"},{"name":"tokenSymbol","type":"string"},{"name":"index","type":"uint16"},{"name":"owner","type":"address"}]},
  // Token issued event
  {"type":"event","name":"mint","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
  // Token re-issued event
  {"type":"event","name":"issue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
  // Token burned event
  {"type":"event","name":"burn","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"address","type":"address"},{"name":"amount","type":"uint256"}]},
  // Ownership transferred event
  {"type":"event","name":"transferOwner","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"owner","type":"address"}]},
  // Token type changed event
  {"type":"event","name":"changeTokenType","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]}
]
```

Querying token information function will return execution results in callback.

## mintage_getMintData
Generate request data for issuing new token. Equivalent to `Mint` method in ABI.

- **Parameters**: 

`Object`  
  1. `tokenName`:`string`  Token name in 1-40 characters, including uppercase/lowercase letters, spaces and underscores. Cannot contain consecutive spaces or begin/end by space
  2. `tokenSymbol`: `string` Token symbol in 1-10 characters, including uppercase/lowercase letters, spaces and underscores. 
  3. `totalSupply`: `big.int` Total supply. Cannot exceed $2^{256}-1$
  4. `decimals`: `uint8` Decimal digits. Having $10^{decimals}<=totalSupply$
  5. `isReIssuable`: `bool` Whether the token can be re-issued. `true` means the token is in dynamic total supply and additional tokens can be minted.
  6. `maxSupply`: `uint256` Maximum supply. Mandatory for re-issuable token. Cannot exceed $2^{256}-1$. Having $maxSupply>=totalSupply$
  7. `ownerBurnOnly`: `bool` Whether the token can be burned by the owner only. Mandatory for re-issuable token. All token holders can burn if this flag is `false`

- **Returns**: 
	- `[]byte` Data

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getMintData",
   "params": [{
   	  "tokenName":"Test Token",
   		"tokenSymbol":"test",
   		"totalSupply":"100000000000",
   		"decimals":6,
   		"isReIssuable":true,
   		"maxSupply":"200000000000",
   		"ownerBurnOnly":true
   	}]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": "cbf0e4fa000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000174876e80000000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000002e90edd0000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000a5465737420546f6b656e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000047465737400000000000000000000000000000000000000000000000000000000"
}
```
:::

## mintage_getIssueData
Generate request data for re-issuing the specified amount of tokens. Equivalent to `Issue` method in ABI.

- **Parameters**: 

`Object`
  1. `tokenId`: `TokenId`  Token ID
  2. `amount`: `uint64`  Re-issuance amount
  3. `beneficial`: `Hash`  The account address to receive newly minted tokens


- **Returns**: 
	- `[]byte` Data

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getIssueData",
   "params": [{
   	  "tokenId":"tti_5649544520544f4b454e6e40",
   		"amount":"100000000000",
   		"beneficial":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"
   	}]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": "1adb5572000000000000000000000000000000000000000000005649544520544f4b454e000000000000000000000000000000000000000000000000000000174876e800000000000000000000000000a5a7f08011c2f0e40ccd41b5b79afbfb818d565f"
}
```
:::

## mintage_getBurnData
Generate request data for burning token. Equivalent to `Burn` method in ABI.

- **Parameters**: 

- **Returns**: 

	- `[]byte` Data

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getBurnData",
   "params": []
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": "b7bf21b4"
}
```
:::

## mintage_getTransferOwnerData
Generate request data for transferring token ownership. Equivalent to `TransferOwner` method in ABI.

- **Parameters**: 

`Object`
  1. `TokenId`: Token ID
  2. `Address`: The account address of new owner

- **Returns**: 

	- `[]byte` Data

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getTransferOwnerData",
   "params": [{
      "tokenId":"tti_251a3e67a41b5ea2373936c8",
      "newOwner":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"
   }]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": "a659fe5a00000000000000000000000000000000000000000000251a3e67a41b5ea23739000000000000000000000000a5a7f08011c2f0e40ccd41b5b79afbfb818d565f"
}
```
:::

## mintage_getChangeTokenTypeData
Generate request data for changing token type. This method can only change re-issuable token to non-reissuable. Equivalent to `ChangeTokenType` method in ABI.
- **Parameters**: 

  * `TokenId`: Token ID

- **Returns**: 
	- `[]byte` Data

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getChangeTokenTypeData",
   "params": ["tti_5649544520544f4b454e6e40"]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": "7ecfb4d7000000000000000000000000000000000000000000005649544520544f4b454e"
}
```
:::

## mintage_getTokenInfoList
Return a list of all tokens issued

- **Parameters**: 

  * `int`: Page index，starting from 0
  * `int`: Page size

- **Returns**: 

`Array<TokenInfo>`
  1. `tokenName`: `string`  Token name
  2. `tokenSymbol`: `string`  Token symbol
  3. `totalSupply`: `big.Int` Total supply
  4. `decimals`: `uint8` Decimal digits
  5. `owner`: `Address` Token owner
  6. `isReIssuable`: `bool`  Whether the token can be re-issued
  7. `maxSupply`: `big.Int`  Maximum supply
  8. `ownBurnOnly`: `bool`  Whether the token can be burned by the owner only
  9. `tokenId`: `TokenId` Token ID
  10. `index`: `uint16` Token index between 0-999. For token having the same symbol, sequential indexes will be allocated according to when the token is issued.

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getTokenInfoList",
   "params":[0, 10]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": [{
      "tokenName":"Vite Token",
      "tokenSymbol":"VITE",
      "totalSupply":"1000000000000000000000000000",
      "decimals":18,
      "owner":"vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122",
      "isReIssuable":false,
      "maxSupply":"0",
      "ownBurnOnly":false,
      "tokenId":"tti_5649544520544f4b454e6e40",
      "index":0
   }]
}
```
:::

## mintage_getTokenInfoById
Return token info by ID

- **Parameters**: 

  * `TokenId`: Token ID

- **Returns**: 

`TokenInfo`  
  1. `tokenName`: `string`  Token name
  2. `tokenSymbol`: `string`  Token symbol
  3. `totalSupply`: `big.Int` Total supply
  4. `decimals`: `uint8` Decimal digits
  5. `owner`: `Address` Token owner
  6. `isReIssuable`: `bool`  Whether the token can be re-issued
  7. `maxSupply`: `big.Int`  Maximum supply
  8. `ownBurnOnly`: `bool`  Whether the token can be burned by the owner only
  9. `tokenId`: `TokenId` Token ID
  10. `index`: `uint16` Token index between 0-999. For token having the same symbol, sequential indexes will be allocated according to when the token is issued.

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getTokenInfoById",
   "params":["tti_5649544520544f4b454e6e40"]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": {
      "tokenName":"Vite Token",
      "tokenSymbol":"VITE",
      "totalSupply":"1000000000000000000000000000",
      "decimals":18,
      "owner":"vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122",
      "isReIssuable":false,
      "maxSupply":"0",
      "ownBurnOnly":false,
      "tokenId":"tti_5649544520544f4b454e6e40",
      "index":0
   }
}
```
:::

## mintage_getTokenInfoListByOwner
Return a list of tokens issued by the specified owner

- **Parameters**: 

  * `Address`: The account address of token owner

- **Returns**: 

`Array<TokenInfo>`
  1. `tokenName`: `string`  Token name
  2. `tokenSymbol`: `string`  Token symbol
  3. `totalSupply`: `big.Int` Total supply
  4. `decimals`: `uint8` Decimal digits
  5. `owner`: `Address` Token owner
  6. `isReIssuable`: `bool`  Whether the token can be re-issued
  7. `maxSupply`: `big.Int`  Maximum supply
  8. `ownBurnOnly`: `bool`  Whether the token can be burned by the owner only
  9. `tokenId`: `TokenId` Token ID
  10. `index`: `uint16` Token index between 0-999. For token having the same symbol, sequential indexes will be allocated according to when the token is issued.

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"mintage_getTokenInfoListByOwner",
   "params":["vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": [{
      "tokenName":"Test Token",
      "tokenSymbol":"test",
      "totalSupply":"100000000000",
      "decimals":6,
      "owner":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
      "isReIssuable":true,
      "maxSupply":"200000000000",
      "ownBurnOnly":true,
      "tokenId":"tti_251a3e67a41b5ea2373936c8",
      "index":0
   }]
}
```
:::
