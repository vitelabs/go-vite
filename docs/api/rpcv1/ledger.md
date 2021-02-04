# Ledger
:::tip Maintainer
[lyd00](https://github.com/lyd00)
:::

## ledger_getBlocksByAccAddr
Return transaction list of the specified account

- **Parameters**:

  * `string`: `Addr`  Account address
  * `int`:  `Index` Page index
  * `int`: `Count`  Page size


- **Returns**:  `Array<AccountBlock>`
  
- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 17,
	"method": "ledger_getBlocksByAccAddr",
	"params": ["vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68", 0, 10]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 17,
    "result": [
        {
            "blockType": 4,
            "hash": "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
            "prevHash": "0000000000000000000000000000000000000000000000000000000000000000",
            "accountAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
            "publicKey": "OvmkehEUDGgcKyqFpM6Yf6sGklibLOIzv34XS9QwF3o=",
            "toAddress": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "fromBlockHash": "5113171e23ac1cdfcb6851f9bea7ad050058acccbe2e6faf8f5a2231f02c5f7c",
            "tokenId": "tti_5649544520544f4b454e6e40",
            "receiveBlockHeight": "",
            "receiveBlockHash": null,
            "snapshotHash": "fc08446111289c671fe1547f634afcde92fab289c11fe16380958305b2f379ad",
            "data": null,
            "timestamp": "2018-10-11T01:21:45.899730786+08:00",
            "stateHash": "53af30da1fc818c9a03ef539aadf7a1e0c90039d5c4eb42143dd9cfc211adbe6",
            "logHash": "0000000000000000000000000000000000000000000000000000000000000000",
            "nonce": "1GO9X2PtbDM=",
            "signature": "rVA04yeWgERnmzVJ0LsLqIEkjn6r2BrePyxOCS2N4l+UKy3mjaIWO5ybk8sc6qiVR91reEwXHwyfeFo+CjNNCg==",
            "height": "1",
            "quota": "0",
            "amount": "1000000000000000000000000000",
            "fee": "0",
            "confirmedTimes": "0",
            "tokenInfo": {
                "tokenName": "Vite Token",
                "tokenSymbol": "VITE",
                "totalSupply": "1000000000000000000000000000",
                "decimals": 18,
                "owner": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
                "pledgeAmount": "0",
                "withdrawHeight": "0"
            }
        }
    ]
}

```

```json test
{
	"jsonrpc": "2.0",
	"id": 17,
	"method": "ledger_getBlocksByAccAddr",
	"params": ["vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68", 0, 10]
}
```
:::


## ledger_getAccountByAccAddr
Return account info by address, including account height and token balance information

- **Parameters**: 
  * string: Account address

- **Returns**:

  `Object` : Detailed account information
   -  `AccountAddress` : `string of addr` Account address
   -  `TokenBalanceInfoMap` : `Map<string of TokenTypeId>token` Balance map in various token IDs
   -  `TotalNumber` : `string of uint64` Total transaction number of the account, equivalent to chain's height

- **Example**:

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 5,
	"method": "ledger_getAccountByAccAddr",
	"params": ["vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68"]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 5,
    "result": {
        "accountAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "totalNumber": "1",
        "tokenBalanceInfoMap": {
            "tti_5649544520544f4b454e6e40": {
                "tokenInfo": {
                    "tokenName": "Vite Token",
                    "tokenSymbol": "VITE",
                    "totalSupply": "1000000000000000000000000000",
                    "decimals": 18,
                    "owner": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
                    "pledgeAmount": "0",
                    "withdrawHeight": "0"
                },
                "totalAmount": "1000000000000000000000000000",
                "number": null
            }
        }
    }
}
```
```json test
{
	"jsonrpc": "2.0",
	"id": 5,
	"method": "ledger_getAccountByAccAddr",
	"params": ["vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68"]
}
```
:::

## ledger_getLatestSnapshotChainHash
Return the latest snapshot block hash

- **Parameters**: null 

