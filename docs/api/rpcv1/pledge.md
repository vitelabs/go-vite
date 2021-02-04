---
sidebarDepth: 4
---

# Pledge
:::tip Maintainer
[viteLiz](https://github.com/viteLiz)
:::

## Contract Specification
Built-in staking contract. Contract address is `vite_0000000000000000000000000000000000000003f6af7459b9`

ABI：

```json
[
  // Staking for quota
  {"type":"function","name":"Pledge", "inputs":[{"name":"beneficial","type":"address"}]},
  // Cancel staking
  {"type":"function","name":"CancelPledge","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
  // Staking for a quota via delegation
  {"type":"function","name":"AgentPledge", "inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"bid","type":"uint8"}]},
  // Callback function for delegated staking
  {"type":"function","name":"AgentPledgeCallback","inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]},
  // Cancel staking via delegation
  {"type":"function","name":"AgentCancelPledge","inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"}]},
  // Callback function for cancelling delegated staking
  {"type":"function","name":"AgentCancelPledgeCallback","inputs":[{"name":"pledgeAddress","type":"address"},{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"},{"name":"bid","type":"uint8"},{"name":"success","type":"bool"}]}		
]
]
```

Delegated staking and cancelling delegated staking will return execution results in callbacks.

## pledge_getPledgeData
Generate request data for staking for quota. Equivalent to `Pledge` method in ABI.

- **Parameters**: 

  * `Address`: Quota recipient's address

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"pledge_getPledgeData",
   "params":["vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"]
}
```

:::

## pledge_getCancelPledgeData
Generate request data for retrieving a certain amount of tokens staked for specified quota recipient. Equivalent to `CancelPledge` method in ABI.

- **Parameters**: 

  * `Address`: Quota recipient's address
  * `big.int`: The amount of token to withdraw

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"pledge_getCancelPledgeData",
   "params":[
      "vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
      "10"
    ]
}
```

:::

## pledge_getAgentPledgeData
Generate request data for staking for specified quota recipient via delegation. Equivalent to `AgentPledge` method in ABI.

- **Parameters**: 

`Object`
  1. `pledgeAddr`:`Address`  Staking address
  2. `beneficialAddr`:`Address`  Quota recipient's address
  3. `bid`:`uint8`  Business id. Multiple staking from the same staking address and business id will be merged. As result, staking expiration will be extended
  4. `stakeHeight`:`uint64`  Staking duration between 259200 and 31536000. For example, a staking duration of 259200 represents the staking can be retrieved after 259200 snapshot blocks

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"pledge_getAgentPledgeData",
   "params":[
      {
      	"pledgeAddr":"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23",
      	"beneficialAddr":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
      	"bid":1
      }
   ]
}
```

:::

## pledge_getAgentCancelPledgeData
Generate request data for retrieving a certain amount of tokens staked for specified quota recipient via delegation .Equivalent to `AgentCancelPledge` method in ABI.

- **Parameters**: 

`Object`
  1. `pledgeAddr`:`Address`  Staking address
  2. `beneficialAddr`:`Address`  Quota recipient's address
  3. `bid`:`uint8`  Business id
  4. `amount`:`big.Int`  The amount of tokens to withdraw

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"pledge_getAgentCancelPledgeData",
   "params":[
      {
      	"pledgeAddr":"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23",
      	"beneficialAddr":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
      	"amount":"200000000000000000000",
      	"bid":1
      }
    ]
}
```

:::

## pledge_getPledgeQuota
Return current quota and UTPS (unit transaction per second) of specified account. 

- **Parameters**: 

  * `Address`: Account address

- **Returns**: 

`Object`
  1. `quota`: `uint64`  Current quota
  2. `quotaPerSnapshotBlock`: `uint64` Quota obtained per snapshot block through staking
  3. `currentUt`: `float`: Current quota, measured in UT with 4 decimals
  4. `utpe`: `float`: The maximum quota can be obtained with current staking amount, measured in UT with 4 decimals
  5. `pledgeAmount`: `big.Int`: The staking amount

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"pledge_getPledgeQuota",
   "params": ["vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": {
      "current": "1575000",
      "quotaPerSnapshotBlock": "21000",
      "currentUt": "75",
      "utpe": "75",
      "pledgeAmount": "134000000000000000000"
   }
}
```
:::

## pledge_getPledgeList
Return staking records of specified account, ordered by expiration snapshot block height in descending order

- **Parameters**: 

  * `Address`: Staking address
  * `int`: Page index, starting from 0
  * `int`: Page size

- **Returns**: 
      
`Object`
  1. `totalPledgeAmount`: `big.Int`  The total staking amount of the account
  2. `totalCount`: `int`  The number of total staking records
  3. `pledgeInfoList`: `Array<PledgeInfo>` Staking record list
     * `amount`: `big.int`  Staking amount
     * `withdrawHeight`: `uint64`  The snapshot block height when the staking expires
     * `beneficialAddr`: `Address`  Quota recipient's address
     * `withdrawTime`: `int64`  The estimated staking expiration time
     * `agent`: `bool`  Whether this is delegated staking. `true` for yes and `false` for no.
     * `agentAddress`: `Address` Delegated account address. 0 for non-delegated staking
     * `bid`: `uint8`  Business id. 0 for non-delegated staking

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"pledge_getPledgeList",
   "params":["vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6", 0, 50]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result":  {
      "totalPledgeAmount": "5000000000000000000000",
      "totalCount": 1,
      "pledgeInfoList": [
        {
          "amount":10000000000000000000,
          "withdrawHeight":259200,
          "beneficialAddr":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
          "withdrawTime":1540213336
        }
      ]
   }
}
```
:::

## pledge_getAgentPledgeInfo
Get delegated staking 

- **Parameters**: 

`Object`
  1. `pledgeAddr`:`Address`  Original staking address
  2. `agentAddr`:`Address`  Delegated account address
  3. `beneficialAddr`:`Address`  Quota recipient's address
  4. `bid`:`uint8`  Business id. Multiple staking from the same staking address and business id will be merged. As result, staking expiration will be extended.
- **Returns**: 

`Object`
  1.`amount`: `big.int`  Staking amount
  2.`withdrawHeight`: `uint64`  The snapshot block height when the staking expires
  3.`beneficialAddr`: `Address`  Quota recipient's address
  4.`withdrawTime`: `int64`  The estimated staking expiration time
  5.`agent`: `bool`  Whether this is delegated staking. `true` for yes and `false` for no.
  6.`agentAddress`: `Address`  Delegated account address. 0 for non-delegated staking
  7.`bid`: `uint8`  Business id. 0 for non-delegated staking

- **Example**:


::: demo


```json tab:Request
{
	"jsonrpc": "2.0",
	"id": 1,
	"method": "pledge_getAgentPledgeInfo",
	"params": [
		{
			"pledgeAddr":"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23", 
			"agentAddr":"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
			"beneficialAddr":"vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
			"bid":1
		}
	]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "amount": "1000000000000000000000",
        "withdrawHeight": "259992",
        "beneficialAddr": "vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6",
        "withdrawTime": 1561098331,
        "agent": true,
        "agentAddress": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
        "bid": 1
    }
}
```
:::
