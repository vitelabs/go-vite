---
order: 2
---

# ViteX WebSocket API

## Base Endpoint
* **[MainNet]** `wss://api.vitex.net/v2/ws`

## Quick Start
```shell
npm install -g wscat
wscat -c wss://api.vitex.net/v2/ws

> {"command":"ping"}

< {"code":0,"event":"pong","timestamp":1635682990916}

> { "command": "sub", "params": ["market.VITE_BTC-000.trade"]}

< {"code":0,"event":"sub","topic":"[market.VITE_BTC-000.trade]"}
< {"code":0,"data":[{"a":"0.00018200","bf":"0.00000045","bh":48524712,"bid":"00000401000000000000001bc56000617e877c00001a","id":"6473c6cc1ebc0031053963a4f73c0daf245cde9c","p":"0.00000182","q":"100.0","qid":"tti_b90c9baffffc9dae58d1f33f","qs":"BTC-000","s":"VITE_BTC-000","sf":"0.00000045","sid":"00000400ffffffffffffffe43a9f00617e645f000012","side":1,"t":1635682226,"tid":"tti_5649544520544f4b454e6e40","ts":"VITE"}],"event":"push","timestamp":1635682227087,"topic":"market.VITE_BTC-000.trade"}
```

## Message Patterns

### Request Messages
```json
{
    "command":"sub",  // command
    "params":["market.VX_VITE.depth"]  // params
}
```

### Response Messages
```json
  {
    "code": 0,  // error code
    "event": "push",  // event
    "timestamp": 1635659587885,  // time stamp
    "topic": "market.VX_VITE.depth",  // topics 
    "data": {...}  // data payload
  }
```

## Commands and Events  
* **sub**: subscribe a topic
* **un_sub**: un-subscribe a topic
* **ping**: keep-alive heartbeat message
* **pong**: keep-alive heartbeat acknowledgement
* **push**: data pushed to the client

## Keep Alive
:::demo

```json tab:Subscribe
  {
    "command":"ping"
  }
```

```json tab:Response
  {
    "code": 0,
    "event": "pong",
    "timestamp": 1635659481971
  }
```
:::

:::tip Important
To keep session alive, `ping` heartbeat messages should be sent in every 10 seconds, at most no longer than 1 minute. When heartbeats are sent longer than 1 minute, the client is no more regarded as alive and registered subscriptions will be cleaned up.
:::

## Subscribe
:::demo

```json tab:Subscribe
  {
    "command":"sub",   // command
    "params":[...]  // topics
  }
```

```json tab:Response
  {
    "code": 0,
    "event": "sub",
    "topic": "[...]"
  }
```
:::

### Topics

|Topic|Description| Message |
|:--|:--|:--:|
|`order.$address`|Order update|`Order`|
|`market.$symbol.depth`|Depth data update|`Depth`|
|`market.$symbol.trade`|Trade data update|`Trade`|
|`market.$symbol.tickers`|Market pair statistics update|`TickerStatistics`|
|`market.quoteToken.$symbol.tickers`|Quote token statistics update|`TickerStatistics`|
|`market.quoteTokenCategory.VITE.tickers`|Quote token category statistics update|`TickerStatistics`|
|`market.quoteTokenCategory.ETH.tickers`|Quote token category statistics update|`TickerStatistics`|
|`market.quoteTokenCategory.USDT.tickers`|Quote token category statistics update|`TickerStatistics`|
|`market.quoteTokenCategory.BTC.tickers`|Quote token category statistics update|`TickerStatistics`|
|`market.$symbol.kline.minute`|1-minute kline update|`Kline`|
|`market.$symbol.kline.minute30`|30-minute kline update|`Kline`|
|`market.$symbol.kline.hour`|1-hour kline update|`Kline`|
|`market.$symbol.kline.day`|1-day kline update|`Kline`|
|`market.$symbol.kline.week`|1-week kline update|`Kline`|
|`market.$symbol.kline.hour6`|6-hour kline update|`Kline`|
|`market.$symbol.kline.hour12`|12-hour kline update|`Kline`|