- **Returns**: `Hash` The hash of latest snapshot block

- **Example**:
::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "ledger_getLatestSnapshotChainHash",
    "params": null
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 0,
    "result": "7f1b1c35a3f9c05cd621388cd8756240fad56568f777098b394005037237319e"
}
```
```json test
{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "ledger_getLatestSnapshotChainHash",
    "params": null
}
```
::: 

## ledger_getLatestBlock
Return the latest account block

- **Parameters**: `Address` Account address

- **Returns**: `AccountBlock` Latest account block 

- **Example**:
::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "ledger_getLatestBlock",
    "params": [
        "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68"
    ]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 3,
    "result": {
        "blockType": 4,
        "hash": "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
        "prevHash": "0000000000000000000000000000000000000000000000000000000000000000",
        "accountAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "publicKey": "OvmkehEUDGgcKyqFpM6Yf6sGklibLOIzv34XS9QwF3o=",
        "toAddress": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
        "fromBlockHash": "5113171e23ac1cdfcb6851f9bea7ad050058acccbe2e6faf8f5a2231f02c5f7c",
        "receiveBlockHeight": "",
        "receiveBlockHash": null,
        "tokenId": "tti_5649544520544f4b454e6e40",
        "snapshotHash": "fc08446111289c671fe1547f634afcde92fab289c11fe16380958305b2f379ad",
        "data": null,
        "timestamp": "2018-10-11T01:21:45.899730786+08:00",
        "stateHash": "53af30da1fc818c9a03ef539aadf7a1e0c90039d5c4eb42143dd9cfc211adbe6",
        "logHash": "0000000000000000000000000000000000000000000000000000000000000000",
        "nonce": "1GO9X2PtbDM=",
        "signature": "rVA04yeWgERnmzVJ0LsLqIEkjn6r2BrePyxOCS2N4l+UKy3mjaIWO5ybk8sc6qiVR91reEwXHwyfeFo+CjNNCg==",
        "height": "1",
        "quota": "0",
        "amount": "1000000000000000000000000000",
        "fee": "0",
        "confirmedTimes": "0",
        "tokenInfo": {
            "tokenName": "Vite Token",
            "tokenSymbol": "VITE",
            "totalSupply": "1000000000000000000000000000",
            "decimals": 18,
            "owner": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
            "pledgeAmount": "0",
            "withdrawHeight": "0"
        }
    }
}
```
```json test
{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "ledger_getLatestBlock",
    "params": ["vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68"]
}
```
::: 

## ledger_getBlockByHeight
Return an account block by address and height

- **Parameters**: 
    - `string` : `address` Account address
    - `string` : `height`  The height of account block

