---
sidebarDepth: 4
---

# Wallet
:::tip Maintainer
[viteshan](https://github.com/viteshan)
:::

## wallet_getEntropyFilesInStandardDir
Return all entropy files in standard directory

- **Parameters**: `none`

- **Returns**:
	- `Array[string]` Absolute paths of entropy files

- **Example**:


::: demo


```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_getEntropyFilesInStandardDir",
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

## wallet_getAllEntropyFiles
Return all entropy files maintained in wallet

- **Parameters**: `none`

- **Returns**:
	- `Array[string]` Absolute paths of entropy files


- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_getAllEntropyFiles",
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

## wallet_exportMnemonics
Return mnemonic phrase of the given entropy file

- **Parameters**:
	- `string` : Absolute path of entropy file, or file name (address at index 0) if it is under standard directory. 
	- `string` : Passphrase

- **Returns**:
	- `string`: Mnemonics

- **Example**:

::: demo
```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_exportMnemonics",
	"params": [
		"vite_15391ac8b09d4e8ad78bfe5f9f9ab9682fe689572f6f53655e",
		"123456"
		]
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
Unlock entropy file

- **Parameters**:
	- `string` : Absolute path of entropy file, or file name (address at index 0) if it is under standard directory. 
	- `string` : Passphrase

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
Lock entropy file

- **Parameters**:
	- `string` : Absolute path of entropy file, or file name (address at index 0) if it is under standard directory. 

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


## wallet_deriveAddressesByIndexRange
Return a list of addresses in the given entropy file

- **Parameters**:
	- `string` : Absolute path of entropy file, or file name (address at index 0) if it is under standard directory. 
	- `uint32`: Start index, included
	- `uint32`: End index, excluded

- **Returns**:
	- `Array[string]`: Addresses

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_deriveAddressesByIndexRange",
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


## wallet_createEntropyFile
Create new entropy file

- **Parameters**:
	- `string`: Passphrase

- **Returns**:
	- `Object`:
		- `mnemonics : string` : Mnemonic phrase
		- `firstAddress : string` : Address at index 0
		- `filePath : string`: Absolute path of entropy file

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_createEntropyFile",
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
        "mnemonics": "pear lonely piece base local lift material damp animal siege error throw ride flag version dumb parent clever upper toe lumber great wild vivid",
        "firstAddress": "vite_f646dc1f32b0ea88289bbfe4e4138d26edc9f9eac33a9e5292",
        "filePath": "/Users/xxx/Library/GVite/testdata/wallet/vite_f646dc1f32b0ea88289bbfe4e4138d26edc9f9eac33a9e5292"
    }
}
```
:::

## wallet_deriveAddressByIndex
Derive address by index

- **Parameters**:
	- `string` : Absolute path of entropy file, or file name (address at index 0) if it is under standard directory. 
	- `uint32`: Index

- **Returns**:
	- `Object`:
		-  `bip44Path : string` : BIP44 path of the address
		-  `address : string` : Address
		-  `privateKey : []byte`: Private key in base64 format

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_deriveAddressByIndex",
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

## wallet_deriveAddressByPath
Derive address by bip44 path

- **Parameters**:
	- `string` : Absolute path of entropy file, or file name (address at index 0) if it is under standard directory
	- `string`: BIP44 path

- **Returns**:
	- `Object`:
		-  `bip44Path : string` : BIP44 path of the address
		-  `address : string` : Address
		-  `privateKey : []byte`: Private key in base64 format

- **Example**:

::: demo
```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "wallet_deriveAddressByPath",
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

## wallet_recoverEntropyFile
Recover entropy file from mnemonic phrase

- **Parameters**:
	- `string` : Mnemonic phrase
	- `string`: New passphrase

- **Returns**:
- `Object`:
	- `mnemonics : string` : Mnemonic phrase
	- `firstAddr : string` : Address at index 0
	- `filePath : string`: Absolute path of entropy file  

- **Example**:

::: demo

```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 4,
	"method": "wallet_recoverEntropyFile",
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
        "mnemonics": "utility client point estate auction region jump hat sick blast tomorrow pottery detect mixture clog able person matrix blast volume decide april congress resource",
        "firstAddr": "vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0",
        "filePath": "/Users/xxx/Library/GVite/testdata/wallet/vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0"
    }
}

```

:::
## wallet_isUnlocked
Verify if the entropy file is unlocked

- **Parameters**:  `string`: Absolute path of entropy file, or file name (address at index 0) if it is under standard directory

- **Returns**: `bool` If `true`, the entropy file is unlocked

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
Return the index of specified address in entropy file. The entropy file must be unlocked when calling this method

- **Parameters**:  
	- `string`: Absolute path of entropy file, or file name (address at index 0) if it is under standard directory
	- `string`:`address`： Address

- **Returns**:
	- `Object`
		- `entropyStoreFile : string` : Absolute path of entropy file
		- `index : uint32 `: Index of address
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
	 * `string`:`address`： Address

- **Returns**:
	- `Object`
		- `entropyStoreFile : string` : Absolute path of entropy file
		- `index : uint32 `: Index of address
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
Send transaction

- **Parameters**:  
	-  `Object`:
		- `entropystoreFile : string` :  Absolute path of entropy file, or file name (address at index 0) if it is under standard directory, optional
		- `selfAddr : string address` : Address of current account, required
		- `toAddr : string address` : Address of transaction's recipient, required
		- `tokenTypeId : string tokentypeid` : Token id, required
		- `passphrase : string` : Password of entropy file, required
		- `amount : string bigint` : Transfer amount, required
		- `data : string base64` : Additional data carried by the transaction, optional
		- `difficulty : string bigint` : PoW difficulty, optional

- **Returns**: `none`


:::

## wallet_addEntropyStore
Add a new entropy file. This method can be used to add an entropy file in non-standard directory.

- **Parameters**:  
	- `string`: Absolute path of entropy file, or file name (address at index 0) if it is under standard directory, optional

- **Returns**: `none`
