# FAQ

::: tip Where are all the questions?

You might know there's a hackathon coming up, and we're expecting to see a lot of new developers on Vite! (See ViteLab's [Gitcoin's GR10 Hackathon page](https://gitcoin.co/hackathon/gr10/?org=vitelabs)).

As such, we're doing updates on this FAQ as new good questions come in. Visit the [Discord](https://discordapp.com/invite/CsVY76q) or our [Reddit](https://www.reddit.com/r/vitelabs/) and ask away. Expect to see this page change quickly.

—Wes (Sunday June 13, 20:56:25 PDT 2021)
:::


### Testnet

#### Q: How do I connect to Testnet in my code?
The new Vite TestNet is alive at [https://buidl.vite.net/](https://buidl.vite.net/).
Use `wss://buidl.vite.net/gvite/ws` or `https://buidl.vite.net/gvite` to connect to the Testnet in your code. 

#### Q: How to get Testnet tokens?
Go to Vite Discord and DM ```!send vite_YOURTESTNETVITEADDRESS``` to @faucet#9018. You will get 10k Testnet VITE each time. 

#### Q: Is there a Testnet browser? 
Open [http://viteview.xyz/](http://viteview.xyz/), fill in `https://buidl.vite.net/gvite` or `http://18.217.135.251:48132` in the left menu and save. 

#### Q: Can I use the Vite app in the Testnet? 
The Vite app wallet does not fully support the Testnet. So please do not use the Vite app in the Testnet. You can generate a test-only seed phrase in the Vite app (or in the soliditypp Visual studio code extension), and then restore it into the Testnet web wallet. (https://buidl.vite.net/).

#### Q: How can I use the ViteX API in the Testnet? 
Use ```https://buidl.vite.net/vitex/api/v2``` along with the [Trade API](https://docs.vite.org/go-vite/dex/api/dex-apis.html).