### Push Message Definitions

Push messages will be sent to the client continually after successsfully subscribing a topic.

#### Push messages pattern
```json tab:Response
  {
    "code": 0,  // error code
    "event": "push",  // event
    "timestamp": 1635659587885,  // time stamp
    "topic": "market.VX_VITE.depth, market.BTC-000_VITE.depth",  // topics, support single and multiple topic subscriptions, separated by ","
    "data": {...}  // data payload
  }
```

#### Order

* **Definition:**
```java
// order id
private String oid;
// symbol
private String s;
// trade token symbol
private String ts;
// quote token symbol
private String qs;
// trade tokenId
private String tid;
// quote tokenId
private String qid;
// side
private Integer side;
// price
private String p;
// quantity
private String q;
// amount
private String a;
// executed quantity
private String eq;
// executed amount
private String ea;
// executed percentage
private String ep;
// executed average price
private String eap;
// fee
private String f;
// status
private Integer st;
// type
private Integer tp;
// create time
private Long ct;
// address
private String d;
```

* **Example:**

  :::demo
  
  ```json tab:Subscribe
  {
    "command": "sub",
    "params": ["order.vite_cc392cbb42a22eebc9136c6f9ba416d47d19f3be1a1bd2c072"]
  }
  ```
  
  ```json tab:Response
  {
    "code": 0,
    "event": "push",
    "timestamp": 1635680744551,
    "topic": "order.vite_cc392cbb42a22eebc9136c6f9ba416d47d19f3be1a1bd2c072",
    "data": {
      "a":"13.72516176",
      "ct":1588142062,
      "d":"vite_cc392cbb42a22eebc9136c6f9ba416d47d19f3be1a1bd2c072",
      "ea":"13.7251",
      "eap":"0.1688",
      "ep":"1.0000",
      "eq":"81.3102",
      "f":"0.0308",
      "oid":"b0e0e20739c570d533679315dbb154201c8367b6e23636b6521e9ebdd9f8fc0a",
      "p":"0.1688",
      "q":"81.3102",
      "qid":"tti_80f3751485e4e83456059473",
      "qs":"USDT-000",
      "s":"VX_USDT-000",
      "side":0,
      "st":2,
      "tid":"tti_564954455820434f494e69b5",
      "tp":0,
      "ts":"VX"
    }
  }
  ```
  :::

#### Trade

* **Definition:**
```java
// tradeId
private String id;
// symbol
private String s;
// trade token symbol
private String ts;
// quote token symbol
private String qs;
// trade tokenId
private String tid;
// quote tokenId
private String qid;
// price
private String p;
// quantity
private String q;
// amount
private String a;
// time
private Long t;
// side: 0-buy, 1-sell
private Integer side;
// buyer orderId
private String bid;
// seller orderId
private String sid;
// buyer fee
private String bf;
// seller fee
private String sf;
// block height
private Long bh;
```

* **Example:**

  :::demo
  
  ```json tab:Subscribe
  {
    "command": "sub",
    "params": ["market.VITE_BTC-000.trade"]
  }
  ```
  
  ```json tab:Response
  {
    "code": 0,
    "event": "push",
    "timestamp": 1635682227087,
    "topic": "market.VITE_BTC-000.trade",
    "data":[
      {
        "a":"0.00018200",
        "bf":"0.00000045",
        "bh":48524712,
        "bid":"00000401000000000000001bc56000617e877c00001a",
        "id":"6473c6cc1ebc0031053963a4f73c0daf245cde9c",
        "p":"0.00000182",
        "q":"100.0",
        "qid":"tti_b90c9baffffc9dae58d1f33f",
        "qs":"BTC-000",
        "s":"VITE_BTC-000",
        "sf":"0.00000045",
        "sid":"00000400ffffffffffffffe43a9f00617e645f000012",
        "side":1,
        "t":1635682226,
        "tid":"tti_5649544520544f4b454e6e40",
        "ts":"VITE"
      }]
    }
  ```
  :::

