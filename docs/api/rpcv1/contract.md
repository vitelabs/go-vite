---
sidebarDepth: 4
---

# Contract

:::tip Maintainer
[viteLiz](https://github.com/viteLiz)
:::

## Create Contract

Essentially, creating contract is sending a special transaction whose transaction type is 1 with contract's code and specified parameters. It has the following steps.

1. Call RPC method `contract_getCreateContractToAddress` to generate a new contract address
2. Encode passed-in parameters according to ABI. This step can be completed by calling either `abi.encodeParameters` in **vite.js**, or `contract_getCreateContractParams` in RPC
3. Call RPC method `contract_getCreateContractData` to generate transaction data
4. Call RPC method `tx_sendRawTx` to create contract. Here parameter `toAddress` is the contract address generated in Step 1. `data` is transaction data created in Step 3. `blockType` is set to 1, indicating creating contract transaction. `amount` and `tokenId` stand for the amount of token that are transferred to the contract during creation. `fee` is the cost, fixed at 10 VITE in Pre-Mainnet.

:::tip Tips
Above steps are implemented in method `builtinTxBlock.createContract` in **vite.js** 
:::

## Call Contract

Essentially, calling contract is sending a special transaction to smart contract's account, specifying calling method name and parameters. 

1. Encode method name and passed-in parameters according to ABI. This step can be completed by calling either `abi.encodeFunctionCall` method in **vite.js** (recommended), or `contract_getCallContractData` in RPC
2. Call RPC method `tx_sendRawTx` to send a transaction to contract. Here parameter `toAddress` is the contract address. `data` is transaction data created in Step 1. `blockType` is set to 2 for calling contract. `amount` and `tokenId` stand for the amount of token that are transferred to the contract. `fee` is 0, standing for free.

:::tip Tips
Above steps are implemented in method `builtinTxBlock.callContract` in **vite.js** 
:::

## Read States Off-chain

Contract's states can be accessed off-chain through `getter` methods. The ABI definition of `getter` and off-chain code are generated when the contract is compiled. 

1. Encode `getter` method name and passed-in parameters according to ABI. This step can be completed by calling either `abi.encodeFunctionCall` method in **vite.js** (recommended), or `contract_getCallOffChainData` in RPC
2. Call RPC method `contract_callOffChainMethod` to visit contract's state

## Call Built-in Contract

Similar to calling common contract, calling built-in contract is sending a special transaction to built-in contract's account, specifying calling method name and parameters. 
Vite has implemented [Staking](./pledge.html), [Token Issuance](./mintage.html) and [Consensus](./consensus.html) 3 built-in smart contracts.

1. Generate transaction data by calling relevant `xxx_getxxxData` RPC method. For example, call `mintage_getMintData` when issuing new token.
2. Call RPC method `tx_sendRawTx` to send the transaction to built-in contract.

:::tip Tips
Most methods of built-in contracts are provided in `builtinTxBlock` in **vite.js** 
:::

## API

## contract_getCreateContractToAddress
Return a new address for contract

- **Parameters**: 

  * `Address`: The address of requester
  * `uint64`: Current account block height
  * `Hash`: The hash of previous account block

- **Returns**: 
	- `Address` New contract address

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"contract_getCreateContractToAddress",
   "params":[
      "vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6", 
      "2", 
      "3a56babeb0a8140b12ac55e91d2e05c41f908ebe99767b0e4aa5cd7af22d6de7", 
      "3a56babeb0a8140b12ac55e91d2e05c41f908ebe99767b0e4aa5cd7af22d6de7"]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": "vite_22f4f195b6b0f899ea263241a377dbcb86befb8075f93eeac8"
}
```
:::

## contract_getCreateContractParams
Encode passed-in parameters for creating contract

- **Parameters**: 

  * `string`: Contract's ABI definition
  * `[]string`: Passed-in parameters. Element is either a string for simple type or JSON string for complex

- **Returns**: 
	- `[]byte` Data

- **Example**:

::: demo

```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "contract_getCreateContractParams",
    "params": [
        "[{\"constant\":false,\"inputs\":[{\"name\":\"voter\",\"type\":\"address\"}],\"name\":\"authorization\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"proposal\",\"type\":\"uint256\"}],\"name\":\"vote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"proposalNames\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]",
        ["[\"0x1111111111111111111111111111111111111111111111111111111111111111\",\"0x2222222222222222222222222222222222222222222222222222222222222222\"]"]
    ]
}
```
:::

## contract_getCreateContractData
Generate request data for creating contract

- **Parameters**: 
  
  * `gid`:`Gid`: The id of delegated consensus group. A contract must assign a consensus group to have contract transaction executed. For example, global delegated consensus group has id "00000000000000000002"
  * `confirmTime`:`uint8`: The number of snapshot blocks (between 0-75) by which request sent to this contract is confirmed before responding to the transaction. 0 stands for no additional confirmation required. This parameter must be positive if random number, timestamp or snapshot block height is used in the contract. If $confirmTime>0$, additional quota will be consumed for response 
  * `seedCount`:`uint8`: The number of snapshot blocks (between 0-75) having random seed by which request sent to this contract is confirmed before responding to the transaction. 0 stands for no additional confirmation required. This parameter must be positive if random number is used in the contract. Having $confirmTime>=seedCount$
  * `quotaRatio`:`uint8`: Quota multiply factor between 10-100. Additional quota will be consumed for contract's caller when a number above 10 is passed in. For example, quotaRatio=15 means 1.5 times of quota will be charged for caller.
  * `hexCode`:`string`: The hex code of contract
  * `params`:`[]byte`: Encoded parameters, returned by `contract_getCreateContractParams`

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "contract_getCreateContractData",
    "params": [{
        "gid":"00000000000000000002",
        "confirmTime":2,
        "seedCount":1,
        "quotaRatio":10,
        "hexCode":"608060405234801561001057600080fd5b506101ca806100206000396000f3fe608060405260043610610041576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806380ae0ea114610046575b600080fd5b6100bd6004803603602081101561005c57600080fd5b810190808035906020019064010000000081111561007957600080fd5b82018360208201111561008b57600080fd5b803590602001918460208302840111640100000000831117156100ad57600080fd5b90919293919293905050506100bf565b005b60006002838390508115156100d057fe5b061415156100dd57600080fd5b600080905060008090505b8383905081101561018a576000848483818110151561010357fe5b9050602002013590506000858560018501818110151561011f57fe5b905060200201359050808401935080841015151561013c57600080fd5b600081111561017d578173ffffffffffffffffffffffffffffffffffffffff164669ffffffffffffffffffff168260405160405180820390838587f1505050505b50506002810190506100e8565b50348114151561019957600080fd5b50505056fea165627a7a723058203cef4a3f93b33e64e99e0f88f586121282084394f6d4b70f1030ca8c360b74620029", 
        "params":""
    }]
}
```