- **Returns**: `AccountBlock` Account block

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlockByHeight",
    "params": [
        "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "1"
    ]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": [
        {
            "blockType": 4,
            "hash": "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
            "prevHash": "0000000000000000000000000000000000000000000000000000000000000000",
            "accountAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
            "publicKey": "OvmkehEUDGgcKyqFpM6Yf6sGklibLOIzv34XS9QwF3o=",
            "toAddress": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "fromBlockHash": "5113171e23ac1cdfcb6851f9bea7ad050058acccbe2e6faf8f5a2231f02c5f7c",
            "tokenId": "tti_5649544520544f4b454e6e40",
            "receiveBlockHeight": "",
            "receiveBlockHash": null,
            "snapshotHash": "fc08446111289c671fe1547f634afcde92fab289c11fe16380958305b2f379ad",
            "data": null,
            "timestamp": "2018-10-11T01:21:45.899730786+08:00",
            "stateHash": "53af30da1fc818c9a03ef539aadf7a1e0c90039d5c4eb42143dd9cfc211adbe6",
            "logHash": "0000000000000000000000000000000000000000000000000000000000000000",
            "nonce": "1GO9X2PtbDM=",
            "signature": "rVA04yeWgERnmzVJ0LsLqIEkjn6r2BrePyxOCS2N4l+UKy3mjaIWO5ybk8sc6qiVR91reEwXHwyfeFo+CjNNCg==",
            "height": "1",
            "quota": "0",
            "amount": "1000000000000000000000000000",
            "fee": "0",
            "confirmedTimes": "0",
            "tokenInfo": {
                "tokenName": "Vite Token",
                "tokenSymbol": "VITE",
                "totalSupply": "1000000000000000000000000000",
                "decimals": 18,
                "owner": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
                "pledgeAmount": "0",
                "withdrawHeight": "0"
            }
        }
    ]
}
```
```json test
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlockByHeight",
    "params": [
        "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "1"
    ]
}
```
:::

## ledger_getBlockByHash
Return an account block by transaction hash

- **Parameters**: 
    - `string` : `hash`  Transaction hash

- **Returns**: `AccountBlock` Account block

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlockByHash",
    "params": [
        "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a"
    ]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": 
    {
        "blockType": 4,
        "hash": "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
        "prevHash": "0000000000000000000000000000000000000000000000000000000000000000",
        "accountAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "publicKey": "OvmkehEUDGgcKyqFpM6Yf6sGklibLOIzv34XS9QwF3o=",
        "fromAddress": "vite_00000000000000000000000000000000000000056ad6d26692",
        "toAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "fromBlockHash": "5113171e23ac1cdfcb6851f9bea7ad050058acccbe2e6faf8f5a2231f02c5f7c",
        "receiveBlockHeight": "",
        "receiveBlockHash": null,
        "tokenId": "tti_5649544520544f4b454e6e40",
        "snapshotHash": "fc08446111289c671fe1547f634afcde92fab289c11fe16380958305b2f379ad",
        "data": null,
        "timestamp": "2018-10-11T01:21:45.899730786+08:00",
        "logHash": "0000000000000000000000000000000000000000000000000000000000000000",
        "nonce": "1GO9X2PtbDM=",
        "signature": "rVA04yeWgERnmzVJ0LsLqIEkjn6r2BrePyxOCS2N4l+UKy3mjaIWO5ybk8sc6qiVR91reEwXHwyfeFo+CjNNCg==",
        "height": "1",
        "quota": "0",
        "amount": "1000000000000000000000000000",
        "fee": "0",
        "confirmedTimes": "0",
        "tokenInfo": {
            "tokenName": "Vite Token",
            "tokenSymbol": "VITE",
            "totalSupply": "1000000000000000000000000000",
            "decimals": 18,
            "owner": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
            "pledgeAmount": "0",
            "withdrawHeight": "0"
        }
    }
}
```
:::

## ledger_getBlocksByHash
Return a certain number of consecutive account blocks backward from the specified block by hash

- **Parameters**: 
    - `string` : `address` Account address
    - `string` : `hash`  Start account block hash
    - `int` :   Number of blocks to query

