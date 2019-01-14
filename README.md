<hr />
<div align="center">
    <img src="https://github.com/vitelabs/doc.vite.org/blob/master/docs/.vuepress/public/logo_black.svg" alt="Logo" width='300px' height='auto'/>
</div>
<hr />

## What is Vite?

> Vite is a next-generation Reactive Blockchain that adopts a _message-driven, asynchronous architecture and a DAG-based ledger_. The goal for Vite’s design is to _provide a reliable public platform for industrial dApps_, with features of ultra-high throughput and scalability.

## Guides & Documentation
   * [White Paper](https://www.vite.org/whitepaper/vite_en.pdf)
   * [Documentation](https://vite.wiki/)
   * [Techblog](https://vite.blog/)
   * [Runing a node](https://vite.wiki/tutorial/node/install.html)
   
## Product
   * [Web Wallet](https://wallet.vite.net)
   * [Desktop Wallet](https://github.com/vitelabs/vite-wallet)
   * [iOS Wallet](https://itunes.apple.com/us/app/vite-official-hd-wallet/id1437629486?mt=8)
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

## RELEASE VERSION
the stable version is the v1.2.2 tag. https://github.com/vitelabs/go-vite/releases/tag/v1.2.2

## BUILD 

1. [Install Go](https://golang.org/doc/install)
2. Run `go get github.com/vitelabs/go-vite` in your terminal, then you will find the source code here: `$GOPATH/src/github.com/vitelabs/go-vite/` (as default, $GOPATH is `~/go`)
3. Go to the source code directory and run `make gvite`, you will get an executable file here: `$GOPATH/src/github.com/vitelabs/go-vite/build/cmd/gvite/gvite`
4. Configuration use config file `node_config.json` to set Config, the file should be  in the same directory of vite. you can use the default config to connect to the testNet 1.0.0 network.
5. Run the appropriate binary file on your OS. eg.  use ```nohup ./gvite >> gvite.log 2>&1 &``` to start the node.


## CONFIG

As default, Vite will give a default config. but you can set your config use two way as following.


### configFile

use config file `node_config.json` to set Config. for example:

```json
{
  "NetID": 2,
  "Identity": "vite-fullNode",
  "PrivateKey": "",
  "MaxPeers": 200,
  "MaxPendingPeers": 20,
  "BootNodes": [
    "vnode://864c763b198f7234e90e25c935c77f84866def8590afec4af1545ca2e45ca926@3.8.77.15:8483","vnode://c4134dcfa3d2630613e5dae9efdc69a6eb94554a5039e56e8aa0992ab22945c6@34.247.68.140:8483","vnode://766fbe9b0406d1978b4f433e558e1895e94c3698e6c29ec2c2042a5e516825a1@35.182.1.144:8483","vnode://88e9933d098cad9a387cdd5ea2431c9fcb9abf0f98f95a9a7773d616cf8eab77@54.164.163.91:8483","vnode://63b8794c10ee807f8f4617187d9eeac06532aee023f7d1f3484748d092ebf759@54.245.179.219:8483","vnode://9355d23d1be9659987a019953ba5fd22a722db89914075004560862a909a371b@13.113.140.139:8483","vnode://1ce4ce54cc978fdc333398bbb8beda3ae3fe3eacc34d04de1976d7fb91074406@52.78.84.56:8483","vnode://8a6079744a54147dd6e95ec66aed5aac52bec5b5f5d85426e3888bda22a9f6f2@13.229.135.72:8483","vnode://3ada84473109cc881d65c3d80dfef348c2f6f038c52f5b9dcea1e96cb3ebc2e9@13.233.84.63:8483","vnode://6913de145fe933f2ba2835ab33a00c289b93167ce82e7bcccffedb67d7e19e3f@18.194.106.196:8483","vnode://99d333bc795cb2b42f1a64309669356ae47cac8a5fc652ca39b212bd0bb8564b@13.210.254.88:8483","vnode://22ac75beb6302823c15003fdf2972f4d1c8690e2afffa9aa76b7c7826372ca2a@45.32.120.252:8483","vnode://0b459ee0817dc0e59dacff0d257220ea69aa7fb7ac88633df592ea20b13b6419@104.238.189.237:8483","vnode://2b7cb786a1f7745b743139dfcd8a8a8323d7610da52cb2f2d4f27b1d0531e09e@108.61.170.32:8483","vnode://6a01f4333f6b6466229d6cdf88892ef57c8ef78aaf41f9a5ae0d4938b59a3f31@95.179.147.156:8483","vnode://fb528a6231fee579d7797679c128b7efef72f486b58881e06df52fd41b381900@118.25.177.35:8483","vnode://11da939194ff9e605072608d86faacd06f7aa0fe443db4267025a701aac9c26b@118.25.72.17:8483","vnode://681e4ffd550a86b2b308fc2058660acc1deb87b09ccb5cf7682b324414698e74@118.25.141.229:8483","vnode://17d4fa71d89b06452c6e1fbd5b859550ff4ed55cadf519f155cd5a9aaf6c18f7@119.28.32.48:8483","vnode://f0929aaaf8a8f7bb11494c0d973b52c6776313d26ad83fa124abcde7aa54ff46@119.28.221.175:8483","vnode://e83d7675cefe682a5fc801d490c423e09f811a7464b7ac4e6bbc6642183dd229@150.109.40.238:8483","vnode://f5d44b70b561471ec96bab6bc2313b1efa71022f0f1ecbe73860d1edfa2434d3@150.109.46.50:8483","vnode://c201fb8388f7e7aabf21c851c7f75c5eda66f094c94866e5d9388e9c4fef4246@150.109.101.112:8483","vnode://23c36e0e5f4fe2e1daf9af7bd91c7fc2a84453152fde4ff9422118ff50e28e7a@35.236.34.242:8483","vnode://f2d3b0bd08b14d7b50149b259524907ccc63297173b129c496e64307aa4feef1@35.231.210.8:8483","vnode://5e3520758a462b9f8175ce872090d5bd44342aac52c4704f0d12128acd610096@150.109.105.154:8483","vnode://61afd431ccd9079fc644acc7c643f04e4b92c379f5c8ab92e4fe11a87ee1bd59@118.25.109.87:8483","vnode://cedf763228c7fa841b67ee04e57d7ee6d2e90e927585c0f96872b8ee92a1e4ff@118.25.49.80:8483","vnode://cb4153736d23d1858f621447963c54e8c0e0fae71a1529ad57ea86e3ba22760a@118.24.129.159:8483","vnode://abdfba548c32b0dd8ae7265def5314a9ea98f231939a6552cf000ef7962c327f@118.24.112.219:8483","vnode://8f89b521d4ce2437fe5872287187646a06a9ca2810d2988469ed6ee8a2003ab8@118.24.26.130:8483","vnode://b3bfad13fe29078c7719256345ffb871a8184af211e45fd2ad9ee1f3b155f5eb@118.24.112.185:8483","vnode://2e0ae36065b544d82f1b9e04e51c0c12d4596279f1924118550d414f016e1345@118.24.80.136:8483","vnode://445fac2e8045f53ebe6da7f4c173820ab303d11b047e6fc381d5c1f96e12df4a@188.131.179.254:8483","vnode://af1a36543edbcb473254eb46359f16e9f63dc96468017511448648217788cf12@188.131.180.157:8483","vnode://62c05a8850ae35f91d1c729412376e046df1a151d54b9d6727247824450abd1e@188.131.150.140:8483","vnode://697ead367c7121a05424ba36749f36d4b769339a8077f776a0aaacc3bc6bc1de@188.131.179.248:8483","vnode://1d39caaf81e89e5d711b10b33e3097d538d8f7858244357eb492e3e3e6a6fab5@140.143.8.202:8483","vnode://f0591ba79efd68de030fb2e49607f87ea944c40652d82f29305c2c28b7d5b4e7@139.199.74.104:8483","vnode://962216b6287fab85f92adf2f8b289fca528eb8a533388d1ff75aa7c16f8a8eb3@134.175.105.236:8483","vnode://1514ec5f5fb9628dfce9b2cf6ccb0bc9a59166f266f08ebe977c396a977cf0e2@139.199.76.167:8483","vnode://b877dc9d759a78e39e8e37ec6f68963ef78f5d5b7d367bc007e7113b3dc97eeb@134.175.1.34:8483","vnode://2bcdda8b936ccf3aac2c87960e20b6be458e82fc65e64ceb428b8d2873549479@134.175.18.252:8483"
  ],
  "Port": 8483,
  "RPCEnabled": false,
  "HttpHost": "0.0.0.0",
  "HttpPort": 48132,
  "WSEnabled": false,
  "WSHost": "0.0.0.0",
  "WSPort": 41420,
  "IPCEnabled": true,
  "HttpVirtualHosts": ["*"],
  "PublicModules":[
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
    "debug",
    "pow",
    "testapi"
  ],
  "Miner": false,
  "CoinBase": "",
  "EntropyStorePath": "",
  "EntropyStorePassword": "",
  "TopoDisabled": true,
  "LogLevel": "warn"
}
```

`node_config.json` should be in the same directory of vite.
