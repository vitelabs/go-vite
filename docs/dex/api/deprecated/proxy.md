# ViteX API V1 (deprecated)

## Summary
ViteX Trading Service is a central service provided by Vite Labs, running on ViteX decentralized exchange. Users can set/cancel/query orders through the service without giving out private keys. 

* Trading is done by Vite Labs trading engine on behalf of you by providing **API Key** and **API Secret**. Your fund is safe in your exchange's account and cannot be misappropriated. 
* You must authorize to enable the service on trading pair explicitly. Trading service is inactivated without authorization. 
* You can cancel authorization at any time. 
* Orders set by trading service are not signed by your private key. Therefore, you cannot query the orders by your address. 

## Authorization on Trading Pair
In order to use trading service, authorization is required. 

It's highly recommended to authorize only on the specific trading pairs that you need to turn trading service on. Private keys are not required. DO NOT give private key to anyone including Vite Labs. 

Trading service engine will apply a unique address for each user to make the orders. It's your responsibility to provide necessary quota for the address by staking. Forgetting to stake may cause trading failure on the address.

## Base Endpoint
* 【Mainnet】: `https://api.vitex.net`

## API Authorization
ViteX has 2 category of APIs - public and private. The latter requires authorization by providing **API Key** and **API Secret**.

At the time being, you should contact marketing representative of Vite Labs to get your API key and secret (case sensitive).

Please note besides normal API parameters, 3 additional parameters `key`, `timestamp` and `signature` should be specified. 

* `key` - Your **API Key**
* `timestamp` - UNIX-style timestamp in epoch millisecond format, like 565360340430. The API call will be regarded invalid if the timestamp in request is 5,000 ms earlier or 1,000 ms later than standard time to avoid replay attack.  
* `signature` - HMAC SHA256 signature on parameter string by feeding **API Secret** as encryption key

Timestamp checking:

```
    if (timestamp < (serverTime + 1000) && (serverTime - timestamp) <= 5000) {
      // process request
    } else {
      // reject request
    }
```

## Return Value
API response is returned in JSON string. 

HTTP code:

* HTTP `200` API returned successfully
* HTTP `4XX` Wrong API request
* HTTP `5XX` Service error

JSON format:

Key | Value
------------ | ------------
code | `0` - success. Error code returned if request failed
msg | Error message
data | Return data

Example:
```javascript
{
  "code": 1,
  "msg": "Invalid symbol",
  "data": {}
}
```

Error code: 

* `1001` Too frequent request - Request frequency exceeds limit. 
* `1002` Invalid parameter - This includes invalid timestamp, wrong order price format, invalid order amount, too small order, invalid market, insufficient permission, symbol not exist
* `1003` Network - includes network jam, network failure, insufficient quota
* `1004` Other failure - includes attempting to cancel order belonging to other address, attempting to cancel order unable to cancel, order status querying exception
* `1005` Server error - Trading service error

## Trigger Limit
The counter is reset in every counting period (60s). In each period, API request would fail if trigger limit is reached.  

## Signature of Request String

* List all parameters (parameter and API key) in alphabet order;
* Generate normalized request string by concatenating parameter name and value with `=` and name-value pairs with `&`;
* Sign the request string by HMAC SHA256 signature algorithm, as the encryption key is API secret;
* Signature is case in-sensitive;
* Signature is also required to pass in API in `signature` field;
* When both request string and request body are present, put request string in ahead of request body to produce signature.

## Example

Let's take API `/api/v1/order` as example. Assume we have the following API key/secret:

API Key | API Secret
------------ | ------------
11111111 | 22222222

We place an order on market ETH-000_BTC-000 to buy 10 ETH at price 0.09 BTC. The request has the following parameters:

Key | Value
------------ | ------------
symbol | ETH-000_BTC-000
side | 0
amount | 10
price | 0.09
timestamp | 1567067137937

The ordered request string is:

