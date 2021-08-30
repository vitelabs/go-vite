---
order: 4
---

# Contract Debugging

## Debugging Environment

In order to debug smart contract, you need setup a local node and do debugging through your node's RPC interface.

Vite provides 2 debugging environments: development and testing. 

**Development Environment**
* Account balance is NOT verified before sending transaction. 
* Each transaction can utilize maximum 47.62`UT` quota with no staking. 

**Testing Environment**
* Quota and balance are verified normally. Pay attention to your balance and quota during debugging.

::: tip Tips
It is possible to debug your smart contract in Microsoft Visual Studio Code (aka VS Code) in development environment.
:::
## Debugging in VS Code

Soliditypp extension for VS Code supports debugging `solidity++` smart contract in local development environment.

Following features are supported:

* `solidity++` syntax highlighting
* `solidity++` code auto completion
* Auto compilation when saving `.solpp` file
* Compilation error highlighting
* Detailed error message displaying when mouse over 
* One-click smart contract deployment and debugging
* Support for multiple smart contract interaction
* Deployment/debugging result displaying
* Support for offchain queries
* Example `solidity++` code

### Install the Extension

Search for "Soliditypp" in VS Code and install.

![](./vscode-extension.png)

### Create HelloWorld.solpp

Open VS Code workbench, press `⇧⌘P`(or `F1`) in Mac(or `Ctrl+Shift+P` in Windows) to bring Command Palette, and then execute command `soliditypp: Generage HelloWorld.solpp`. This will generate `HelloWorld.solpp` under current folder as an example.

### Write Smart Contract

Write your contract in a new `.solpp` file and then save it by pressing `⌘S` in Mac(or `Ctrl+S` in Windows). Your source file will be automatically compiled each time it is saved. 
Lines with compilation error in the code will be marked with red underscore. Detailed error message will be displayed when you mouse over.

VS Code will recognize `.solpp` file as `solidity++` source by default. 

### Deploy/Debug Smart Contract

In Debug panel, start debugging and choose `Soliditypp` environment. This will launch a local `gvite` node and all following deployment and debugging steps will take place on this node.
Please note that all data will be cleared from the node after debugging is complete.

![](./vscode-debug.png)

