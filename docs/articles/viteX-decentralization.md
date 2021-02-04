# ViteX’s Decentralization Explained


Generally speaking, if one certain sub-system of exchange is implemented in a decentralized approach, it can be categorized as a decentralized exchange in a sense. If we look at the existing decentralized exchanges from the perspective of decentralization degree, we can find that some of them are more decentralized, and some are less. In regards to what components should be decentralized, and to what degree of decentralization an exchange should achieve, there is no common understanding in the industry, and the actual implementation of the exchange reflects the architect’s view of decentralization, flexibility, performance, and trade-offs between them. As a firm practitioner of DEX, ViteX’s mission is to build an exchange of the best decentralization, leading to a hybrid approach of fully decentralized core components and, in order to take into account flexibility and user experience, a few centralized non-critical sub-systems. Let me explain in detail below.


## Decentralized Implementation


Sub-systems of asset management, order placement, order matching, settlement, mining, and dividends are decentralized on ViteX.


### 1. Asset Management


Asset management involves deposit, withdrawal and asset storage. In deposit, a user initiates a transaction in his account to call the built-in contract of dexFund to lock the deposit assets in the exchange. Except for the withdrawal interface, there is no other place to transfer assets out of the contract. In withdrawal, the user calls dexFund to initiate an asset transfer from the contract to the user’s account. The dexFund contract will create a key-value record in contract storage for each user to track asset balance. The cumulative balance of the accounts should be exactly the same as the asset balance held by the dexFund contract on the Vite chain.


### 2. Order Placement, Matching and Cancellation


When placing an order, the user initiates an order placement request to dexFund. The dexFund contract will verify the request, lock passed-in assets, and, if complete, call dexTrade contract to fill the order or fail otherwise. The matching engine in dexTrade adopts the Taker-Maker model on the basis of price-time priority. Let’s see an example.


For a newly submitted order A, if another order B in the order book can be matched, A becomes taker, and the order will be executed at the price of maker B. If A is not fully filled, it will attempt to match the next order on the order book until there is no further order to match or fill. In this case, A becomes the maker waiting for others to take.
Cancel an order is similar to order placement. It looks up the order id in the order book, and remove the order and unlock assets if the order id is found.
On ViteX, the process of order placement, matching, and cancellation are performed by two smart contracts (dexFund, dexTrade), and the order book is implemented in the dexTrade contract. It is a completely decentralized design.


### 3. Settlement


On ViteX, order matching and cancellation, mining, and dividends may involve assets settlement. Since the assets are secured in the dexFund contract, the settlement process is simplified by updating the balance of each account in the contract. For order matching and cancellation, the dexTrade contract initiates a request transaction containing a list of [address, token, change of amount] triples to dexFund, so the contract can look up in the key-value based storage and update the balance accordingly. For mining and dividends, since they are executed in dexFund, the settlement can be processed in the contract independently. Mining activities raise the circulating token supply, while dividends lock VX in the contract and distribute fee income to the accounts.


### 4. Mining


Mining on ViteX is related to VX minting, destuction, transfer, VX emittance curve, mining metrics and algorithm.
VX has no inflation. After minted, all VX tokens are transferred to the dexFund contract and locked. The emittance of VX is strictly following the release curve. The smart contract guarantees that no VX will be released ahead of schedule.
In December 2019, over 70m VX were burned according to the new emittance curve, through a special interface of the contract. The new mining curve is determined by an algorithm written in dexFund, which uses a base cycle and the distance of a given cycle to the base cycle to calculate the release quantity of the day, so no one can tamper with or interfere.
ViteX supports multiple mining methods, including stake mining, trade mining, referral mining, and liquidity mining. In stake mining, the balance change caused by staking or un-staking is tracked in the dexFund contract. Similar to stake mining, trade mining tracks the change of transaction fees while referral mining will calculate the cumulative rewards of both inviter and invitee in the contract too. At the end of a mining cycle, the dexFund contract executes the relevant mining algorithm to calculate the proportion of mined VX in total for each mining account and credit VX to the account.
The total mined VX of liquidity mining in each cycle will first be sent to a temporary account in dexFund, and then an off-chain service will call the dexFund contract to distribute VX from the temp account to individual mining accounts. Liquidity mining is not fully decentralized.
Except for liquidity mining, all mining activities on ViteX are completely decentralized.


### 5. Dividends


Dividends on ViteX involve VX locking, unlocking and dividend algorithm.
Locking and unlocking are initiated by the user, through a transaction call the dexFund contract. The contract tracks the locked VX of each account for the current cycle and calculates dividends at the cycle end. At present, the dividends distributed is 1% of the cumulative dividend pool, which is composed of 99% of the pool in the previous cycle plus the total trading fees collected in the current cycle. On ViteX, both dividend calculation and distribution are completed in dexFund, which makes it decentralized.


### 6. Trading Pairs


On ViteX, anyone can send 10,000 VITE to dexFund to open a new trading pair for the token owned and obtain operating right. The token ownership and operating right can be transferred. With operating right, one can set trading fees, and suspend or lift trading for the trading pair. All steps are completed by calling the corresponding interfaces of the dexFund contract, making trading pair operating decentralized.


## Centralized Implementation


### 1. Market Trend


The various market-related data, such as price change, trading volume, candlestick chart, and order depth, are generated from the on-chain order matching results by a centralized crawler service.


### 2. Advanced Order Query


Similar to market trend data, the order query on ViteX is supported by a centralized service too. Users can call the service API to look up historical orders, query unfilled orders according to different conditions, and cancel orders as needed.


### 3. Liquidity Mining


Compared with stake mining and trade mining, liquidity mining has too complex rules to be implemented in the contract. Therefore we put it in a centralized service. We will introduce liquidity mining in a following article.


## Summary


In this article, we have introduced the implementation details of the ViteX sub-systems from the perspective of decentralization. As a highly decentralized exchange, ViteX fulfills the commitments of traceability, transparency and security.