`amount=10&key=11111111&price=0.09&side=0&symbol=ETH-000_BTC-000&timestamp=1567755178560`

Create signature:

    ```
    $ echo -n "amount=10&key=11111111&price=0.09&side=0&symbol=ETH-000_BTC-000&timestamp=1567755178560" | openssl dgst -sha256 -hmac "22222222"
    (stdin)= 409cf00bb97c08ae99317af26b379ac59f4cfaba9591df7738c0604a4cb68b9a
    ```

Make the API call:

    ```
    $ curl -X POST -d "amount=10&key=11111111&price=0.09&side=0&symbol=ETH-000_BTC-000&timestamp=1567755178560&signature=409cf00bb97c08ae99317af26b379ac59f4cfaba9591df7738c0604a4cb68b9a" https://api.vitex.net/api/v1/order
    ```

# ViteX Public API (deprecated)

## Data Definition

### Order Status
Code | Status | Description
------------ | ------------ | ------------
0 | Unknown | Status unknown
1 | Pending Request| Order submitted. A corresponding request transaction has been created on chain
2 | Received | Order received
3 | Open | Order unfilled
4 | Filled | Order filled
5 | Partially Filled | Order partially filled
6 | Pending Cancel | Cancel order request submitted. A corresponding request transaction has been created on chain
7 | Cancelled | Order cancelled
8 | Partially Cancelled| Order partially cancelled (the order was partially filled)
9 | Failed | Request failed
10 | Expired | Order expired

### Order Type

Code | Status | Description
------------ | ------------ | ------------
0 | Limit Order | Limit Order
1 | Market Order | Market Order (not supported yet)

### Order Side

Code | Status | Description
------------ | ------------ | ------------
0 | Buy Order | Buy
1 | Sell Order | Sell

### Time in Force

Code | Status | Description
------------ | ------------ | ------------
0 | GTC - Good till Cancel | Order valid until it is fully filled or cancelled
1 | IOC - Immediate or Cancel | Place an order and cancel the unfilled part if any (not supported yet)
2 | FOK - Fill or Kill | Place an order only if it can be fully filled (not supported yet)

## General Endpoint
### Ping
```
GET /api/v1/ping
```
Ping API server

**Parameter:**
None

**Response:**
```javascript
{
  "code": 0,
  "msg": "Success"
}
```

### Get Timestamp
```
GET /api/v1/timestamp
```
Get server timestamp

**Parameter:**
None

**Response:**
```javascript
{
  "code": 0,
  "msg": "Success",
  "data": 1565360340430
}
```

### Get Market Info
```
GET /api/v1/markets
```
**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
operator | STRING | NO | Operator ID. If present, only trading pairs belongs to the specific operator are returned
quoteCurrency | STRING | NO | Quote token symbol. If present, only trading pairs having the specific quote token are returned



**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
tradingCurrency | STRING | Trade token symbol
quoteCurrency | STRING | Quote token symbol
tradingCurrencyId | STRING | Trade token ID
quoteCurrencyId | STRING | Quote token ID
tradingCurrencyName | STRING | Trade token name
quoteCurrencyName | STRING | Quote token name
operator | STRING | Operator ID
operatorName | STRING | Operator name
operatorLogo | STRING | URL of operator logo
pricePrecision | INT | Price precision (decimals)
amountPrecision | INT | Amount precision (decimals)
minOrderSize | STRING | Minimum order size (in quote token)
operatorMakerFee | STRING | Operator fee (maker). For example, 0.001
operatorTakerFee | STRING | Operator fee (taker). For example, 0.001


