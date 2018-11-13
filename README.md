<hr />
<div align="center">
    <img src="https://github.com/vitelabs/doc.vite.org/blob/master/docs/.vuepress/public/logo_black.svg" alt="Logo" width='300px' height='auto'/>
</div>
<hr />

## What is Vite?

---

Vite is a next-generation Reactive Blockchain that adopts a _message-driven, asynchronous architecture and a DAG-based ledger_. The goal for Vite’s design is to _provide a reliable public platform for industrial dApps_, with features of ultra-high throughput and scalability.


---

## Guides & Documentation
   * [White Paper](https://www.vite.org/whitepaper/vite_en.pdf)
   * [Documentation](https://doc.vite.org/)
   * [Techblog](https://vite.blog/)
   
## Product
   * [Desktop Wallet](https://github.com/vitelabs/vite-wallet)
   * [Testnet Explorer](https://testnet.vite.net/)
   * [Vite.net](https://vite.net/)
   
## Links & Resources
   * [Website](https://www.vite.org/)
   * [Twitter](https://twitter.com/vitelabs)
   * [Telegram](https://t.me/vite_en)
   * [Telegram Announcement](https://t.me/vite_ann)
   * [Reddit](https://www.reddit.com/r/vitelabs)
   * [Discord](https://discordapp.com/invite/CsVY76q)
   * [Youtube](https://www.youtube.com/channel/UC8qft2rEzBnP9yJOGdsJBVg)
   
## BUILD 

1. [Install Go](https://golang.org/doc/install)
2. Run `go get github.com/vitelabs/go-vite` in your terminal, then you will find the source code here: `$GOPATH/src/github.com/vitelabs/go-vite/` (as default, $GOPATH is `~/go`)
3. Go to the source code directory and run `make all`, you will get executable files of darwin, linux, windows here: `$GOPATH/src/github.com/vitelabs/go-vite/build/cmd/gvite`
4. Run the appropriate binary file on your OS.


## CONFIG

As default, Vite will give a default config. but you can set your config use two way as following.

### cmd

| key | type | default | meaning |
|:--- |:--- |:--- |:--- |
| name | string | "vite-server" | the server name, use for log |
| maxpeers | number | 50 | the maximum number of peers can be connected |
| addr | string | "0.0.0.0:8483" | will be listen by vite |
| dir | string | "~/viteisbest" | the directory in which vite will store all files (like log, ledger) |
| netid | number | 2 | the network vite will connect, default 2 means TestNet |
| priv | string | "" | the hex code string of ed25519 privateKey |

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