* Section 1: soliditypp source code panel
* Section 2: current address used for contract deployment and debugging. If you want to use a different address, click `+` to generate a new address and then choose the address in the droplist on the right side
* Section 3: deploy panel. Field `amount` can be used to send VITE (1 VITE = 1e18 attov) upon deploying the contract. Click `deploy` button to deploy the contract in local development environment.
* Section 4: contracts deployed. Multiple results will be displayed if more than one contracts are deployed. Parameter `amount` is used to send VITE to the contract when calling a function. For example, clicking `call "SayHello"` will call method `SayHello` of contract `HelloWorld`
* Section 5: deployment/debugging result. `Send`/`Receive` shows the information of request/response transaction. New request transaction is displayed in `Receive` if it calls other method. Please note that transactions in Vite are asynchronous and user may need wait until the response transaction is generated after sending the request. More information about `Send` and `Receive` please see [AccountBlock](../../api/rpc/common_models_v2.html#accountblock)。

## Debugging in Command Line
 
### Development Environment

Download [Gvite Debugging Package](https://github.com/vitelabs/gvite-contracts/releases) in development environment and install

#### Install

```bash
## Unzip
tar -xzvf contractdev-v1.3.1-darwin.tar.gz
```
```bash
## Enter folder extracted
cd contract_dev
```
```bash
## Start gvite debugging environment
sh run.sh
```
Check gvite.log to see if gvite debugging environment has been started successfully
```bash
cat gvite.log
```
Following messages show a successful startup
```bash
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.DataDir:/home/ubuntu/contract_dev/ledger/devdata module=gvite/node_manager
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.KeyStoreDir:/home/ubuntu/contract_dev/ledger/devdata/wallet module=gvite/node_manager
Prepare the Node success!!!
Start the Node success!!!
```

#### Create Contract

First write contract in Solidity++ and save into a ".solpp" file under the same directory with start script

Compile contract
```bash
## Compile contract using solppc, and genereate binary code and ABI
./solppc --bin --abi HelloWorld.solpp
```

Deploy contract in local debugging environment
```bash
## Create contract from HelloWorld.solpp with test account(created during startup)
sh create_contract.sh HelloWorld.solpp
```

```bash
## Below script shows how to create contract with passed-in parameters
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "vmdebug_createContract",
    "params": [
        {
          "fileName":"'`pwd`/AsyncCall.solpp'",
          "params":{
            "A":{
              "amount":"0",
              "params":[]
            },
            "B":{
              "amount":"0",
              "params":["1"]
            }
          }
        }
    ]
}'
```
The parameters are explained below
```json
{
  // Create contract from AsyncCall.solpp under current directory
  "fileName":"'`pwd`/AsyncCall.solpp'",
  "params":{
    // Passed-in parameters for contracts. This example shows two contracts - A and B
    "A":{
      // Transfer amount when creating contract
      "amount":"0",
      "params":[]
    },
    "B":{
      "amount":"0",
      // Passed-in parameters for contructor. In this example, it passes in a uint value for contract B's contructor
      "params":["1"]
    }
  }
}
```

Following return message shows the contract has been created successfully
```json
{
  "jsonrpc": "2.0", 
  "id": 0, 
  "result": [
    {
      "accountAddr": "vite_21483c46a64799c7db0cba88cf7b007a2d1a37e863f7be94b7", 
      "accountPrivateKey": "b18bcd61db171fb0c97712c24dbfc4fe7d279a6e9f40be2a81f5e279206887237ee77ed82025fbe821a969cc8321c139ed69dde16bed9c5dfabbc6343868bb68", 
      "contractAddr": "vite_d624b0bead067237700a86314287849163e4a0fb6139fdff42", 
      "sendBlockHash": "265930575e035976f0e89b7b4ad00c5e91fefc9230647b47dadd7c7274797d3b", 
      "methodList": [
        {
          "contractAddr": "vite_d624b0bead067237700a86314287849163e4a0fb6139fdff42", 
          "accountAddr": "vite_21483c46a64799c7db0cba88cf7b007a2d1a37e863f7be94b7", 
          "amount": "0", 
          "methodName": "SayHello", 
          "params": [
            "address"
          ]
        }
      ]
    }
  ]
}
```
Explained below
```json
{
  "jsonrpc":"2.0",
  "id":0,
  "result":[ 
    // List of contracts created. If multiple contracts exist in the .solpp file, they will all be listed here
    {
      // Test account address
      "accountAddr":"vite_21483c46a64799c7db0cba88cf7b007a2d1a37e863f7be94b7",  
      // Test account private key
      "accountPrivateKey":"b18bcd61db171fb0c97712c24dbfc4fe7d279a6e9f40be2a81f5e279206887237ee77ed82025fbe821a969cc8321c139ed69dde16bed9c5dfabbc6343868bb68",
      // Contract account address
      "contractAddr":"vite_d624b0bead067237700a86314287849163e4a0fb6139fdff42",
      // Hash of request transaction that created the contract
      "sendBlockHash":"265930575e035976f0e89b7b4ad00c5e91fefc9230647b47dadd7c7274797d3b",
      // Method list of the contract
      "methodList":[
        {
          // Contract account address
          "contractAddr":"vite_d624b0bead067237700a86314287849163e4a0fb6139fdff42",
          // Test account address 
          "accountAddr":"vite_21483c46a64799c7db0cba88cf7b007a2d1a37e863f7be94b7",
          // Transfer amount when the method is called
          "amount":"0",
          // Method name
          "methodName":"SayHello",
          // Parameter list with pseudo value. Multiple parameters are displayed if the method requires more than one pararmeter.
          "params":["address"]
        }
      ]
    }
  ]
}
```

#### Call Contract

```bash
## Call a contract method. Remember to specify the right parameters that match your testing environment
curl -X POST \
  http://127.0.0.1:48132/ \
  -H 'content-type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": 0,
    "method": "vmdebug_callContract",
    "params": [
        {
          "contractAddr":"vite_d624b0bead067237700a86314287849163e4a0fb6139fdff42",
          "accountAddr":"vite_21483c46a64799c7db0cba88cf7b007a2d1a37e863f7be94b7",
          "amount":"0",
          "methodName":"SayHello",
          "params":["vite_21483c46a64799c7db0cba88cf7b007a2d1a37e863f7be94b7"]
        }
    ]
}'
```
Return message is explained below
```json
{
    "jsonrpc": "2.0", 
    "id": 0, 
    "result": {
        // Contract account address
        "contractAddr": "vite_0a49d38e769162f05d0df645b890ac450f80cb49d52e8765ab", 
        // Test account address that called the contract. This account was created automatically when your created the contract
        "accountAddr": "vite_a4aa32b30a4564d3c5ffac1f7416d09cd4dd36bbf365df5be5", 
        // Test account private key
        "accountPrivateKey": "2bef2ba485ed3e4de8b93bd0fb8746db47a91f4bdde0c007127b5bc6548ff49642d4138c403cc26e20299a2f145687bf562f6ba1e7d0d45a75d7c7f58de42b25", 
        // Hash of request transaction that called the contract
        "sendBlockHash": "eea88399209cd5abdedec1128b8bdfd1a28e2d6ac6ade6d5cee72e997a800893"
    }
}
```
Script call.sh shows an example how to call a contract

#### Verify Execution Result

Since the response transactions of creating or calling contract are processed asynchronously, we have to wait for a while(1~6s in local environment) after sending out the request.
Executing steps and results are recorded in two separate logs.

Logs are under `/ledger/devdata/vmlog` in the folder unzipped

 * `vm.log`：Contains logs of contract execution entrance, result and events
 * `interpreter.log`：Contains detailed info of contract execution steps, including VM instructions, status of stack, memory, storage and etc.

It's also feasible to query contract account chain to check contract execution result
```bash
sh query_block.sh vite_0a49d38e769162f05d0df645b890ac450f80cb49d52e8765ab
```

### Testing Environment

Download [Gvite Debugging Package](https://github.com/vitelabs/gvite-contracts/releases) in testing environment and install

#### Install

Installation steps in testing environment are similar as in development environment. See [Install](#install).

#### Create Test Account

Remember to create new test account and stake for the account each time gvite has been rebooted. The test account will be used to create or call contract.

Following steps are finished during account creation
 * Creating new test account address
 * A amount of tokens are sent out from genesis account to test account
 * Tokens are received by test account 
 * Genesis account stakes for test account to obtain quota
 * Test account waits to receive quota

```bash
sh create_account.sh
```

#### Create Contract

Compared with in development environment, additional test account address should be specified when running create_contract.sh
```bash
sh create_contract.sh HelloWorld.solpp vite_d5fe580d0ba8fa4002e2a33af2cd10645a58ad1552d4562c0a
```
Run pledge_for_contract.sh to stake for contract account created
```bash
sh pledge_for_contract.sh vite_8739653f7fee7e39c3fbeee14e8c17fe4f7ff20e8607fb05ab
```

#### Call Contract

See [Call Contract](#call-contract) in development environment

