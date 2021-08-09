---
sidebarDepth: 4
---

# Wallet

:::tip Maintainer
[TiantaoZhu](https://github.com/TiantaoZhu)
:::

## wallet_listEntropyFilesInStandardDir
Return all `EntropyStore` in the standard directory

- **Parameters**: `none`

- **Returns**: 
	- `Array[string]` An list of absolute paths of `EntropyStore`

- **Example**:


::: demo


```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_listEntropyFilesInStandardDir",
	"params": []
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": [
        "/Users/xxx/Library/GVite/testdata/wallet/vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
        "/Users/xxx/Library/GVite/testdata/wallet/vite_5b013ec4f3c235da12e47b525713e2f5edd0b04df965fafc22",
        "/Users/xxx/Library/GVite/testdata/wallet/vite_67de981eff372d4a757541b05f0e8a269eee11c2f6c9fbdae6",
        "/Users/xxx/Library/GVite/testdata/wallet/vite_f24bb4eceadc65020de5de6a4aeb22c52edd6cb72ee2279a97"
    ]
}

```
:::

## wallet_listAllEntropyFiles 
Return all `EntropyStore` managed by the wallet

- **Parameters**: `none`

- **Returns**: 
	- `Array[string]` An list of absolute paths of `EntropyStore`


- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_listAllEntropyFiles",
    "params": []
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": [
        "/Users/xxx/Library/GVite/testdata/wallet/vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
        "/Users/xxx/Library/GVite/testdata/wallet/vite_5b013ec4f3c235da12e47b525713e2f5edd0b04df965fafc22",
        "/Users/xxx/Library/GVite/testdata/wallet/vite_67de981eff372d4a757541b05f0e8a269eee11c2f6c9fbdae6"
    ]
}
```
:::

## wallet_extractMnemonic
Return mnemonics of specified `EntropyStore`

- **Parameters**: 
	- `string` : The absolute file path of the `EntropyStore`, or `EntropyStore` name if the file is in standard directory. Address 0 is used as standard `EntropyStore` name.
	- `string` : Wallet password

- **Returns**: 
	- `string`: Mnemonic phrase

- **Example**: 

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_extractMnemonic",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
		"123456"]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": "goddess crush pattern cluster level combine survey give seminar uniform invite beach"
}
```
:::

## wallet_unlock
Unlock the specified `EntropyStore`

- **Parameters**: 
	- `string` : The absolute file path of the `EntropyStore`, or `EntropyStore` name if the file is in standard directory. Address 0 is used as standard `EntropyStore` name.
	- `string` : Wallet password

- **Returns**: 
	- `null`

- **Example**: 

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_unlock",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
		"123456"]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": null
}
```
:::

## wallet_lock
Lock the specified `EntropyStore`

- **Parameters**: 
	- `string` : The absolute file path of the `EntropyStore`, or `EntropyStore` name if the file is in standard directory. Address 0 is used as standard `EntropyStore` name.


- **Returns**:
	- `null`

- **Example**:

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_lock",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e"]
}
```

```json tab:Response Success
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": null
}
```

:::

## wallet_listEntropyStoreAddresses
Return a specified range of addresses in the `EntropyStore`

- **Parameters**: 
	- `string` : The absolute file path of the `EntropyStore`, or `EntropyStore` name if the file is in standard directory. Address 0 is used as standard `EntropyStore` name.
	- `uint32`: Start index, included
	- `uint32`: End index, excluded

