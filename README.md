<div align="center">
    <img src="https://github.com/vitelabs/doc.vite.org/blob/master/docs/.vuepress/public/logo_black.svg" alt="Logo" width='300px' height='auto'/>
</div>

<br />

[Vite](https://vite.org) is a next-generation Reactive Blockchain that adopts a _message-driven, asynchronous architecture and a DAG-based ledger_.
The goal for Vite’s design is to _provide a reliable public platform for industrial dApps_, with features of ultra-high throughput and scalability.

## Go Vite

Official golang implementation of Vite Protocol

![GitHub release](https://img.shields.io/github/release/vitelabs/go-vite.svg)
![Go Report Card](https://goreportcard.com/badge/github.com/vitelabs/go-vite)
![GitHub closed pull requests](https://img.shields.io/github/issues-pr-closed/vitelabs/go-vite.svg)
![Downloads](https://img.shields.io/github/downloads/vitelabs/go-vite/total.svg)
<!-- ![Discord](https://img.shields.io/discord/:serverId.svg) -->


The go-vite binary files can be download from [releases](https://github.com/vitelabs/go-vite/releases).


## Guides & Documentation
   * [White Paper](https://www.vite.org/whitepaper/vite_en.pdf)
   * [Documentation](https://vite.wiki/)
   * [Techblog](https://vite.blog/)
   * [Runing a node](https://vite.wiki/tutorial/node/install.html)
   
<!-- ## Develop -->

## Product
   * [Products Navigation](https://vite.net)
   * [Web Wallet](https://wallet.vite.net)
   * [Desktop Wallet](https://github.com/vitelabs/vite-wallet)
   * [Wallet App](https://app.vite.net) open through mobile browser
   * [TestNet Block Explorer](https://testnet.vite.net)
   
## Links & Resources
   * [Project Vite official Website](https://www.vite.org/)
   * [Vite products](https://vite.net)
   * [Twitter](https://twitter.com/vitelabs)
   * [Telegram](https://t.me/vite_en)
   * [Telegram Announcement](https://t.me/vite_ann)
   * [Reddit](https://www.reddit.com/r/vitelabs)
   * [Discord](https://discordapp.com/invite/CsVY76q)
   * [Youtube](https://www.youtube.com/channel/UC8qft2rEzBnP9yJOGdsJBVg)
   * [Forum](https://forum.vite.net/)



## Build the source

1. [Install Go](https://golang.org/doc/install)
2. Run `go get github.com/vitelabs/go-vite` in your terminal, then you will find the source code here: `$GOPATH/src/github.com/vitelabs/go-vite/` (as default, $GOPATH is `~/go`)
3. Go to the source code directory and run `make gvite`, you will get an executable file here: `$GOPATH/src/github.com/vitelabs/go-vite/build/cmd/gvite/gvite`
4. Configuration use config file `node_config.json` to set Config, the file should be in the same directory of vite. you can use the default config to connect to the testNet.
5. Run the appropriate binary file on your OS. eg.  use ```nohup ./gvite >> gvite.log 2>&1 &``` to start the node.



## Contribution

We are very very welcome contributions from anyone, even if little change (eg. improving comment, formatter) can help us make VITE better!

If you\`d like to contribute to vite, please fork, fix, commit and make a pull request. We\`ll review and handle the PR. Please 
make sure your code follows our guidelines:

- Code must be formatted&commented to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting)&[comment](https://golang.org/doc/effective_go.html#commentary) guidelines

- Pull request is recommended to be base on `master` branch



## License

The go-vite source code is under [GPLv3](https://www.gnu.org/licenses/gpl-3.0.html), also announced in the `LICENSE` file
in our code repository.
