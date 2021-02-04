---
demoUrl: "https://vitex.vite.net/test"
---

# Data Query API

## HTTP API

### Network

* **Mainnet**: https://vitex.vite.net/

* **Test**: https://vitex.vite.net/test

### `/api/v1/limit`

Get minimum order quantity for all 4 markets

* **Method**: `GET`

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`Limit`|
  |1|error_msg|null|

* **Example**

  :::demo
  ```json test:Run url: /api/v1/limit method: GET
  {}
  ```
  :::

### `/api/v1/tokens`

Get tokens list

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |category|query|Default `all`. Allowed value: [`quote`,`all`]|no|string|
  |tokenSymbolLike|query|Token symbol. Example: `ETH`|no|string|
  |offset|query|Starting with `0`. Default `0`|no|integer|
  |limit|query|Default `500`. Max `500`|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`Token`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": [
      {
        "tokenId": "tti_4e88a475c675971dab7ec917",
        "name": "Bitcoin",
        "symbol": "BTC",
        "originalSymbol": "BTC",
        "totalSupply": "2100000000000000",
        "owner": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a"
      }
    ]
  }
  ```
  
  ```json test:Run url: /api/v1/tokens method: GET
  {}
  ```
  :::

### `/api/v1/token/detail`

Get token information

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |tokenSymbol|query|Token symbol. Example: `VITE`|no|string|
  |tokenId|query|Token Id. Example: `tti_5649544520544f4b454e6e40`|no|string|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`TokenDetail`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "tokenId": "tti_4e88a475c675971dab7ec917",
      "name": "Bitcoin",
      "symbol": "BTC",
      "originalSymbol": "BTC",
      "totalSupply": "2100000000000000",
      "publisher": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
      "tokenDecimals": 8,
      "tokenAccuracy": "0.00000001",
      "publisherDate": null,
      "reissue": 2,
      "urlIcon": null,
      "gateway": null,
      "website": null,
      "links": null,
      "overview": null
    }
  }
  ```
  
  ```json test:Run url: /api/v1/token/detail?tokenId=tti_5649544520544f4b454e6e40 method: GET
  {}
  ```
  :::
  
### `/api/v1/token/mapped`

Get a list of tokens have opened trading pair(s)

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |quoteTokenSymbol|query|Token symbol. Example: `VITE` |yes|string|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`TokenMapping`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": [
      {
        "tokenId": "tti_c2695839043cf966f370ac84",
        "symbol": "VCP"
      }
    ]
  }
  ```
  
  ```json test:Run url: /api/v1/token/mapped?quoteTokenSymbol=VITE method: GET
  {}
  ```
  :::
  
### `/api/v1/token/unmapped`

Get a list of tokens haven't opened any trading pair

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |quoteTokenSymbol|query|Token symbol. Example: `VITE`|yes|string|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`TokenMapping`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": [
      {
        "tokenId": "tti_2736f320d7ed1c2871af1d9d",
        "symbol": "VTT"
      }
    ]
  }
  ```
  
  ```json test:Run url: /api/v1/token/unmapped?quoteTokenSymbol=VITE method: GET
  {}
  ```
  :::
  
### `/api/v1/markets`

Get market pairs

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |offset|query|Starting with `0`. Default `0`|no|integer|
  |limit|query|Default `500`. Max `500`|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`Market`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": [
      {
        "symbol": "BTC-A_USDT",
        "tradeTokenSymbol": "BTC-A",
        "quoteTokenSymbol": "USDT",
        "tradeToken": "tti_322862b3f8edae3b02b110b1",
        "quoteToken": "tti_973afc9ffd18c4679de42e93",
        "pricePrecision": 8,
        "quantityPrecision": 8
      }
    ]
  }
  ```
  
  ```json test:Test url: /api/v1/markets method: GET
  {}
  ```
  :::
  

### `/api/v1/order`

Get an order for a given address and order id

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |address|query|the buyer/seller address|yes|string|
  |orderId|query|the order id|yes|string|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`Order`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
        "orderId": "quEOx/ai3o7Xyv9em+qJIbJu7pM=",
        "symbol": "VCP-4_VITE",
        "tradeTokenSymbol": "VCP.test",
        "quoteTokenSymbol": "VITE",
        "tradeToken": "tti_c2695839043cf966f370ac84",
        "quoteToken": "tti_5649544520544f4b454e6e40",
        "side": 1,
        "price": "6.00000000",
        "quantity": "1.00000000",
        "amount": "6.00000000",
        "executedQuantity": "0.00000000",
        "executedAmount": "0.00000000",
        "executedPercent": "0.00000000",
        "executedAvgPrice": "0.00000000",
        "fee": "0.00000000",
        "status": 1,
        "type": 0,
        "createTime": 1554722699
    }
  }
  ```
  
  ```json test:Run url: /api/v1/order method: GET
  {}
  ```
  :::  

