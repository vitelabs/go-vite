---
order: 2
---

# Wallet Management

:::tip
This document mainly introduces how to config wallet on full node. Before starting this tutorial, please see [Node - Installation][install] to install gvite first.
:::

:::danger Strongly NOT recommended：
* Using ***same address*** for both SBP address and SBP registration address
* Using ***same mnemonics*** to generate both SBP address and SBP registration address
:::

## Create Wallet

### Start Full Node

Follow [Full Node Installation][install] to start a full node.

### Connect Full Node in Command Line Console

Navigate to [Full Node Installation Directory][pwd] and execute the following command:

  ```bash
  ./gvite attach ~/.gvite/maindata/gvite.ipc
  ```

  Below output indicates the full node has been connected successfully:
  ```
  Welcome to the Gvite JavaScript console!
  ->
  ```
### Create a New Wallet
  
Execute the following command
```javascript
vite.wallet_createEntropyFile("Your_Password")
```
This will give you below result
```json
{
    "jsonrpc": "2.0", 
    "id": 1, 
    "result": {
        "mnemonic": "ancient rat fish intact viable flower now rebuild monkey add moral injury banana crash rabbit awful boat broom sphere welcome action exhibit job flavor", 
        "primaryAddr": "vite_f1c2d944b1e5b8cbfcd5f90f94a0e877beafeced1f331d9acf", 
        "filename": "~/.gvite/maindata/wallet/vite_f1c2d944b1e5b8cbfcd5f90f94a0e877beafeced1f331d9acf"
    }
}
```

* `mnemonic`: Mnemonic phrase. Please keep it safe
* `primaryAddr`: Vite address at index 0 corresponding to the mnemonic
* `filename`: The location of the keyStore file

Run `exit` to abort

## Recover Wallet from Mnemonic

Execute the following command

```javascript
vite.wallet_recoverEntropyFile("Your_Mnemonic", "Your_Password")
```

For example：

:::demo
```javascript tab: Input
vite.wallet_recoverEntropyFile("utility client point estate auction region jump hat sick blast tomorrow pottery detect mixture clog able person matrix blast volume decide april congress resource","123456")
```
```json tab: Ouput
{
    "jsonrpc": "2.0",
    "id": 4,
    "result": {
        "mnemonic": "utility client point estate auction region jump hat sick blast tomorrow pottery detect mixture clog able person matrix blast volume decide april congress resource",
        "primaryAddr": "vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0",
        "filename": "~/.gvite/maindata/wallet/vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0"
    }
}
```
:::

Now the keystore file "vite_981bca7a348de85bd431b842d4b6c17044335f71e5f3da59c0" has been regenerated under "~/.gvite/maindata/wallet/".


[install]: <./install.md>
[pwd]: <./install.md#Description-of-installation-directory>


