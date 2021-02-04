# Simple Introduction to Vite <Badge :text="$page.version"/>

The Vite Mainnet was launched on September 25, 2019. Visit [Token Swap Guide](https://medium.com/vitelabs/announcing-the-vite-mainnet-launch-4d55fc4b4bd2) to learn about how to convert ERC20 VITE to native Vite coins.

## How to Get Vite Coins

### Purchase at Exchanges

* [ViteX][vitex]
* [Binance][binance]
* [OKEx][okex]
* [Bittrex][bittrex]
* [Upbit][upbit]

### SBP Rewards

**SBP (Snapshot Block Producer)** is responsible for producing snapshot blocks and securing consensus on the Vite chain. SBP rewards are issued to incentivize the SBP nodes. There are 25 SBP nodes in Vite Mainnet.

:::tip
SBP reward is **0.951293759512937595** VITE per snapshot block plus additional voting reward. The rewards are issued on-chain. Each SBP should send an explicit **Reward Retrieval Transaction** to get rewards.
:::

Related links:

* [How to Run a SBP][sbp-manage]
* [Rules for SBP Rewards Allocation][sbp-reward]

### Full Node Rewards

Full node is the cornerstone of the Vite ecosystem. The **Full Node Reward Program** was firstly launched on December 13, 2018 in Vite Testnet, and has been upgraded on January 4, 2020. Full node rewards are distributed on a daily basis.

Related links:

* [Rules for Full Node Rewards][fullnode-reward]
* [Configuration Specification](../node/install.md#full-node-reward)

## Features of Vite Mainnet

### Fee-less Transactions

Transactions are free on Vite. There is no "Gas" (as in Ethereum) charged. To address the transaction cost, Vite has implemented a **Quota-Based Model** to measure the number of transactions the account can process in UT (Unit Transaction). 

Think of "Quota" as the fuel that keeps the Vite public chain running. Every transaction that is sent requires a certain amount of quota. For users that have high transaction frequency, they will need to obtain quota by staking VITE coins.

Nevertheless, users who do not send frequent transactions can choose to run a quick PoW on their respective devices to get a small amount of quota instead.

### HDPoS Consensus

The consensus algorithm of Vite is called **Hierarchical Delegated Proof of Stake(HDPoS)**. 

HDPoS is a multi-tiered algorithm featured by different levels of consensus groups. The eventual consensus in HDPoS is secured by supernodes of **Snapshot Consensus Group**, while nodes in **Delegated Consensus Group** and **Private Consensus Group** are responsible for reaching consensus for transactions of smart contracts and user accounts.

The existence of multiple instances of consensus group in HDPoS guarantees that transactions can be verified, written into the ledger, and confirmed across smart contracts and account chains at high speed. 

### Fast Transactions

The DAG ledger and asynchronous communication scheme play a big role in achieving high performance. In general, Vite's asynchronous design lies in three areas: 

* Asynchronous Design of Request and Response 
* Asynchronous Design of Transaction Writing and Confirmation 
* Asynchronous Design of Cross-Contract Calls

### Built-In Smart Contracts

Built-in smart contracts are widely used in Vite. Features like SBP registration, voting, staking, token issuance, and ViteX exchange, are implemented in built-in smart contracts.

#### SBP Registration

In the Mainnet, the steps for launching an SBP node are as follows:

* Staking **1,000,000** VITE with a locking period of 3 months;
* Operating a node server and having skills to maintain the said server;
* Having substantial community influence and being able to solicit votes from VITE holders.

#### Voting for SBP

Many SBPs distribute voting rewards to supporters. Users can vote for them and get rewards! The VITE balance in the user's address will count toward the voting weight. Voting does not lock VITE, and the user can revote to another SBP at any time. 

:::tip Notice
NOT every SBP will distribute voting rewards. Refer to the SBP list that gives rewards on the Vite forum.
:::

#### Staking for Quota

It is recommended to get quota through staking. In the Mainnet, the minimum staking amount is 134 VITE, and there is no maximum limit. 

**Staking Lock-up Period**: 3 days

The lock-up period is defined as the period of time in which the staked VITE are locked up and cannot be withdrawn. When the lock-up expires, the staking address will be able to withdraw the staked coins.

#### Staking for a Recipient Account

The staking address can designate a quota beneficiary. If not specified, the default beneficiary is the staking address.

#### Staking Withdrawal

The staking address can cancel a staking after the lock-up period has passed and retrieve the VITE coins staked. In correspondence with the minimum staking amount, the minimum withdrawal amount is also **134** VITE.

#### Token Issuance

Unlike Ethereum on which the issuer has to write ERC20 smart contract to mint a token, issuing tokens on Vite is simplified as just filling a form. This greatly lowers the barrier of token issuance with additional security improvement by avoiding from introducing unexpected code flaws.

## Coins on Vite

**VITE**, and two other native coins, **VCP** and **VX**, are officially issued in Vite Mainnet.

### VITE
Full name：*Vite Coin*

VITE is the native coin of the Vite public chain. VITE is the fuel to power up transactions on the Vite chain. Unlike the traditional fuel of "Gas" that will be consumed on other public chains (e.g. Ethereum), VITE is used to stake for the computational and bandwidth resources, and can be retained at no cost once the transactions are complete. VITE is currently listed on the top exchanges and can be traded for serious investment considerations.

### VCP

Full name：*Vite Community Points*

VCP is the credit of the community. They are distributed for free to recognize community members who have significant contribution. At the time being, VCP is mainly used for redeeming Vite merchandise (such as T-shirts and hats) in ViteStore. 
VCP will not be listed or trade in exchange.

### VX

Full name：*ViteX Coin*

As the platform token of ViteX, VX holders benefit from the exchange's dividend distributions and governance privileges. All VX are 100% mined through real tradings, staking, coin listings, referrals, and market-making activities based on an agreed remittance schedule.

[sbp-reward]: <../rule/sbp.html#SBP-rewards>
[fullnode-reward]: <../rule/fullnode.html>
[fullnode-install]: <../node/install.html>
[sbp-manage]: <../node/sbp.html>
[web-wallet]: <https://wallet.vite.net>
[app-wallet]: <https://app.vite.net>
[vitex]: <https://x.vite.net/trade?symbol=VITE_BTC-000&category=BTC>
[okex]: <https://www.okex.com/spot/trade#product=vite_btc>
[bittrex]: <https://international.bittrex.com/Market/Index?MarketName=BTC-VITE>
[upbit]: <https://upbit.com/exchange?code=CRIX.UPBIT.BTC-VITE>
[binance]: <https://www.binance.com/en/trade/VITE_BTC>
[solidity++]: </zh/tutorial/contract/soliditypp.html>