### `/api/v1/orders/open`

Get open orders for a given address

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |address|query|the buyer/seller address|yes|string|
  |quoteTokenSymbol|query|Quote token symbol|no|string|
  |tradeTokenSymbol|query|Trade token symbol|no|string|
  |offset|query|Starting with `0`. Default `0`|no|integer|
  |limit|query|Default`30`. Max `100`|no|integer|
  |total|query|Total number required. `0` for not required and `1` for required. Default is not required and will return total=-1 in response|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`OrderList`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "order": [
        {
          "orderId": "quEOx/ai3o7Xyv9em+qJIbJu7pM=",
          "symbol": "VCP-4_VITE",
          "tradeTokenSymbol": "VCP.test",
          "quoteTokenSymbol": "VITE",
          "tradeToken": "tti_c2695839043cf966f370ac84",
          "quoteToken": "tti_5649544520544f4b454e6e40",
          "side": 1,
          "price": "6.00000000",
          "quantity": "1.00000000",
          "amount": "6.00000000",
          "executedQuantity": "0.00000000",
          "executedAmount": "0.00000000",
          "executedPercent": "0.00000000",
          "executedAvgPrice": "0.00000000",
          "fee": "0.00000000",
          "status": 1,
          "type": 0,
          "createTime": 1554722699
        }
      ],
      "total": 13
    }
  }
  ```
  
  ```json test:Run url: /api/v1/orders/open?address=vite_ff38174de69ddc63b2e05402e5c67c356d7d17e819a0ffadee method: GET
  {}
  ```
  :::

### `/api/v1/orders`

Get orders list for a given address 

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |address|query|The buyer/seller address|yes|string|
  |quoteTokenSymbol|query|Symbol of quote token|no|string|
  |tradeTokenSymbol|query|Symbol of trade token|no|string|
  |startTime|query|Start time in Seconds|no|long|
  |endTime|query|End time in Seconds|no|long|
  |side|query|Order side. Allowed value: [`0`:buy, `1`:sell]|no|integer|
  |status|query|Order status list. Allowed value: [`1`:open, `2`:closed, `3`:canceled, `4`:failed]|no|integer|
  |offset|query|Starting with `0`. Default `0`|no|integer|
  |limit|query|Default `30`. Max value `100`|no|integer|
  |total|query|Total number required. `0` for not required and `1` for required. Default is not required and will return total=-1 in response|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`OrderList`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "order": [
        {
          "orderId": "ZDFDKEcwCs2BVs8fE8vULzb10/g=",
          "symbol": "CSTT-47E_VITE",
          "tradeTokenSymbol": "CSTT.test",
          "quoteTokenSymbol": "VITE",
          "tradeToken": "tti_b6f7019878fdfb21908a1547",
          "quoteToken": "tti_5649544520544f4b454e6e40",
          "side": 0,
          "price": "1.00000000",
          "quantity": "118222.20000000",
          "amount": "118222.20000000",
          "executedQuantity": "0.00000000",
          "executedAmount": "0.00000000",
          "executedPercent": "0.00000000",
          "executedAvgPrice": "0.00000000",
          "fee": "0.00000000",
          "status": 4,
          "type": 0,
          "createTime": 1554702092
        }
      ],
      "total": 1215
    }
  }
  ```
  
  ```json test:Run url: /api/v1/orders?address=vite_ff38174de69ddc63b2e05402e5c67c356d7d17e819a0ffadee method: GET
  {}
  ```
  :::

### `/api/v1/ticker/24hr`

Get 24-hour price change statistics for a given market pair

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |symbols|query|Market pair split by `,`. Example: `ABC-000_VITE, ABC-001_VITE`|no|string|
  |quoteTokenSymbol|query|Symbol of quote token|no|string|
  |quoteTokenCategory|query|The category of quote token. Allowed value: [`VITE`,`ETH`,`BTC`,`USDT`]|no|string|
* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`TickerStatistics`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": [
      {
        "symbol": "CSTT-47E_VITE",
        "tradeTokenSymbol": "CSTT",
        "quoteTokenSymbol": "VITE",
        "tradeToken": "tti_b6f7019878fdfb21908a1547",
        "quoteToken": "tti_5649544520544f4b454e6e40",
        "openPrice": "1.00000000",
        "prevClosePrice": "0.00000000",
        "closePrice": "1.00000000",
        "priceChange": "0.00000000",
        "priceChangePercent": 0.0,
        "highPrice": "1.00000000",
        "lowPrice": "1.00000000",
        "quantity": "45336.20000000",
        "amount": "45336.20000000",
        "pricePrecision": 8,
        "quantityPrecision": 8
      }
    ]
  }
  ```
  
  ```json test:Run url: /api/v1/ticker/24hr?quoteTokenSymbol=VITE method: GET
  {}
  ```
  :::


### `/api/v1/ticker/bookTicker`

Get the best bid/ask price for a given market pair

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |symbol|query|Market pair. Example: `ABC-000_VITE`|yes|string|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`BookTicker`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
        "symbol": "CSTT-47E_VITE",
        "bidPrice": "1.00000000",
        "bidQuantity": "45336.20000000",
        "askPrice": "1.00000000",
        "askQuantity": "45336.20000000"
      }
  }
  ```
  
  ```json test:Run url: /api/v1/ticker/bookTicker?symbol=BTC-000_VITE-000 method: GET
  {}
  ```
  :::
  
### `/api/v1/trades`

Get a list of historical trades for a given market pair

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |symbol|query|Market pair. Example: `BTC-000_VITE`|yes|string|
  |orderId|query|Order id|no|string|
  |startTime|query|Start time in Seconds|no|long|
  |endTime|query|End time in Seconds|no|long|
  |side|query|Order side. Allowed value: [`0`:buy, `1`:sell].|no|integer|
  |offset|query|Starting with `0`. Default `0`.|no|integer|
  |limit|query|Default `30`. Max `100`.|no|integer|
  |total|query|Total number required. `0` for not required and `1` for required. Default is not required and will return total=-1 in response|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`TradeList`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "trade": [
        {
          "tradeId": "4EOgUqsCyZ73O4+A2gZuK9RfOXs=",
          "symbol": "VTT-F_ETH",
          "tradeTokenSymbol": "VTT",
          "quoteTokenSymbol": "ETH",
          "tradeToken": "tti_2736f320d7ed1c2871af1d9d",
          "quoteToken": "tti_06822f8d096ecdf9356b666c",
          "price": "0.10000000",
          "quantity": "1.00000000",
          "amount": "0.10000000",
          "time": 1554793244,
          "side": 0,
          "buyerOrderId": "DqpoIXTCT+4s1rMBFVCoWY9iDys=",
          "sellerOrderId": "FB4eiknqAQpIEOYi+HgamZOj/ac=",
          "buyFee": "0.00010000",
          "sellFee": "0.00010000",
          "blockHeight": 2806
        }
      ],
      "total": 1
    }
  }
  ```
  
  ```json test:Run url: /api/v1/trades?symbol=BTC-000_VITE-000 method: GET
  {}
  ```
  :::
  
### `/api/v1/depth`

Get the order book depth data for a given market pair

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |symbol|query|Market pair. Example: `CSTT-47E_VITE`|yes|string|
  |limit|query|Default `100`. Max `100`|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`MarketDepth`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "asks": [
        {
          "price": "1.00000000",
          "quantity": "111233.50000000",
          "amount": "111233.50000000"
        }    
      ],
      "bids": [
        {
          "price": "2.00000000",
          "quantity": "111233.50000000",
          "amount": "111233.50000000"
        }
      ]
    }
  }
  ```
  
  ```json test:Run url: /api/v1/depth?symbol=BTC-000_VITE-000 method: GET
  {}
  ```
  :::

### `/api/v1/klines`