```javascript
{
  "code": 0,
  "msg": "Success",
  "data": [
    {
      "symbol": "ETH-000_BTC-000",
      "tradingCurrency": "ETH-000",
      "quoteCurrency": "BTC-000",
      "tradingCurrencyId": "tti_687d8a93915393b219212c73",
      "quoteCurrencyId": "tti_b90c9baffffc9dae58d1f33f",
      "tradingCurrencyName": "Ethereum",
      "quoteCurrencyName": "Bitcoin",
      "operator": "vite_050697d3810c30816b005a03511c734c1159f50907662b046f",
      "operatorName": "Vite Labs",
      "operatorLogo": "https://token-profile-1257137467.cos.ap-hongkong.myqcloud.com/icon/e6dec7dfe46cb7f1c65342f511f0197c.png",
      "pricePrecision": 6,
      "amountPrecision": 4,
      "minOrderSize": "0.0001",
      "operatorMakerFee": "0",
      "operatorTakerFee": "0.001"
    },
    {
      //...
    }
  ]
}
```

### Get Market Info (by given trading pair)
```
GET /api/v1/market
```
**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | NO | Trading pair name. For example, "ETH-000_BTC-000"


**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
tradingCurrency | STRING | Trade token symbol
quoteCurrency | STRING | Quote token symbol
tradingCurrencyId | STRING | Trade token ID
quoteCurrencyId | STRING | Quote token ID
tradingCurrencyName | STRING | Trade token name
quoteCurrencyName | STRING | Quote token name
operator | STRING | Operator ID
operatorName | STRING | Operator name
operatorLogo | STRING | URL of operator logo
pricePrecision | INT | Price precision (decimals)
amountPrecision | INT | Amount precision (decimals)
minOrderSize | STRING | Minimum order size (in quote token)
operatorMakerFee | STRING | Operator fee (maker). For example, 0.001
operatorTakerFee | STRING | Operator fee (taker). For example, 0.001
highPrice | STRING | Highest price in 24h
lowPrice | STRING | Lowest price in 24h
lastPrice | STRING | Current price
volume | STRING | Volume in 24h (in trade token)
quoteVolume | STRING | Volume in 24h (in quote token)
bidPrice | STRING | Best bid price
askPrice | STRING | Best ask price
openBuyOrders | INT | Number of open buy orders
openSellOrders | INT | Number of open sell orders

```javascript
{
  "code": 0,
  "msg": "Success",
  "data": {
    "symbol": "ETH-000_BTC-000",
    "tradingCurrency": "ETH-000",
    "quoteCurrency": "BTC-000",
    "tradingCurrencyId": "tti_687d8a93915393b219212c73",
    "quoteCurrencyId": "tti_b90c9baffffc9dae58d1f33f",
    "tradingCurrencyName": "Ethereum",
    "quoteCurrencyName": "Bitcoin",
    "operator": "vite_050697d3810c30816b005a03511c734c1159f50907662b046f",
    "operatorName": "Vite Labs",
    "operatorLogo": "https://token-profile-1257137467.cos.ap-hongkong.myqcloud.com/icon/e6dec7dfe46cb7f1c65342f511f0197c.png",
    "pricePrecision": 6,
    "amountPrecision": 4,
    "minOrderSize": "0.0001",
    "operatorMakerFee": "0",
    "operatorTakerFee": "0.001",
    "highPrice": "0.023511",
    "lowPrice": "0.022790",
    "lastPrice": "0.023171",
    "volume": "3798.4379",
    "quoteVolume": "46.5912",
    "bidPrice": "0.023171",
    "askPrice": "0.023511",
    "openBuyOrders": 25,
    "openSellOrders": 36
  }
}
```

## Public Endpoint

### Get 24h Info

```
GET /api/v1/ticker/24hr
```

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
quoteCurrency | STRING | NO | Quote token symbol. If present, only trading pairs having the specific quote token are returned
symbol | STRING | NO | Trading pair name. For example, "ETH-000_BTC-000"

