---
order: 1
---

# Installation

## What is gvite node?

In Vite network, nodes are categorized into full nodes and supernodes. Supernode is special full nodes for producing snapshot blocks. This document mainly introduces full node.

Full node is responsible for maintaining a complete copy of ledger, sending or receiving transactions, and verifying all transactions in the network.
Full node can also participate in SBP election and voting.
Full node exposes HTTP/WEBSOCKET APIs externally and has a command line console at local. 

## Before the start

Gvite supports installation from binary or source code

| OS | ubuntu  |  mac |   windows |
| ------------- | ------------------------------ |------|-------|
| {{$page.version}}  | Yes  |Yes |Yes |


## Install from binary

Download latest gvite installation package at [gvite Releases](https://github.com/vitelabs/go-vite/releases) in command line then install

### Installation example on ubuntu
```bash replace version
## Download
curl -L -O  https://github.com/vitelabs/go-vite/releases/download/${version}/gvite-${version}-linux.tar.gz
```
```bash replace version
## Unpack package
tar -xzvf gvite-${version}-linux.tar.gz
```
```bash replace version
## Enter the folder extracted. It should have three files: gvite, bootstrap and node_config.json
cd gvite-${version}-linux
```
```
## Boot up gvite node
./bootstrap
```
Check the content of gvite.log in the same folder to determine whether the program is up and running.
```bash
tail -100f gvite.log
```
The following messages indicate boot is successful.
```bash
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.DataDir:/home/ubuntu/.gvite/maindata module=gvite/node_manager
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.KeyStoreDir:/home/ubuntu/.gvite/maindata/wallet module=gvite/node_manager
Prepare the Node success!!!
Start the Node success!!!
```

### Installation example on mac 

```bash replace version
## Download
curl -L -O https://github.com/vitelabs/go-vite/releases/download/${version}/gvite-${version}-darwin.tar.gz
## Unpack package
tar -xzvf gvite-${version}-darwin.tar.gz
## Enter the folder extracted. It should have three files: gvite, bootstrap and node_config.json
cd gvite-${version}-darwin
## Boot up gvite node
./bootstrap
```

Check the content of gvite.log in the same folder to determine whether the program is up and running.

```bash
cat gvite.log
```

The following messages indicate boot is successful.

```bash
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.DataDir:~/Library/GVite/maindata module=gvite/node_manager
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.KeyStoreDir:~/Library/GVite/maindata/wallet module=gvite/node_manager
Prepare the Node success!!!
Start the Node success!!!
```

### Installation example on windows 
Open up your preferred browser and paste in the following link:

```bash replace version
https://github.com/vitelabs/go-vite/releases/download/${version}/gvite-${version}-windows.tar.gz
```
and save file to preferred directory. Upon download completion, open up explorer, navigate to the directory where file is downloaded to, right click file and select extract file option.

Extracted destination should contain three files:

 `gvite-windows-386.exe (32bit executable)`
 `gvite-windows-amd64.exe (64bit executable)`
 `node_config.json (node config file).`

The folder contains the command .exe files and can be used without installing.

Configure `node_config.json` prior to launching executable (use the 32bit executable if you have a 32bit CPU or 64bit executable if you have a 64bit CPU).

To launch node, simply open up command prompt (by pressing Win + R, then, type cmd and press Enter or click/tap OK.)

Then in command prompt:

```bash replace version
C:\Users\user>d:

D:\>cd gvite-${version}-windows

D:\gvite-${version}-windows>gvite-windows-amd64.exe (or your preferred executable)

```
The following messages indicate boot is successful.

```bash
INFO[11-21|09:28:42] NodeServer.DataDir:C:\Users\user\AppData\Roaming\GVite\maindata module=gvite/node_manager
INFO[11-21|09:28:42] NodeServer.KeyStoreDir:C:\Users\user\AppData\Roaming\GVite\maindata\wallet module=gvite/node_manager
Prepare the Node success!!!
Start the Node success!!!
```
### Description of installation directory

**Installation Directory**：Refers to the folder where gvite boot script and configuration file are located. For example, `~/gvite-${version}-${os}` is an installation directory.

* `gvite`： Gvite executable file
* `bootstrap`： Boot script
* `node_config.json`： Configuration file. See [Configuration Description](./node_config.md)

### Ports

The default ports are 8483/8484. If you choose to go with default ports, please ensure that they are not occupied by other programs or blocked by firewall.

```bash
 netstat -nlp|grep 8483 
```

Check if the default ports are occupied. Gvite will display the following messages if it boots up successfully.
 
```bash
netstat -nlp|grep 8483
(Not all processes could be identified, non-owned process info
 will not be shown, you would have to be root to see it all.)
tcp6       0      0 :::8483                 :::*                    LISTEN      22821/gvite     
udp6       0      0 :::8483                 :::*                                22821/gvite
```

### Description of working directory

```bash
cd ~/.gvite/maindata
```
Gvite working directory, containing sub-directories/files such as "ledger", "ledger_files", "LOCK", "p2p", "rpclog", "runlog" and "wallet".

* `ledger`： Ledger directory
* `rpclog`： RPC log directory
* `runlog`： Run-time log directory
* `wallet`： Wallet keystore directory for storing keystore files that secure private keys. Do remember **KEEP YOUR PRIVATE KEY SAFE**.

## Install from source
### Golang environment check

```
go env
```

:::warning
Go 1.11.1 or above version is required. See [Go Installation Guide](https://golang.org/doc/install).
:::

### Compile source code

  Pull gvite source code
  ```
    go get github.com/vitelabs/go-vite
  ```
  and will be downloaded at:
  ```
  $GOPATH/src/github.com/vitelabs/go-vite/
  ```
  The system default `GOPATH` is ```~/go```
  
  Go to the source code directory and run 
  ```
  make gvite
  ```
  Executable file is generated at: 
  
  ```
  $GOPATH/src/github.com/vitelabs/go-vite/build/cmd/gvite/gvite
  ```

### Configuration file
  `node_config.json` is gvite configuration file. It should reside in the same directory with gvite executables. Details can be found at: [Config Description](./node_config.md)

### Boot script
  Taking Linux as example, the script has the following content:
  ```
  nohup ./gvite -pprof >> gvite.log 2>&1 &
  ```

## Monitoring

### Query snapshot block height in command line

* Start a full node as instructed above
* Connect to gvite command line console: Navigate to [Full Node Installation Directory](./install.md#Description-of-installation-directory) and execute the following command

  Linux/Unix:
  ```bash
  ./gvite attach ~/.gvite/maindata/gvite.ipc
  ```
  Windows:
  ```bash
  gvite-windows-amd64.exe attach \\.\pipe\gvite.ipc
  ```
  Then execute command:
  ```javascript
  vite.ledger_getSnapshotChainHeight();
  ```
  The following result will be displayed:
  ```
  "{\"id\":0,\"jsonrpc\":\"2.0\",\"result\":\"51821203\"}"
  ```
  51821203 is current block height. 
  
* For more information please run command `vite.help`.
  
## Full node rewards

In Vite Pre-Mainnet, rewards will be distributed to full node owners as incentives. 

### Node configuration

Additional settings in `node_config.json` are required:
* Set full node stats URL: `"DashboardTargetURL": "wss://stats.vite.net"`
* Add "dashboard" to `PublicModules`
* Set `"RewardAddr": "${your_address}"` to receive full node reward

The modified part of node_config.json is as below(please note this is not the full config file):
```
  "PublicModules": [
    "ledger",
    "public_onroad",
    "net",
    "contract",
    "pledge",
    "register",
    "vote",
    "mintage",
    "consensusGroup",
    "tx",
    "dashboard"  // new add
  ],
  "DashboardTargetURL":"wss://stats.vite.net",  // new add
  "RewardAddr":"vite_youraddress"   // new add
```

### Node status check

Reboot full node, then visit [Vite Explorer](https://explorer.vite.net/FullNode) to examine if your node has shown up correctly (result will reflect in 5 minutes).
Scroll to filter box and input your node name from above.
  
## Next steps

* [Super node configuration](./sbp.md)
* [Wallet management](./wallet-manage.md)
* [Super node rules](../rule/sbp.md)
