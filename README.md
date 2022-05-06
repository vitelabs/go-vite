<div align="center">
    <img src="https://github.com/vitelabs/doc.vite.org/blob/master/docs/.vuepress/public/logo_black.svg" alt="Logo" width='300px' height='auto'/>
</div>

<br />

[Vite](https://vite.org) is a next-generation Reactive Blockchain that adopts a _message-driven, asynchronous architecture and a DAG-based ledger_.
The goal for Vite’s design is to _provide a reliable public platform for industrial dApps_, with features of ultra-high throughput and scalability.

## Go Vite

Official golang implementation of Vite Protocol

[![GitHub release](https://img.shields.io/github/release/vitelabs/go-vite.svg)](https://github.com/vitelabs/go-vite/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/vitelabs/go-vite)](https://goreportcard.com/report/github.com/vitelabs/go-vite)
[![GitHub closed pull requests](https://img.shields.io/github/issues-pr-closed/vitelabs/go-vite.svg)](https://github.com/vitelabs/go-vite/pulls)
[![Downloads](https://img.shields.io/github/downloads/vitelabs/go-vite/total.svg)](https://github.com/vitelabs/go-vite/releases)


The go-vite binary files can be download from [releases](https://github.com/vitelabs/go-vite/releases).


## Guides & Documentation
   * [White Paper](https://github.com/vitelabs/whitepaper/blob/master/vite_en.pdf)
   * [Documentation](https://docs.vite.org)
   * [Techblog](https://docs.vite.org/vite-docs/articles/)
   * [Runing a node](https://vite.wiki/tutorial/node/install.html) 

## Product
   * [Products Navigation](https://vite.net)
   * [Web Wallet](https://wallet.vite.net)
   * [Desktop Wallet](https://github.com/vitelabs/vite-wallet)
   * [Wallet App](https://app.vite.net) open through mobile browser
   * [Block Explorer](https://vitescan.io/)

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

## Installation

You can choose one of the following installation options:

- [Install from binary](https://docs.vite.org/vite-docs/tutorial/node/install.html#install-from-binary)
- [Install from source](https://docs.vite.org/vite-docs/tutorial/node/install.html#install-from-source)

### Faster ledger sync

[download](ledger_snapshot.md) gvite ledger file manually.

## Versioning

Given a version number MAJOR.MINOR.BUILD, increment the:

1. MAJOR version when you make a significant update of the protocol (like Ethereum, Ethereum 2.0 is rather a new blockchain from Ethereum 1.0),
2. MINOR version when you introduce some breaking changes, and
3. BUILD version when you make non-breaking changes such as bug fixes, RPC modifications, code refactoring, test updates, etc.

## Branching

The `master` branch is the working branch for the next release. 
- Breaking changes should not be merged into `master` unless as described below.
- Non-breaking changes should be merged into `master`.

1) A new branch should be checked out from `master` before releasing a new version. 
    - Say the latest version running on mainnet is 2.12.2, we can checkout a branch named `release_v2.12.3` from `master` for building, deploying on testnet, releasing the binary and deploying on mainnet.

2) A development branch for the next breaking changes, for example `v2.13`, will be checked out from `master`. 
    - Any commits from `master` are required to be merged into `v2.13` (or rebase `v2.13` onto `master`) as soon as possible. 
    - All unit tests and integration tests should pass before merging a pull request.

3) Merge `v2.13` into `master` and checkout `release_v2.13.0` from `master` before the mainnet upgrade. 
    - This means that v2.13.0 is the next release and there are no more releases of version 2.12.x. 
    - The branch `v2.13` reached end-of-life and a new branch `v2.14` for the next breaking changes should be checked out from `master`.
    - From this point onwards, any new commits which are merged into `master` are based on version 2.13.

Any changes should be committed to your personal fork followed by opening a PR in the official repository.

## Contribution

Thank you for considering to help out with the source code! We welcome any contributions no matter how small they are!

If you'd like to contribute to go-vite, please fork, fix, commit and send a pull request for the maintainers to review and merge into the main code base.

Please make sure your contributions adhere to our coding guidelines:

- Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines.
- Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
- Pull requests need to be based on and opened against the `master` branch.
- Open an issue before submitting a PR for non-breaking changes.
- Publish a VEP proposal before submitting a PR for breaking changes.

## License

The go-vite source code is licensed under [GPLv3](https://www.gnu.org/licenses/gpl-3.0.html), also included in the `LICENSE` file.