Get kline bars for a given market pair

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |symbol|query|Market pair. Example: `CSTT-47E_VITE`|yes|string|
  |interval|query|Interval. Allowed value: [`minute`、`hour`、`day`、`minute30`、`hour6`、`hour12`、`week`]|yes|string|
  |limit|query|Default `500`. Max `1500`|no|integer|
  |startTime|query|Start time in Seconds|no|integer|
  |endTime|query|End time in Seconds.|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`MarketKline`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "t": [
        1554207060
      ],
      "c": [
        1.0
      ],
      "p": [
        1.0
      ],
      "h": [
        1.0
      ],
      "l": [
        1.0
      ],
      "v": [
        12970.8
      ]
    }
  }
  ```
  
  ```json test:Run url: /api/v1/klines?symbol=BTC-000_VITE-000&interval=minute method: GET
  {}
  ```
  :::
  
### `/api/v1/deposit-withdraw`

Get historical deposit/withdraw records for a given address

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |address|query|The buyer/seller address|yes|string|
  |tokenId|query|Token id|yes|string|
  |offset|query|Starting with `0`. Default `0`|no|integer|
  |limit|query|Default `100`. Max `100`|no|integer|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`DepositWithdrawList`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": {
      "record": [
        {
          "time": 1555057049,
          "tokenSymbol": "VITE",
          "amount": "1000000.00000000",
          "type": 1
        }
      ],
      "total": 16
    }
  }
  ```
  
  ```json test:Run url: /api/v1/deposit-withdraw?address=vite_ff38174de69ddc63b2e05402e5c67c356d7d17e819a0ffadee&tokenId=tti_5649544520544f4b454e6e40 method: GET
  {}
  ```
  :::
  
### `/api/v1/exchange-rate`

Get cryptocurrency rates

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|
  |tokenSymbols|query|Token symbols split by `,`. Example: `VITE, ETH`|no|string|
  |tokenIds|query|Token ids split by `,`. Example: `tti_5649544520544f4b454e6e40,tti_5649544520544f4b454e6e40`|no|string|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|[`ExchangeRate`]|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": [
      {
        "tokenId": "tti_5649544520544f4b454e6e40",
        "tokenSymbol": "VITE",
        "usdRate": 0.03,
        "cnyRate": 0.16
      }
    ]
  }
  ```
  
  ```json test:Run url: /api/v1/exchange-rate?tokenIds=tti_5649544520544f4b454e6e40 method: GET
  {}
  ```
  :::  

### `/api/v1/usd-cny`

Get currency exchange rate of USD/CNY

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`double`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": 6.849
  }
  ```
  
  ```json test:Run url: /api/v1/usd-cny method: GET
  {}
  ```
  ::: 

### `/api/v1/time`

Get the current time in milliseconds according to the HTTP service

* **Method**: `GET` 

* **Parameters**

  |Name|Located In|Description|Required|Schema|
  |:--|:--|:---|:---|:--:|

* **Responses**

  |code|msg|data|
  |:--|:--|:--:|
  |0|success|`long`|
  |1|error_msg|null|

* **Example**

  :::demo
  
  ```json tab:Response
  {
    "code": 0,
    "msg": "ok",
    "data": 1559033445000
  }
  ```
  
  ```json test:Run url: /api/v1/time method: GET
  {}
  ```
  ::: 

## WebSocket API

### Network
* **Pre-mainnet**: https://vitex.vite.net/websocket

* **Testnet**: https://vitex.vite.net/test/websocket

* op_type: The `ping` heartbeat message needs to be sent at least once per minute. If the interval exceeds 1 minute, the registered event will expire.


### Protocol Model
```
syntax = "proto3";

package protocol;

option java_package = "org.vite.data.dex.bean.protocol";
option java_outer_classname = "DexProto";

message DexProtocol {
    string client_id = 1; // Identify a single client
    string topics = 2; // See below
    string op_type = 3; // sub,un_sub,ping,pong,push
    bytes message = 4; // See proto data
    int32 error_code = 5; // Error code. 0:normal, 1:illegal_client_id, 2:illegal_event_key, 3:illegal_op_type, 5:visit limit
}

```
### Definition of op_type  
* sub: subscription
* un_sub: un-subscription
* ping: heartbeat message sent every 10 seconds to validate client_id
* pong: server-side acknowledgement
* push: push data to client


### Definition of Topic

Multiple topics can be subscribed to by `topic_1, topic2, ...`