**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
tradingCurrency | STRING | Trade token symbol
quoteCurrency | STRING | Quote token symbol
tradingCurrencyId | STRING | Trade token ID
quoteCurrencyId | STRING | Quote token ID
openPrice | STRING | Open price
prevPrice | STRING | 　Previous deal price
lastPrice | STRING | Current price
priceChange | STRING | Price change. Having $priceChange=lastPrice - openPrice$
priceChangePercent | STRING | Rate of price change. Having $priceChangePercent=(lastPrice - openPrice)/openPrice$ 
highPrice | STRING | Highest price in 24h
lowPrice | STRING | Lowest price in 24h
volume | STRING | Volume in 24h (in trade token)
quoteVolume | STRING | Volume in 24h (in quote token)
pricePrecision | INT | Price precision (decimals)
amountPrecision | INT | Amount precision (decimals)

```javascript
{
  "code": 0,
  "msg": "Success",
  "data": [
    {
      "symbol": "CSTT-47E_VITE",
      "tradingCurrency": "CSTT",
      "quoteCurrency": "VITE",
      "tradingCurrencyId": "tti_b6f7019878fdfb21908a1547",
      "quoteCurrencyId": "tti_5649544520544f4b454e6e40",
      "openPrice": "1.00000000",
      "prevPrice": "0.00000000",
      "lastPrice": "1.00000000",
      "priceChange": "0.00000000",
      "priceChangePercent": 0.0,
      "highPrice": "1.00000000",
      "lowPrice": "1.00000000",
      "volume": "45336.20000000",
      "quoteVolume": "45336.20000000",
      "pricePrecision": 8,
      "amountPrecision": 8
    }
  ]
}
```


### Get Book Ticker

```
GET /api/v1/ticker/bookTicker
```

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"

**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
bidPrice | STRING | Best bid price
bidVolume | STRING | Best bid amount (in quote token)
askPrice | STRING | Best ask price
askVolume | STRING | Best ask amount (in quote token)

```javascript
{
  "code": 0,
  "msg": "Success",
  "data": {
      "symbol": "CSTT-47E_VITE",
      "bidPrice": "1.00000000",
      "bidVolume": "45336.20000000"
      "askPrice": "1.00000000",
      "askVolume": "45336.20000000"
    }
}
```

### Get Market Depth
```
GET /api/v1/depth
```

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
limit | INT | NO | Number of item returned. Valid value can be [5, 10, 20, 50, 100]. Default is 10. Max is 100
precision | INT | NO | Price aggregation precision (decimals). If not present, order price will not be aggregated


**Response:**

Name | Type | Description
------------ | ------------ | ------------
timestamp | LONG | Timestamp
asks | ARRAY | List of sell orders (in ascending order). Format:[price, amount]
bids | ARRAY | List of buy orders (in descending order). Format:[price, amount]

```javascript
{
  "code": 0,
  "msg": "Success",   
  "data": {
    "timestamp": 1565360340430,
    "asks": [
      ["0.023511", "0.1000"],
      ["0.024000", "2.0000"],
      ["0.025000", "14.0000"],
      ["0.026000", "20.0000"],
      ["0.027000", "100.0000"]      
    ],  
    "bids": [
      ["0.023171", "1.2300"],
      ["0.023000", "3.7588"],
      ["0.022000", "5.0000"],
      ["0.021000", "10.0000"],
      ["0.020000", "50.0000"]
    ]
  }
}
```

### Get Trading Records
```
GET /api/v1/trades
```
Get latest trading records

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
limit | INT | NO | Number of record returned. Max is 500

**Response:**

Name | Type | Description
------------ | ------------ | ------------
timestamp | LONG | Timestamp
price | STRING | Price
amount | STRING | Amount (in trade token)
side | INT | Buy - `0`, Sell - `1`

```javascript
{
  "code": 0,
  "msg": "Success",   
  "data": [
    {
      "timestamp": 1565360340430,
      "price": "0.023171",
      "amount": "0.0123",
      "side": 0
    }, {
        // ...
    }
  ]
}
```

### Get K-lines