:::

## contract_getCallContractData
Generate request data for calling contract

- **Parameters**: 

  * `string`: Contract's ABI definition
  * `string`: Calling method name
  * `[]string`: Passed-in parameters. Element is either a string for simple type or JSON string for complex

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "contract_getCallContractData",
    "params": [
        "[{\"constant\":false,\"inputs\":[{\"name\":\"voter\",\"type\":\"address\"}],\"name\":\"authorization\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"proposal\",\"type\":\"uint256\"}],\"name\":\"vote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"proposalNames\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]",
        "vote",
        ["0x1111111111111111111111111111111111111111111111111111111111111111"]
    ]
}
```

:::

## contract_getContractInfo
Return contract information by specified contract address

- **Parameters**: 

  * `string`: Contract address
  
- **Returns**: 
	`ContractInfo`
    1. `code`: `[]byte`  Contract's code
    2. `gid`: `Gid`  The id of delegated consensus group that the contract has assigned
    3 `confirmTime`:`uint8`: The number of snapshot blocks by which request sent to this contract is confirmed before responding to the transaction
    4 `seedCount`:`uint8`: The number of snapshot blocks having random seed by which request sent to this contract is confirmed before responding to the transaction

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "contract_getContractInfo",
    "params": ["vite_22f4f195b6b0f899ea263241a377dbcb86befb8075f93eeac8"]
}
```
:::