- **Returns**: `Array<AccountBlock>` Account block list

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlocksByHash",
    "params": [
        "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
        2
    ]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": [
        {
            "blockType": 4,
            "hash": "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
            "prevHash": "0000000000000000000000000000000000000000000000000000000000000000",
            "accountAddress": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
            "publicKey": "OvmkehEUDGgcKyqFpM6Yf6sGklibLOIzv34XS9QwF3o=",
            "toAddress": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "fromBlockHash": "5113171e23ac1cdfcb6851f9bea7ad050058acccbe2e6faf8f5a2231f02c5f7c",
            "tokenId": "tti_5649544520544f4b454e6e40",
            "receiveBlockHeight": "",
            "receiveBlockHash": null,
            "snapshotHash": "fc08446111289c671fe1547f634afcde92fab289c11fe16380958305b2f379ad",
            "data": null,
            "timestamp": "2018-10-11T01:21:45.899730786+08:00",
            "stateHash": "53af30da1fc818c9a03ef539aadf7a1e0c90039d5c4eb42143dd9cfc211adbe6",
            "logHash": "0000000000000000000000000000000000000000000000000000000000000000",
            "nonce": "1GO9X2PtbDM=",
            "signature": "rVA04yeWgERnmzVJ0LsLqIEkjn6r2BrePyxOCS2N4l+UKy3mjaIWO5ybk8sc6qiVR91reEwXHwyfeFo+CjNNCg==",
            "height": "1",
            "quota": "0",
            "amount": "1000000000000000000000000000",
            "fee": "0",
            "confirmedTimes": "0",
            "tokenInfo": {
                "tokenName": "Vite Token",
                "tokenSymbol": "VITE",
                "totalSupply": "1000000000000000000000000000",
                "decimals": 18,
                "owner": "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
                "pledgeAmount": "0",
                "withdrawHeight": "0"
            }
        }
    ]
}
```
```json test
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlocksByHash",
    "params": [
        "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
        2
    ]
}
```
:::
## ledger_getBlocksByHashInToken
Return a certain number of consecutive account blocks backward from the specified block by hash. The transactions in the blocks were settled in the specified token.

- **Parameters**: 
    - `string` : `address` Account address
    - `string` : `hash`  Start account block hash
    - `string` : `tokenId` Token id
    - `int` :   Number of blocks to query

- **Returns**: `Array<AccountBlock>` Account block list

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlocksByHashInToken",
    "params": [
        "vite_00000000000000000000000000000000000000056ad6d26692",
        null,
        "tti_5649544520544f4b454e6e40",
        10
    ]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": [
        {
            "blockType": 4,
            "hash": "5a03aff84943cf1f5e3d981ae748816049e290e5d2137acdbfeb6bb63aca11bc",
            "prevHash": "5f1bfd19d52154a266f7046216499dafbd472831b5d1150e5674dd449d9087fe",
            "accountAddress": "vite_00000000000000000000000000000000000000056ad6d26692",
            "publicKey": "ZlFXeR1h9Y2eHlFrk0BzTVv5cIJvDTMASVWUpoKFqYg=",
            "toAddress": "vite_00000000000000000000000000000000000000056ad6d26692",
            "fromBlockHash": "ac15375d01664ee9194d582c1772c57889fb4475f2790de966c605bfbb9a4156",
            "tokenId": "tti_5649544520544f4b454e6e40",
            "snapshotHash": "5e93be7f632617fa5385816ceb23fc0cfe5a33665ced6c372d6c2f92fe2e7e85",
            "data": "S+pUvi6hVg9eNNrGmbewiSMLAUXd9dtJTwxS32hO4csA",
            "logHash": null,
            "nonce": null,
            "signature": "nfrz9nF6a5KhOFWdwnfcy1hqvoAeFkokHyk0XkiLEXiY+t11XnzlFsR04Y1t8ZzVCC1x17JezKU6W+BZ1JGKBA==",
            "fromAddress": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
            "receiveBlockHeight": "",
            "receiveBlockHash": null,
            "height": "5",
            "quota": "0",
            "amount": "0",
            "fee": "1000000000000000000000",
            "difficulty": null,
            "timestamp": 1546935398,
            "confirmedTimes": "4322",
            "tokenInfo": {
                "tokenName": "Vite Token",
                "tokenSymbol": "VITE",
                "totalSupply": "1000000000000000000000000000",
                "decimals": 18,
                "owner": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "pledgeAmount": "0",
                "withdrawHeight": "0",
                "tokenId": "tti_5649544520544f4b454e6e40"
            }
        },
        {
            "blockType": 4,
            "hash": "d9b053f24ae0844d2105ad01d62da90723f8537c62f1953ada10cae6d58d9ac0",
            "prevHash": "f17aae62fb6c15c752f43f7f4e49ebc83a206aa39874b6805bb012e31b3a5de9",
            "accountAddress": "vite_00000000000000000000000000000000000000056ad6d26692",
            "publicKey": "xP0t/cCgrTNjOrkS8HYoFD7RDKPtCPzkdrk12MIjMgM=",
            "toAddress": "vite_00000000000000000000000000000000000000056ad6d26692",
            "fromBlockHash": "2c757064c78cf25bdbd80dfc4af0377c00d155b1d0f71f209bf7a0589670354f",
            "tokenId": "tti_5649544520544f4b454e6e40",
            "snapshotHash": "4f9e834598ebad22047308a5b7489ef27de1120ea80d33f5310801cd1eaa5e4f",
            "data": "0e0HOSvpbeG+SKedm33fgHuNqHlmEFCIQhf2z3O3iQ8A",
            "logHash": null,
            "nonce": null,
            "receiveBlockHeight": "",
            "receiveBlockHash": null,
            "signature": "n4PqczrUj4YWRB1xExYehKrusbSlKS2kIwTXQuodjuAOK4vEXGx+IklZY71yY2TPKE2tbGk3PW1XmTfKUHz8AA==",
            "fromAddress": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
            "height": "3",
            "quota": "0",
            "amount": "0",
            "fee": "1000000000000000000000",
            "difficulty": null,
            "timestamp": 1546935356,
            "confirmedTimes": "4342",
            "tokenInfo": {
                "tokenName": "Vite Token",
                "tokenSymbol": "VITE",
                "totalSupply": "1000000000000000000000000000",
                "decimals": 18,
                "owner": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "pledgeAmount": "0",
                "withdrawHeight": "0",
                "tokenId": "tti_5649544520544f4b454e6e40"
            }
        }
    ]
}
```
```json test
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getBlocksByHash",
    "params": [
        "vite_098dfae02679a4ca05a4c8bf5dd00a8757f0c622bfccce7d68",
        "8f37904d4df342569a2f79d8deb496c03c89eb89353cf027b1d7dc6dafcb351a",
        2
    ]
}
```
:::