#### TickerStatistics

* **Definition:**
```java
// symbol
private String s;
// trade token symbol
private String ts;
// quote token symbol
private String qs;
// trade tokenId
private String tid;
// quote tokenId
private String qid;
// open price
private String op;
// previous close price
private String pcp;
// close price
private String cp;
// price change 
private String pc;
// price change percentage
private String pCp;
// high price 
private String hp;
// low price
private String lp;
// quantity 
private String q;
// amount 
private String a;
// price precision
private Integer pp;
// quantity precision
private Integer qp;
```

* **Example:**

  :::demo
  
  ```json tab:Subscribe
  {
    "command": "sub",
    "params": ["market.VX_VITE.tickers"]
  }
  ```
  
  ```json tab:Response
  {
    "code": 0,
    "event": "push",
    "timestamp": 1635682791806,
    "topic": "market.VX_VITE.tickers",
    "data": {
      "a":"1438685.6001",
      "cp":"3.1004",
      "hp":"3.3499",
      "lp":"2.9297",
      "op":"3.2255",
      "pc":"-0.1251",
      "pcp":"3.1379",
      "pp":4,
      "q":"453495.3000",
      "qid":"tti_5649544520544f4b454e6e40",
      "qp":1,
      "qs":"VITE",
      "s":"VX_VITE",
      "tid":"tti_564954455820434f494e69b5",
      "ts":"VX"
    }
  }
  ```
  :::

#### KLine/Candlestick bars

* **Definition:**
```java
private Long t;  // timestamp in seconds
private Double c;  // close price
private Double o;  // open price
private Double v;  // volume
private Double h;  // high price
private Double l;  // low price
```

* **Example:**

  :::demo
  
  ```json tab:Subscribe
  {
    "command": "sub",
    "params": ["market.VX_VITE.kline.minute"]
  }
  ```
  
  ```json tab:Response
  {
    "code": 0,
    "event": "push",
    "timestamp": 1635683003661,
    "topic": "market.VX_VITE.kline.minute",
    "data": {
      "c":3.1128,
      "h":3.1128,
      "l":3.1128,
      "o":3.1128,
      "t":1635682980,
      "v":79.9
    }
  }
  ```
  :::

#### Depth

* **Definition:**
```java
private List<List<String>> asks;  // [[price, quantity],[price, quantity]]
private List<List<String>> bids;  // [[price, quantity],[price, quantity]]
```