## contract_getCallOffChainData
Encode passed-in parameters for `getter` method. The returned value can be passed in method `contract_callOffChainMethod` as data input

- **Parameters**: 

  * `string` Contract's ABI definition
  * `string` `getter` method name
  * `[]string` Passed-in parameters
  
- **Returns**: 
	`[]byte` 

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "contract_getCallOffChainData",
    "params": [
      "[{\"constant\":true,\"inputs\":[],\"name\":\"getTokenList\",\"outputs\":[{\"name\":\"\",\"type\":\"tokenId[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"offchain\"}]",
      "getTokenList",
      []
    ]
}
```
:::

## contract_callOffChainMethod
Query contract's state by calling the specified `getter` method off-chain. Please note that only one parameter of either `offchainCode` or `offChainCodeBytes` is needed.

- **Parameters**: 

  * `Object`:
    * `selfAddr`:`Address` Contract address
    * `offChainCode`:`string` Hex code generated as `OffChain Binary` when compiling the contract by specifying `--bin` argument
    * `offChainCodeBytes`:`[]byte` Base64 byte code generated as `OffChain Binary` when compiling the contract by specifying `--bin` argument
    * `Data`:`[]byte` Encoded parameters, returned by `contract_getCallOffChainData`
    
- **Returns**: 
	`[]byte` Return value of the `getter` method

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "contract_callOffChainMethod",
    "params": [{
      "selfAddr":"vite_22f4f195b6b0f899ea263241a377dbcb86befb8075f93eeac8",
      "offChainCodeBytes":"YIBgQFJgBDYQYQBQV2AANXwBAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJAEY/////8WgGO+RoE6FGEAVFeAY/EnHggUYQCxV2EAUFZbW1sAW2EAjWAEgDYDYCCBEBVhAGtXYABgAP1bgQGQgIA1af////////////8WkGAgAZCSkZBQUFBhARFWW2BAUYCEgVJgIAGDgVJgIAGCgVJgIAGTUFBQUGBAUYCRA5DzW2EAuWEB0VZbYEBRgIBgIAGCgQOCUoOBgVGBUmAgAZFQgFGQYCABkGAgAoCDg2AAW4OBEBVhAP1XgIIBUYGEAVJbYCCBAZBQYQDhVltQUFBQkFABklBQUGBAUYCRA5DzW2AAYABgAGACYABQYACFaf////////////8Waf////////////8WgVJgIAGQgVJgIAFgACFgAFBgAAFgAFBUYAJgAFBgAIZp/////////////xZp/////////////xaBUmAgAZCBUmAgAWAAIWAAUGABAWAAUFRgAmAAUGAAh2n/////////////Fmn/////////////FoFSYCABkIFSYCABYAAhYABQYAIBYABQVJJQklCSUGEBylZbkZOQklBWW2BgYAFgAFCAVIBgIAJgIAFgQFGQgQFgQFKAkpGQgYFSYCABgoBUgBVhAlpXYCACggGRkGAAUmAgYAAhkGAAkFuCgpBUkGEBAAqQBGn/////////////Fmn/////////////FoFSYCABkGAKAZBgIIJgCQEEkoMBkmABA4ICkVCAhBFhAhFXkFBbUFBQUFCQUGECZlZbkFb+oWVienpyMFgg9JX2H2l/JeRsqoaMCbNbV1qzMePGCBeYgOGTK1hIq6oAKQ==",
      "data":"f1271e08"
    }]
}
```
:::