## ledger_getSnapshotChainHeight
Return current snapshot chain height

- **Parameters**: `none`

- **Returns**: `string of uint64` The current height of snapshot chain

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "ledger_getSnapshotChainHeight",
	"params": null
}

```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": "1816565"
}
```
```json test
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "ledger_getSnapshotChainHeight",
	"params": null
}

```
:::

## ledger_getSnapshotBlockByHash
Return a snapshot block by hash

- **Parameters**: 
    - `Hash` Snapshot block hash

- **Returns**: 
               
    `Object` : Snapshot block
     -  `producer` : `string` Snapshot block producer
     -  `hash` : `Hash` Snapshot block hash
     -  `prevHash` : `Hash` Previous snapshot block hash
     -  `height` : `uint64` Snapshot block height
     -  `publicKey` : `ed25519.PublicKey` Producer's public key
     -  `signature` : `[]byte` Signature
     -  `timestamp` : `time` Timestamp when the snapshot block was produced
     -  `seed`: `uint64` Random seed that was generated by this producer in last round
     -  `seedHash`: `Hash` Hash of random seed generated in current round
     -  `snapshotContent` : `map[types.Address]HashHeight` Snapshot content


- **Example**:

::: demo

```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getSnapshotBlockByHash",
    "params": ["579db20cb0ef854bba4636d6eaff499ae106ecd918826072a75d47f3e7cbe857"]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "producer": "vite_94badf80abab06dc1cdb4d21038a6799040bb2feb154f730cb",
        "hash": "579db20cb0ef854bba4636d6eaff499ae106ecd918826072a75d47f3e7cbe857",
        "prevHash": "18cf03a6c5d5128bc0a419f23713689cb279165d057759640c700c28c9315470",
        "height": 1807756,
        "publicKey": "zpwPhKs0jClH2JYqn3HieI3SPqm97PMKZlsive8PjBw=",
        "signature": "EzgWq2h2h+qkIHhsKSHK7IMIn3M9bAVR3Sy8ZpaLx2U7BJ6mjVhKIuerEKLcEsY9qbPfc9IYgJ9YYpd1uVK4Dw==",
        "seed": 15994478024988707574,
        "seedHash": "360f20aa86891f67fdab4da09fc4068521c7ffb581f54761f602c2771ecdb097",
        "snapshotContent": {
            "vite_61088b1d4d334271f0ead08a1eec17b08e7ef25141dd427787": {
                "height": 9596,
                "hash": "b8a272bcebb5176fc5b918b6d1e4fc9aca5fd6a0be1fcea99386c6f8ae98a5c1"
            },
            "vite_866d14993fd17f8090d1b0b99e13318c0f99fdd180d3b6cca9": {
                "height": 777,
                "hash": "c78843e347f5927d255f4b57704335dc43222041bf5f27d45980ac83fcf1dbb3"
            }
        },
        "timestamp": 1560422154
    }
}
```
:::