* **Example:**

  :::demo
  
  ```json tab:Subscribe
  {
    "command": "sub", 
    "params": ["market.VX_VITE.depth"]
  }
  ```
  
  ```json tab:Response
  {
    "code": 0,
    "event": "push",
    "timestamp": 1635659587760,
    "topic": "market.VX_VITE.depth",
    "data": {
      "asks":[
        ["3.3240","78.2"],["3.3499","1000.0"],["3.3500","1947.0"],["3.3999","57.1"],["3.4000","300.0"],["3.4100","42.3"],["3.4199","1000.0"],["3.4200","50.0"],["3.4300","40.0"],["3.4398","108.2"],["3.4399","2000.0"],["3.4400","244.8"],["3.4499","1000.0"],["3.4500","52.3"],["3.4800","29.0"],["3.5000","50.0"],["3.5500","14.1"],["3.5998","1000.0"],["3.6000","623.9"],["3.6496","2000.0"],["3.6497","94.2"],["3.6500","399.5"],["3.6900","169.5"],["3.6999","20.0"],["3.7499","938.5"],["3.7500","842.5"],["3.7900","187.1"],["3.7999","1123.2"],["3.8000","300.0"],["3.8300","50.0"],["3.8600","20.0"],["3.9000","50.0"],["3.9800","100.0"],["3.9996","14.2"],["3.9997","7207.0"],["3.9998","1491.7"],["4.0000","1071.3"],["4.0500","30.9"],["4.0800","50.0"],["4.0900","1871.2"],["4.1000","100.0"],["4.1110","445.7"],["4.1400","25.7"],["4.1597","25.0"],["4.1600","192.6"],["4.2000","300.0"],["4.2396","100.0"],["4.2398","50.0"],["4.2400","25.4"],["4.2500","25.3"],["4.2600","25.0"],["4.2700","120.2"],["4.2750","43.0"],["4.2800","20.0"],["4.2995","54.0"],["4.3540","169.7"],["4.3900","50.0"],["4.3999","11.7"],["4.4000","50.0"],["4.4800","100.0"],["4.5000","300.0"],["4.6000","61.0"],["4.7500","100.0"],["4.7997","100.0"],["4.8000","300.0"],["4.9500","100.0"],["4.9800","100.0"],["5.0000","100.0"],["5.1000","300.0"],["5.1994","60.0"],["5.2800","100.0"],["5.3087","63.2"],["5.5000","337.3"],["5.5800","100.0"],["5.6000","100.0"],["5.7232","188.1"],["5.7500","100.0"],["5.7997","100.0"],["5.8961","119.5"],["5.9800","100.0"],["5.9995","100.0"],["5.9999","100.0"],["6.0000","463.6"],["6.1110","122.4"],["6.2800","100.0"],["6.5000","200.0"],["6.5170","100.0"],["6.6000","100.0"],["6.7500","100.0"],["6.7997","100.0"],["6.8999","136.6"],["6.9000","50.0"],["6.9170","100.0"],["6.9995","100.0"],["6.9999","100.0"],["7.0000","300.0"],["7.1100","1675.9"],["7.1170","100.0"],["7.2998","145.4"],["7.4170","100.0"]
      ],
      "bids":[
        ["3.1401","1564.6"],["3.1400","317.7"],["3.1301","1569.3"],["3.1200","445.8"],["3.1188","46.7"],["3.1100","22.7"],["3.1003","1924.5"],["3.1002","221.0"],["3.1001","72.7"],["3.1000","9897.7"],["3.0001","1151.4"],["2.9800","45.5"],["2.9712","1510.7"],["2.9708","1071.0"],["2.9700","134.0"],["2.9505","5418.0"],["2.9400","62.7"],["2.9300","46.0"],["2.9200","10000.0"],["2.9000","4799.5"],["2.8601","1014.0"],["2.8501","1000.0"],["2.8500","1358.4"],["2.8400","500.0"],["2.8200","10000.0"],["2.8100","100.0"],["2.8001","1000.0"],["2.8000","224.8"],["2.7900","50.0"],["2.7800","150.0"],["2.7700","47.0"],["2.7600","18.4"],["2.7501","1734.9"],["2.7500","1441.0"],["2.7401","50.0"],["2.7352","50.0"],["2.7200","53.0"],["2.7111","100.0"],["2.7100","50.0"],["2.7010","63.3"],["2.6900","51.9"],["2.6800","53.3"],["2.6700","59.1"],["2.6600","103.0"],["2.6501","88.0"],["2.6500","21878.7"],["2.6460","188.9"],["2.6400","121.2"],["2.6350","60.6"],["2.6300","50.0"],["2.6200","100.0"],["2.6000","48.5"],["2.5900","56.4"],["2.5700","108.0"],["2.5600","61.7"],["2.5500","46.3"],["2.5100","200.0"],["2.5000","118.3"],["2.3700","631.4"],["2.2222","448.9"],["2.2200","61.7"],["2.0100","496.3"],["2.0000","68.5"],["1.4600","125493.8"],["1.0687","1400.0"],["0.2001","53329.3"]
      ]
    }
  }
  ```
  :::