```
GET /api/v1/klines
```
Get K-lines

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
limit | INT | NO | Number of record returned. Max is 500
interval|STRING|YES| Interval. Valid value can be [`minute`、`hour`、`day`、`minute30`、`hour6`、`hour12`、`week`]
startTime|LONG|NO| Start time
endTime|LONG|NO| End time

**Response:**

Name | Type | Description
------------ | ------------ | ------------
t | LONG | Timestamp
c | STRING | Close price
p | STRING | Open price
h | STRING | Highest price
l | STRING | Lowest price
v | STRING | Trading volume (in trade token)

```javascript
{
  "code": 0,
  "msg": "Success",
  "data": {
    "t": [ 1554207060 ],
    "c": [ 1.0 ],
    "p": [ 1.0 ],
    "h": [ 1.0 ],
    "l": [ 1.0 ],
    "v": [ 12970.8 ]
  }
}
```


## Private Endpoint

### Place Order (test)
```
POST /api/v1/order/test
```
This API can be used to verify signature

**Quota consumption:**
0 UT

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
amount | STRING | YES | Order amount (in trade token)
price | STRING | YES | Order price
side | INT | YES | Buy - `0`, Sell - `1`
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature

**Response:**
None

```javascript
{
  "code": 0,
  "msg": "Success",   
  "data": null
}
```


### Place Order
```
POST /api/v1/order
```

**Quota consumption:**
1 UT

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
amount | STRING | YES | Order amount (in trade token)
price | STRING | YES | Order price
side | INT | YES | Buy - `0`, Sell - `1`
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature

**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
orderId | STRING | Order ID

```javascript
{
  "code": 0,
  "msg": "Success",   
  "data": {
    "symbol": "ETH-000_BTC-000",
    "orderId": "f848eb32ea44d810b0f35c6c44ccb6795f7d31503b15bdd8171be2809bee4a25"
  }
}
```

### Query Order
```
GET /api/v1/order
```
Get detailed order info

**Quota consumption:**
0 UT

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
orderId | STRING | YES | Order ID
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature

**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
orderId | STRING | Order ID
status | INT | Order status. See [Order Status Definition](#order-status)
side | INT | Buy - `0`, Sell - `1`
price | STRING | Order price (in quote token)
amount | STRING | Order amount (in trade token)
filledAmount | STRING | Filled amount (in trade token)
filledValue | STRING | Filled value (in quote token)
fee | STRING | Trading fee (in quote token)
created | LONG | Order create time
updated | LONG | Order last update time
timeInForce | INT | See [Time in Force](#time-in-force)
type | INT | Order type. See [Order Type Definition](#order-type)


```javascript
{
  "code": 0,
  "msg": "Success",   
  "data": {
    "symbol": "ETH-000_BTC-000",
    "orderId": "f848eb32ea44d810b0f35c6c44ccb6795f7d31503b15bdd8171be2809bee4a25",
    "status": 3,
    "side": 0,
    "price": "0.020000",
    "amount": "1.1100",
    "filledAmount": "0.1000",
    "filledValue": "0.002000",
    "fee": "0.000002",
    "created": 1565360340430,
    "updated": 1565360340430,
    "timeInForce": 0,
    "type": 0
  }
}
```

### Cancel Order
```
DELETE /api/v1/order
```
Cancel order by given order ID. Please note this API initiates a cancel request, it doesn't guarantee the order will be cancelled eventually

**Quota consumption:**
1 UT

**Parameter:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
orderId | STRING | YES | Order ID
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature

**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
orderId | STRING | Order ID
cancelRequest | STRING | Cancel request ID

```javascript
{
  "code": 0,
  "msg", "Success",   
  "data": {
    "symbol": "ETH-000_BTC-000",
    "orderId": "f848eb32ea44d810b0f35c6c44ccb6795f7d31503b15bdd8171be2809bee4a25",
    "cancelRequest": "fd7d9ecd2471dbc458ac5d42024e69d6d0a033b9935a3c0524e718adb30b27db"
  }
}
```

### Cancel Orders
```
DELETE /api/v1/orders
```

Cancel all orders under given trading pair. Please note this API initiates a number of cancel requests, it doesn't guarantee the orders will be cancelled eventually

**Quota consumption:**
N UT(N=Order Number)

**Parameters:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature

**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
orderId | STRING | Order ID
cancelRequest | STRING | Cancel request ID

```javascript
{
  "code": 0,
  "msg", "Success",   
  "data": [
    {
      "symbol": "ETH-000_BTC-000",
      "orderId": "f848eb32ea44d810b0f35c6c44ccb6795f7d31503b15bdd8171be2809bee4a25",
      "cancelRequest": "fd7d9ecd2471dbc458ac5d42024e69d6d0a033b9935a3c0524e718adb30b27db"
    }
  ]
}
```

### Query orders
```
GET /api/v1/orders
```
Get historical order info

**Quota consumption:**
0 UT

**Parameters:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
symbol | STRING | YES | Trading pair name. For example, "ETH-000_BTC-000"
startTime |LONG | NO | Start time
endTime |LONG | NO | End time
side | INT | NO | Buy - `0`, Sell - `1`
status | INT | NO | Order status. See [Order Status Definition](#order-status)
limit | INT | NO | Number of record returned. Max is 500
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature


**Response:**

Name | Type | Description
------------ | ------------ | ------------
symbol | STRING | Trading pair name
orderId | STRING | Order ID
status | INT | Order status. See [Order Status Definition](#order-status)
side | INT | Buy - `0`, Sell - `1`
price | STRING | Order price (in quote token)
amount | STRING | Order amount (in trade token)
filledAmount | STRING | Filled amount (in trade token)
filledValue | STRING | Filled value (in quote token)
fee | STRING | Trading fee (in quote token)
created | LONG | Order create time
updated | LONG | Order last update time
timeInForce | INT | See [Time in Force](#time-in-force)
type | INT | Order type. See [Order Type Definition](#order-type)


```javascript
{
  "code": 0,
  "msg": "Success",
  "data": [
    {
      "symbol": "ETH-000_BTC-000",
      "orderId": "f848eb32ea44d810b0f35c6c44ccb6795f7d31503b15bdd8171be2809bee4a25",
      "status": 3,
      "side": 0,
      "price": "0.020000",
      "amount": "1.1100",
      "filledAmount": "0.1000",
      "filledValue": "0.002000",
      "fee": "0.000002",
      "created": 1565360340430,
      "updated": 1565360340430,
      "timeInForce": 0,
      "type": 0
    }
  ]
}
```

### Get Balance in Exchange
```
GET /api/v1/account
```
Get account's exchange balance

**Quota consumption:**
0 UT

**Parameters:**

Name | Type | Is Required? | Description
------------ | ------------ | ------------ | ------------
timestamp | LONG | YES | Timestamp (client side)
key | STRING | YES | API Key
signature | STRING | YES | HMAC SHA256 signature

**Response:**

Name | Type | Description
------------ | ------------ | ------------
available | STRING | Balance available
locked | STRING | Balance locked (by order)


```javascript
{
    "code": 0,
    "msg": "Success",
    "data": {
        "VX": {
            "available": "0.00000000",
            "locked": "0.00000000"
        },
        "VCP": {
            "available": "373437.00000000",
            "locked": "0.00000000"
        },
        "BTC-000": {
            "available": "0.02597393",
            "locked": "0.13721639"
        },
        "USDT-000": {
            "available": "162.58284100",
            "locked": "170.40459600"
        },
        "GRIN-000": {
            "available": "0.00000000",
            "locked": "0.00000000"
        },
        "VITE": {
            "available": "30047.62090072",
            "locked": "691284.75633290"
        },
        "ETH-000": {
            "available": "1.79366977",
            "locked": "7.93630000"
        },
        "ITDC-000": {
            "available": "0.00000000",
            "locked": "7186.00370000"
        }
    }
}
```