|Topic|Description| Message Model|
|:--|:--|:--:|
|`order.$address`|Order update| See `OrderProto`|
|`market.$symbol.depth`|Depth data update| See `DepthListProto`|
|`market.$symbol.trade`|Trade data update| See `TradeListProto`|
|`market.$symbol.tickers`|Market pair statistics update|See `TickerStatisticsProto`|
|`market.quoteToken.$symbol.tickers`|Quote token statistics update|See `TickerStatisticsProto`|
|`market.quoteTokenCategory.VITE.tickers`|Quote token category statistics update|See `TickerStatisticsProto`|
|`market.quoteTokenCategory.ETH.tickers`|Quote token category statistics update|See `TickerStatisticsProto`|
|`market.quoteTokenCategory.USDT.tickers`|Quote token category statistics update|See `TickerStatisticsProto`|
|`market.quoteTokenCategory.BTC.tickers`|Quote token category statistics update|See `TickerStatisticsProto`|
|`market.$symbol.kline.minute`|1-minute kline update|See `KlineProto`|
|`market.$symbol.kline.minute30`|30-minute kline update|See `KlineProto`|
|`market.$symbol.kline.hour`|1-hour kline update|See `KlineProto`|
|`market.$symbol.kline.day`|1-day kline update|See `KlineProto`|
|`market.$symbol.kline.week`|1-week kline update|See `KlineProto`|
|`market.$symbol.kline.hour6`|6-hour kline update|See `KlineProto`|
|`market.$symbol.kline.hour12`|12-hour kline update|See `KlineProto`|


### 5. Message Model
```
syntax = "proto3";
option java_package = "org.vite.data.dex.bean.proto";
option java_outer_classname = "DexPushMessage";


message TickerStatisticsProto {

    //symbol
    string symbol = 1;
    //trade token symbol
    string tradeTokenSymbol = 2;
    //quote token symbol
    string quoteTokenSymbol = 3;
    //trade tokenId
    string tradeToken = 4;
    //quote tokenId
    string quoteToken = 5;
    //opening price
    string openPrice = 6;
    //previous closing price
    string prevClosePrice = 7;
    //closing price
    string closePrice = 8;
    //price change
    string priceChange = 9;
    //price change percentage
    string priceChangePercent = 10;
    //highest price
    string highPrice = 11;
    //lowest price
    string lowPrice = 12;
    //trading volumn
    string quantity = 13;
    //turnover
    string amount = 14;
    //price precision
    int32 pricePrecision = 15;
    //quantity precision
    int32 quantityPrecision = 16;
}


message TradeListProto {
    repeated TradeProto trade = 1;
}

message TradeProto {

    string tradeId = 1;
    //symbol
    string symbol = 2;
    //trade token symbol
    string tradeTokenSymbol = 3;
    //quote token symbol
    string quoteTokenSymbol = 4;
    //trade tokenId
    string tradeToken = 5;
    //quote tokenId
    string quoteToken = 6;
    //price
    string price = 7;
    //trading volumn
    string quantity = 8;
    //turnover
    string amount = 9;
    //time
    int64 time = 10;
    //side
    int32 side = 11;
    //buy order Id
    string buyerOrderId = 12;
    //sell order Id
    string sellerOrderId = 13;
    //buyer fee
    string buyFee = 14;
    //seller fee
    string sellFee = 15;
    //block height
    int64 blockHeight = 16;
}

message KlineProto {

    int64 t = 1;

    double c = 2;

    double o = 3;

    double h = 4;

    double l = 5;

    double v = 6;
}

message OrderProto {

    //order ID
    string orderId = 1;
    //symbol
    string symbol = 2;
    //trade token symbol
    string tradeTokenSymbol = 3;
    //quote token symbol
    string quoteTokenSymbol = 4;
    //trade tokenId
    string tradeToken = 5;
    //quote tokenId
    string quoteToken = 6;
    //side
    int32 side = 7;
    //price
    string price = 8;
    //order quantity
    string quantity = 9;
    //order amount
    string amount = 10;
    //filled quantity
    string executedQuantity = 11;
    //filled amount
    string executedAmount = 12;
    //turnover rate
    string executedPercent = 13;
    //average price
    string executedAvgPrice = 14;
    //trading fee
    string fee = 15;
    //order status
    int32 status = 16;
    //order type
    int32 type = 17;
    //creation time
    int64 createTime = 18;
    //address
    string address = 19;
}

message DepthListProto {

    repeated DepthProto asks = 1;

    repeated DepthProto bids = 2;
}

message DepthProto {
    //price
    string price = 1;
    //quantity
    string quantity = 2;
    //amount
    string amount = 3;
}
```