## ledger_getSnapshotBlockByHeight
Return a snapshot block by height

- **Parameters**: 
    - `uint64`  Snapshot block height

- **Returns**: 
               
    `Object` : Snapshot block
     -  `producer` : `string` Snapshot block producer
     -  `hash` : `Hash` Snapshot block hash
     -  `prevHash` : `Hash` Previous snapshot block hash
     -  `height` : `uint64` Snapshot block height
     -  `publicKey` : `ed25519.PublicKey` Producer's public key
     -  `signature` : `[]byte` Signature
     -  `timestamp` : `time` Timestamp when the snapshot block was produced
     -  `seed`: `uint64` Random seed that was generated by this producer in last round
     -  `seedHash`: `Hash` Hash of random seed generated in current round
     -  `snapshotContent` : `map[types.Address]HashHeight` Snapshot content


- **Example**:

::: demo

```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "ledger_getSnapshotBlockByHeight",
    "params": [1815388]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "producer": "vite_d12cfc15515e56d289ff0d17dadc10e1ceeca9129063119a80",
        "hash": "4738036512f2b8098371186174289926a626765bf8b9456cb87d58f13514d1dd",
        "prevHash": "c9e481b85a36d89aa559e2fcc80ea2bfa164f9595c825f943ca70376bd6fc822",
        "height": 1815388,
        "publicKey": "D27o6IBeOJA4X5r7acPv+FVHOWiS9xA2QMlschv8Dvo=",
        "signature": "OKQr77zN85mINoUDXRo8qflIwdNIVx+xzJggpQKVAS9fUA9DmTLfnQ3DHre+88zpxilFNJ/zv+MTFZ+ju60uBA==",
        "seed": 0,
        "seedHash": null,
        "snapshotContent": {
            "vite_2da93df598f39c7afea75b25c1eab09f427ec0dc1d4fce5021": {
                "height": 1078,
                "hash": "c15931c17deefafd594f9eba784e2df53b4d1bd891c767b7122de668c01c14e8"
            },
            "vite_ec432ee636eca9e37479b9bceec950385df82adaeaede35a6c": {
                "height": 105,
                "hash": "7e9fd85e7239cc32be33a02ea4036804e0087eb7d8a43e068063c48f7fb5b726"
            }
        },
        "timestamp": 1560429988
    }
}
```
:::

## ledger_getVmLogList
Return contract execution logs by response transaction hash

- **Parameters**:
   * `string` : `Hash`  Contract response transaction hash

- **Returns**: `VmLogList<array<VmLog>>` VM log list

  `Object` : `VmLog`
    * Topics : `[]types.Hash` Topic list
	  * Data : `[]byte` Log data

- **Example**:

::: demo

```json tab:Request
{
    "jsonrpc":"2.0",
    "id":1,
    "method":"ledger_getVmLogList",
    "params": ["c5073890ca14637efbdceb078eb90cb853fc1a69a244273185848cf6c7e45dd5"]
}

```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": [
        {
            "topics": [
                "96a65b1cd08da045d0318cafda7b8c8436092851d5a4b7e75054c005a296e3fb",
                "000000000000000000000029480d1db34493941a5ab00026e0e25085b8fe5a00"
            ],
            "data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAPYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAK1468WsYgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAF"
        }
    ]
}
```
:::
