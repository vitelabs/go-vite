---
order: 6
---

# Run SBP Node on Ubuntu 16.04

:::tip
This document explains how to set up an SBP node. All steps have been tested on ubuntu 16.04.
:::


## Install gvite

### Install from a Binary Package

Latest installation package can be found at [gvite Releases](https://github.com/vitelabs/go-vite/releases).


```bash replace version
## Download
curl -L -O https://github.com/vitelabs/go-vite/releases/download/${version}/gvite-${version}-linux.tar.gz
```
```bash replace version
## Unpack package
tar -xzvf gvite-${version}-linux.tar.gz
```
```bash replace version
## Enter the folder. You should see 3 files: gvite, bootstrap and node_config.json
mv gvite-${version}-linux vite
cd vite
```
```bash
## Config node_config.json and then save
vi node_config.json
```
```bash
## Boot up gvite node
./bootstrap
```

### Check if gvite is Started

Check the content of gvite.log in the same folder to determine whether the program is up and running.

```bash
cat gvite.log
```

The following messages indicate that boot is successful.

```bash
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.DataDir:/home/ubuntu/.gvite/maindata module=gvite/node_manager
t=2018-11-09T17:44:48+0800 lvl=info msg=NodeServer.KeyStoreDir:/home/ubuntu/.gvite/maindata/wallet module=gvite/node_manager
Prepare the Node success!!!
Start the Node success!!!
```

### Obtain the Path of Installation Directory 

```bash
pwd
```

Please write down the path, which will be used later

For example, if you logged in as root user, the installation directory is:

```bash
/root/vite
```

## Create Wallet
### Create a New Wallet  
  
Execute the following command
```javascript
./gvite rpc ~/.gvite/maindata/gvite.ipc wallet_createEntropyFile '["123456"]'
```
Here `123456` is keystore's password, you should replace it with your own password.

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

### Check if the Wallet has been Created Successfully

Execute the following command
```bash
ls ~/.gvite/maindata/wallet/
```
The following result should be displayed

```bash
vite_f1c2d944b1e5b8cbfcd5f90f94a0e877beafeced1f331d9acf
```
`vite_f1c2d944b1e5b8cbfcd5f90f94a0e877beafeced1f331d9acf` is the wallet address created above. Multiple addresses will be displayed if more than one keystore files were created. 

## Edit node_config.json

```bash
vi node_config.json
```

Edit following content：

```
        "Miner": true,
        "CoinBase": "0:${your_address}",
        "EntropyStorePath": "${your_address}",
        "EntropyStorePassword": "${your_password}",
```

* `${your_address}`: Your wallet address created above. This address will be used to produce snapshot blocks
* `${your_password}`: Your wallet password

Save and quit

## Reboot Node

Kill existing gvite process

```bash
pgrep gvite | xargs kill -9
```

Reboot

```bash
./bootstrap
```
Check if gvite is rebooted successfully


```bash
ps -efww|grep -w 'gvite'
```

Below output indicates reboot is completed successfully:


```bash
root      6560  5939  0 12:29 pts/1    00:00:00 grep --color=auto -w gvite
```

## Query Current Snapshot Height in Command Line

Input：
```bash
  ./gvite rpc ~/.gvite/maindata/gvite.ipc ledger_getSnapshotChainHeight
```
Output:
```json
  "{\"id\":0,\"jsonrpc\":\"2.0\",\"result\":\"499967\"}"
```
499967 is current block height.

For more command usage please run command `vite.help`.

## Config gvite as Auto-start Service

### Create install.sh
```bash
## Navigate to gvite installation directory, and make sure it contains gvite and node_config.json
cd vite
ls
```

```bash
## Create install.sh and copy below script content in
vi install.sh
```

```text

#!/bin/bash

set -e

CUR_DIR=`pwd`
CONF_DIR="/etc/vite"
BIN_DIR="/usr/local/vite"
LOG_DIR=$HOME/.gvite

echo "install config to "$CONF_DIR


sudo mkdir -p $CONF_DIR
sudo cp $CUR_DIR/node_config.json $CONF_DIR
ls  $CONF_DIR/node_config.json

echo "install executable file to "$BIN_DIR
sudo mkdir -p $BIN_DIR
mkdir -p $LOG_DIR
sudo cp $CUR_DIR/gvite $BIN_DIR

echo '#!/bin/bash
exec '$BIN_DIR/gvite' -pprof -config '$CONF_DIR/node_config.json' >> '$LOG_DIR/std.log' 2>&1' | sudo tee $BIN_DIR/gvited > /dev/null

sudo chmod +x $BIN_DIR/gvited

ls  $BIN_DIR/gvite
ls  $BIN_DIR/gvited

echo "config vite service boot."

echo '[Unit]
Description=GVite node service
After=network.target

[Service]
ExecStart='$BIN_DIR/gvited'
Restart=on-failure
User='`whoami`'
LimitCORE=infinity
LimitNOFILE=10240
LimitNPROC=10240

[Install]
WantedBy=multi-user.target' | sudo tee /etc/systemd/system/vite.service>/dev/null

sudo systemctl daemon-reload
```

```bash
## Grant executable permission
sudo chmod +x install.sh
```

### Run install.sh and Enable Service

```bash
## Run install.sh
./install.sh

## Set auto-start
sudo systemctl enable vite
```

### Start gvite service

```bash
## Kill original gvite process
pgrep gvite | xargs kill -s 9

## Check result
ps -ef | grep gvite

## Start gvite service
sudo service vite start

## Check result
ps -ef | grep gvite

## Check service status
sudo service vite status

## Check boot log
tail -n200 ~/.gvite/std.log
```
Below message will be displayed if the service has been started up successfully:
```text
vite.service - GVite node service
   Loaded: loaded (/etc/systemd/system/vite.service; disabled; vendor preset: enabled)
   Active: active (running) since Thu 2018-11-22 21:23:30 CST; 1s ago
 Main PID: 15872 (gvite)
    Tasks: 7
   Memory: 12.1M
      CPU: 116ms
   CGroup: /system.slice/vite.service
           └─15872 /usr/local/vite/gvite -pprof -config /etc/vite/node_config.json

Nov 22 21:23:30 ubuntu systemd[1]: Started GVite node service.
```

### Shut down gvite Service
```bash
sudo service vite stop
```

!!! Gvite service config is located in /etc/vite. Gvite console messages are logged in $HOME/.gvite/std.log.

## Tips
### Print Current Snapshot Block Height 

Execute below command in gvite command line console

```bash
./gvite rpc ~/.gvite/maindata/gvite.ipc  ledger_getSnapshotChainHeight
```

The latest height will be printed out in every second. Run `exit` to abort.
