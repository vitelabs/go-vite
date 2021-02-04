---
sidebarDepth: 4
---

# DexTrade
:::tip Maintainer
[weichaolee](https://github.com/weichaolee)
:::

## ABI
The built-in dex trade contract. Contract address is `vite_00000000000000000000000000000000000000079710f19dc7`

### CancelOrder
ABI definition
```
{
  "type": "function",
  "name": "DexTradeCancelOrder",
  "inputs": [
    {
      "name": "orderId",
      "type": "bytes"
    }
  ]
}
```
Inputs

| Input item | Name | Data type | source | comment |
|:------------:|:-----------:|:-----:|:-----:|:-----:|
| AccountAddress| new order owner |  Address |sendBlock| |
| orderId| - |  bytes |ABI| |

## RPC interface
**Supported protocol:**

|  JSON-RPC 2.0  | HTTP | IPC |Publish–subscribe |Websocket |
|:------------:|:-----------:|:-----:|:-----:|:-----:|
| &#x2713;|  &#x2713; |  &#x2713; | future version| &#x2713; |

### dextrade_getOrderById
query order detail by order Id

- **Parameters**: 

  * `orderId`: base64 encoded orderId
  
- **Returns**: 
  - `Order`:order detail

- **Example**:

::: demo

```json tab:Request
{
   "jsonrpc":"2.0",
   "id":1,
   "method":"dextrade_getOrderById",
   "params": [
         "AAAIAQAAAAAeAAAAAAAAXT6fSQAJVQ=="
         ]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "Id": "AAAIAQAAAAAeAAAAAAAAXT6fSQAJVQ==",
        "Address": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
        "MarketId": 8,
        "Side": true,
        "Type": 0,
        "Price": "30",
        "TakerFeeRate": 90,
        "MakerFeeRate": 90,
        "TakerBrokerFeeRate": 0,
        "MakerBrokerFeeRate": 0,
        "Quantity": "400000000000000000000",
        "Amount": "12000000000000000000000",
        "Status": 0,
        "Timestamp": 1564385097
    }
}
```
:::

### dextrade_getOrdersFromMarket
query open orders from one trade pair, return orders will list by sequence which they will be matched

- **Parameters**: 

  * `tradeToken`: trade token of trade pair
  * `quoteToken`: quote token of trade pair
  * `side`: buy/sell trade token for order, false buy, true sell
  * `begin`: begin index, start from 0
  * `end`: end index
  
- **Returns**: 
  - `[]Order`:open order list

- **Example**:

::: demo

```json tab:Request
{
   "jsonrpc":"2.0",
   "id":1,
   "method":"dextrade_getOrdersFromMarket",
   "params": [
        "tti_2736f320d7ed1c2871af1d9d",
        "tti_5649544520544f4b454e6e40",
        true,
        0,
        10
        ]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "orders": [
            {
                "Id": "AAAIAQAAAAAeAAAAAAAAXT6fSQAJVQ==",
                "Address": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "MarketId": 8,
                "Side": true,
                "Type": 0,
                "Price": "30",
                "TakerFeeRate": 90,
                "MakerFeeRate": 90,
                "TakerBrokerFeeRate": 0,
                "MakerBrokerFeeRate": 0,
                "Quantity": "400000000000000000000",
                "Amount": "12000000000000000000000",
                "Status": 0,
                "Timestamp": 1564385097
            },
            {
                "Id": "AAAIAejUpQ//6NSk6PAAXT6fSQAIrw==",
                "Address": "vite_553462bca137bac29f440e9af4ab2e2c1bb82493e41d2bc8b2",
                "MarketId": 8,
                "Side": true,
                "Type": 0,
                "Price": "40.5",
                "TakerFeeRate": 100,
                "MakerFeeRate": 100,
                "TakerBrokerFeeRate": 0,
                "MakerBrokerFeeRate": 0,
                "Quantity": "100000000000",
                "Amount": "99999999999999999999000",
                "Status": 0,
                "Timestamp": 1564385097
            }
        ],
        "size": 2
    }
}
```
:::