- **Returns**: 
	- `Array[string]`: An list of account addresses

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_listEntropyStoreAddresses",
	"params": [
		"/Users/xxx/Library/GVite/testdata/wallet/vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
		0,
		10
	]
}
```
```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": [
        "vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
        "vite_659fbce2a908bdab3b7c46348f249c90c812e36b9ceac67aa0",
        "vite_cc373442a471a8dd4b2240d5a74f8e4037177a8795d30bdfd7",
        "vite_1beb02f13af1b16a317c927d470ca3118ba738e22da3f8bf6e",
        "vite_d1c10321319de24bcbd865e7f4127f5873bf9c251f0a4abb00",
        "vite_acf35393ba47b8216ebbc5252e8884d518971a57c11b5866e1",
        "vite_87df61f0feddb6121fecf7d5ba8d7e56443d53ead06da90a06",
        "vite_8de52abce25c65116b08d84966d67e3dd7860848b52f388d23",
        "vite_ed346025dc7196ed000caa429306bbed1bda42010597d63676",
        "vite_783baa0caccff4d365f09873660208ee727bfc5d9710b267e5"]
}
```
:::

## wallet_newMnemonicAndEntropyStore
Create new mnemonics and corresponding `EntropyStore`

- **Parameters**: 
	- `string`: Wallet password

- **Returns**: 
	- `Object`:
		- `mnemonic : string` : The newly created mnemonics
		- `primaryAddr : string` : Primary account address
		- `filename : string`: Absolute `EntropyStore` file path

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_newMnemonicAndEntropyStore",
    "params": [
        "123456"
    ]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "mnemonic": "pear lonely piece base local lift material damp animal siege error throw ride flag version dumb parent clever upper toe lumber great wild vivid",
        "primaryAddr": "vite_f646dc1f32b0ea88289bbfe4e4138d26edc9f9eac33a9e5292",
        "filename": "/Users/xxx/Library/GVite/testdata/wallet/vite_f646dc1f32b0ea88289bbfe4e4138d26edc9f9eac33a9e5292"
    }
}
```
:::

## wallet_deriveByIndex
Generate sub address by index

- **Parameters**: 
	- `string` : The primary account address or the absolute file path of the `EntropyStore`
	- `uint32`: The index of sub address

- **Returns**: 
	- `Object`:
		-  `bip44Path : string` : The BIP44 path of the address
		-  `address : string` : Sub address
		-  `privateKey : []byte`: Private key in base64 encoding

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_deriveForIndexPath",
    "params": [
    	"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
        0
    ]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "bip44Path": "m/44'/666666'/1'",
        "address": "vite_8431a5acb599da19529c3e8bd099087a9d550fb29039dada28",
        "privateKey": "SKxAWibv4u85xMdRByCaveOwjw0bhempG9/zi59TjJUESNFMNvoE+wP/X/Zz+Tc3ObdZVO53UQT5BS8xATefbg=="
    }
}
```

:::

## wallet_deriveByFullPath
Generate sub address by BIP44 path. This method supports deriving sub address more flexibly.

- **Parameters**: 
	- `string` : The primary account address or the absolute file path of the `EntropyStore`
	- `string`: The BIP44 path of the address

- **Returns**: 
	- `Object`:
		-  `bip44Path : string` : The BIP44 path of the address
		-  `address : string` : Sub address
		-  `privateKey : []byte`: Private key in base64 encoding

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_deriveByFullPath",
    "params": [
    	"vite_b1c00ae7dfd5b935550a6e2507da38886abad2351ae78d4d9a",
        "m/44'/666666'/2'/4'"
    ]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 2,
    "result": {
        "bip44Path": "m/44'/666666'/2'/4'",
        "address": "vite_a5efba49303b46c42c7e89b6cf5facd897d3a444fdb37af64e",
        "privateKey": "HSe4vB20dKTHYz+xzlAJ+wDhQQTJnJfemLTjbkPBb6ql/LS+lob/77NOdRfky3VWjai4g81mGR8L+goQDgEKoA=="
    }
}
```

:::

## wallet_recoverEntropyStoreFromMnemonic
Recover `EntropyStore` from mnemonics

- **Parameters**: 
	- `string` : Mnemonics
	- `string`: New wallet password

- **Returns**: 
- `Object`:
	- `mnemonic : string` : Mnemonics
	- `primaryAddr : string` : Primary account address
	- `filename : string`: Absolute `EntropyStore` file path 

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_recoverEntropyStoreFromMnemonic",
	"params": [
	"utility client point estate auction region jump hat sick blast tomorrow pottery detect mixture clog able person matrix blast volume decide april congress resource",
		"123456"
	]
}

