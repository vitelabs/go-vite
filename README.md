# go-vite

Official Go implementation of the Vite protocol


## BUILD 

1. [Install Go](https://golang.org/doc/install)
2. Run `go get github.com/vitelabs/go-vite` in your terminal, then you will find the source code here: `$GOPATH/src/github.com/vitelabs/go-vite/` (as default, $GOPATH is `~/go`)
3. Go to the source code directory and run `make all`, you will get executable files of darwin, linux, windows here: `$GOPATH/src/github.com/vitelabs/go-vite/build/cmd/rpc`  
4. Run the appropriate binary file on your OS.


## CONFIG

As default, Vite will give a default config. but you can set your config use two way as following.

### cmd

| key | type | default | meaning |
|:--|:--|:--|
| name | string | "vite-server" | the server name, use for log |
| maxpeers | number | 50 | the maximum number of peers can be connected |
| addr | string | "0.0.0.0:8483" | will be listen by vite |
| dir | string | "~/viteisbest" | the directory in which vite will store all files (like log, ledger) |
| netid | number | 2 | the network vite will connect, default 2 means TestNet |


### configFile

we can also use config file `vite.config.json` to set Config. for example:

```json
{
    "P2P": {
        "Name":                 "vite-server",
        "PrivateKey":           "",
        "MaxPeers":             100,
        "Addr":                 "0.0.0.0:8483",
        "NetID":                2
    },
    "DataDir": ""
}
```

`vite.config.json` should be in the same directory of vite.