```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": {
        "mnemonic": "utility client point estate auction region jump hat sick blast tomorrow pottery detect mixture clog able person matrix blast volume decide april congress resource",
        "primaryAddr": "vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0",
        "filename": "/Users/xxx/Library/GVite/testdata/wallet/vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0"
    }
}

```

:::

## wallet_globalCheckAddrUnlocked
Check if the specified address is unlocked

- **Parameters**: `string` : `address` Account address

- **Returns**: `bool` Whether the address is unlocked. `true` means unlocked

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_globalCheckAddrUnlocked",
	"params": [
	"vite_3fd41bb6ba4f15d5e74214a16153ff2f5abce67f73dc0dc07b"
	]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": false
}
```


:::

## wallet_isAddrUnlocked
Check if the specified address in the `EntropyStore` is unlocked. The specific `EntropyStore` must be unlocked in advance before calling this method.

- **Parameters**: 
	- `string` : The primary account address or the absolute file path of the `EntropyStore`
	- `string`:`address`： Account address

- **Returns**: `bool` Whether the address is unlocked. `true` means unlocked

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_isAddrUnlocked",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
		"vite_3fd41bb6ba4f15d5e74214a16153ff2f5abce67f73dc0dc07b"
	]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": true
}
```


:::

## wallet_isUnlocked
Check if the specified `EntropyStore` is unlocked.

- **Parameters**:  `string`: The primary account address or the absolute file path of the `EntropyStore`

- **Returns**: `bool` Whether the address is unlocked. `true` means unlocked

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_isUnlocked",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e"
	]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": true
}
```

:::

## wallet_findAddr
Return the index of specified address in the `EntropyStore`. The specific `EntropyStore` must be unlocked in advance before calling this method.

- **Parameters**:  
	- `string`: The primary account address or the absolute file path of the `EntropyStore`
	- `string`:`address`： Account address

- **Returns**:
	- `Object` 
		- `entropyStoreFile : string` : Absolute `EntropyStore` file path
		- `index : uint32 `: Index of the address in the `EntropyStore`
- **Example**: 

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_findAddr",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
		"vite_3fd41bb6ba4f15d5e74214a16153ff2f5abce67f73dc0dc07b"
	]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": {
        "entropyStoreFile": "/Users/xxx/Library/GVite/testdata/wallet/vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
        "index": 84
    }
}
```


:::

## wallet_globalFindAddr
Return the index of specified address

- **Parameters**:  
	 * `string`:`address`： Account address

- **Returns**:
	- `Object` 
		- `entropyStoreFile : string` : Absolute `EntropyStore` file path
		- `index : uint32 `: Index of the address in the `EntropyStore`
- **Example**: 

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_globalFindAddr",
	"params": [
	"vite_3fd41bb6ba4f15d5e74214a16153ff2f5abce67f73dc0dc07b"
	]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": {
        "entropyStoreFile": "/Users/xxx/Library/GVite/testdata/wallet/vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
        "index": 84
    }
}
```

:::

## wallet_createTxWithPassphrase
Send a transaction

- **Parameters**:  
	-  `Object`:
		- `entropystoreFile : string` :  The primary account address or the absolute file path of the `EntropyStore`, optional
		- `selfAddr : string address` : The account address from which the transaction is sent, required
		- `toAddr : string address` : The account address to which the transaction is sent, required
		- `tokenTypeId : string tokentypeid` : Token ID, required
		- `passphrase : string` : Wallet password, required
		- `amount : string bigint` : Transfer amount, required
		- `data : string base64` : Additional data carried by the transaction, optional
		- `difficulty : string bigint` : PoW difficulty, optional

- **Returns**: `none`


:::

## wallet_addEntropyStore
Add a new `EntropyStore` from non-standard directory.

- **Parameters**:  
	- `string`: The primary account address or the absolute file path of the `EntropyStore`

- **Returns**: `none`
