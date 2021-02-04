---
sidebarDepth: 4
---

# ConsensusGroup
:::tip Maintainer
[viteLiz](https://github.com/viteLiz)
:::

## Contract Specification
Built-in consensus group contract. Contract address is `vite_0000000000000000000000000000000000000004d28108e76b`

ABI：

```json
[
  // Register block producer
  {"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
  // Update registration
  {"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"Name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
  // Cancel registration
  {"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
  // Retrieve mining rewards
  {"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"}]},
  // Vote for block producer
  {"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeName","type":"string"}]},
  // Cancel voting
  {"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]}
]
```

## register_getRegisterData
Generate request data for registering a new supernode in the specified consensus group. Equivalent to `Register` method in ABI.

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `string`: Supernode name
  * `Address`: Block producing address

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getRegisterData",
   "params": [
      "00000000000000000001",
      "super", 
      "vite_080b2d68a06f52c0fbb454f675ee5435fb7872526771840d22"
    ]
}
```

:::

## register_getCancelRegisterData
Generate request data for cancelling existing supernode registration in the specified consensus group. Equivalent to `CancelRegister` method in ABI.

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `string`: Supernode name

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getCancelRegisterData",
   "params": [
      "00000000000000000001",
      "super"
    ]
}
```

:::

## register_getRewardData
Generate request data for retrieving supernode rewards if applied. Rewards in 90 days(or all accumulated un-retrieved rewards if less than 90 days) since last retrieval can be retrieved per request. Cannot retrieve rewards generated in recent 30 minutes. Equivalent to `Reward` method in ABI. 

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `string`: Supernode name
  * `Address`: The address to receive rewards

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getRewardData",
   "params": [
      "00000000000000000001", 
      "super",
      "vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"
   ]
}
```

:::

## register_getUpdateRegistrationData
Generate request data for changing block producer in existing registration. Equivalent to `UpdateRegistration` method in ABI.

- **Parameters**: 
  
  * `Gid`: Consensus group ID, must be the same value as registered and cannot be changed
  * `string`: Supernode name, must be the same value as registered and cannot be changed
  * `Address`: New block producing address

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getUpdateRegistrationData",
   "params": [
      "00000000000000000001", 
      "super",
      "vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"
    ]
}
```

:::

## register_getRegistrationList
Return a list of supernodes registered in the specified consensus group by account, ordered by expiration height

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `Address`: Staking address

- **Returns**: 

`Array&lt;RegistartionInfo&gt;`
  1. `name`: `string`  Supernode name
  2. `nodeAddr`: `Address`  Block producing address
  3. `pledgeAddr`: `Address`  Staking address
  4. `pledgeAmount`: `big.Int`  Staking amount
  5. `withdrawHeight`: `uint64`  Staking expiration height
  6. `withdrawTime`: `uint64`  Estimated staking expiration time
  7. `cancelHeight`: `uint64`  Staking cancellation time. If the value > 0, meaning already cancelled.

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getRegistrationList",
   "params": [
      "00000000000000000001",
      "vite_080b2d68a06f52c0fbb454f675ee5435fb7872526771840d22"
    ]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": [
    {
      "name": "super",
      "nodeAddr": "",
      "pledgeAddr": "",
      "pledgeAmount": 100000000000,
      "withdrawHeight": 100,
      "withdrawTime":1541573925,
      "cancelTime":0,
    }
   ]
}
```
:::

## register_getAvailableReward
Return un-retrieved rewards in the specified consensus group by supernode name

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `string`: Supernode name

- **Returns**: 

`RewardInfo`
  1. `totalReward`: `string`  Accumulated un-retrieved rewards
  2. `blockReward`: `Address`  Accumulated un-retrieved block creation rewards
  3. `voteReward`: `Address`  Accumulated un-retrieved candidate additional rewards(voting rewards)
  4. `drained`: `bool`  Return `true` only if the supernode has been canceled and all un-retrieved reward is zero

- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getAvailableReward",
   "params": [
      "00000000000000000001",
      "super"
    ]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": 
    {
      "totalReward": "10",
      "blockReward": "6",
      "voteReward": "4",
      "drained":true
    }
}
```
:::

## register_getRewardByDay
Return daily rewards for all supernodes in the specified consensus group by timestamp

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `timestamp`: Unix timestamp

- **Returns**: 

`map<string>RewardInfo` 
  1. `totalReward`: `string`  Total rewards in the day
  2. `blockReward`: `Address`  Block creation rewards in the day
  3. `voteReward`: `Address`  Candidate additional rewards(voting rewards) in the day
  4. `expectedBlockNum`: `uint64` Target block producing number in the day. If zero block is produced by all supernodes in a round, the target block producing number in this round won't be counted in that of the day  
  5. `blockNum`: `uint64`  Real block producing number in the day


- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getRewardByDay",
   "params": [
      "00000000000000000001",
      1555567989
    ]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": 
    {
      "super":{
        "totalReward": "10",
        "blockReward": "6",
        "voteReward": "4",
        "expectedBlockNum":3,
        "blockNum":1,
      }
    }
}
```
:::

## register_getRewardByIndex
Return daily rewards for all supernodes in the specified consensus group by cycle

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `uint64`: Index of cycle in 24h starting at 0 from 12:00:00 UTC+8 05/21/2019

- **Returns**: 

`Object`
  1. `rewardMap`:`map<string>RewardInfo` Detailed reward information
  2. `startTime`:`int64` Start time
  3. `endTime`:`int64` End time


- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"register_getRewardByIndex",
   "params": [
      "00000000000000000001",
      "0"
    ]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": 
    {
      "rewardMap":{
        "super":{
          "totalReward": "10",
          "blockReward": "6",
          "voteReward": "4",
          "expectedBlockNum":3,
          "blockNum":1,
        }
      },
      "startTime": 1558411200,
      "endTime": 1558497600
    }
}
```
:::


## register_getCandidateList
Return a list of SBP candidates in snapshot consensus group

- **Parameters**: 

- **Returns**: 

`Array&lt;CandidateInfo&gt;`
  1. `name`: `string`  SBP name
  2. `nodeAddr`: `Address`  Block producing address
  3. `voteNum`: `string`  Number of votes

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id": 1,
   "method":"register_getCandidateList",
   "params": []
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 17,
    "result": [
        {
            "name": "s1",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s10",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s11",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s12",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s13",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s14",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s15",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s16",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s17",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s18",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s19",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s2",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s20",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s21",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s22",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s23",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s24",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s25",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s3",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s4",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s5",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s6",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s7",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s8",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        },
        {
            "name": "s9",
            "nodeAddr": "vite_0000000000000000000000000000000000000000a4f3a0cb58",
            "voteNum": "0"
        }
    ]
}
```
:::

## vote_getVoteData
Generate request data for voting for the named supernode in the specified consensus group. Equivalent to `Vote` method in ABI.

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `string`: Supernode name

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"vote_getVoteData",
   "params": [
      "00000000000000000001", 
      "super"
    ]
}
```

:::

## vote_getCancelVoteData
Generate request data for cancelling voting in the specified consensus group. Equivalent to `CancelVote` method in ABI.

- **Parameters**: 

  * `Gid`: Consensus group ID

- **Returns**: 
	- `[]byte` Data

- **Example**:


::: demo


```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"vote_getCancelVoteData",
   "params":["00000000000000000001"]
}
```

:::

## vote_getVoteInfo
Return the voting information in the specified consensus group by account

- **Parameters**: 

  * `Gid`: Consensus group ID
  * `Address`: Account address

- **Returns**: 

`Object`
  1. `nodeName`: `string`  Supernode name
  2. `nodeStatus`: `uint8`  Supernode registration status. 1->valid. 2->invalid
  3. `balance`: `big.Int`  Account balance
  
- **Example**:

::: demo

```json tab:Request
{  
   "jsonrpc":"2.0",
   "id":1,
   "method":"vote_getVoteInfo",
   "params": [
      "00000000000000000001", 
      "vite_a5a7f08011c2f0e40ccd41b5b79afbfb818d565f566002d3c6"
    ]
}
```

```json tab:Response
{  
   "jsonrpc":"2.0",
   "id":1,
   "result": {
      "nodeName": "super",
      "nodeStatus": 1
      "balance": 10,
   }
}
```
:::


## vote_getVoteDetails
Return daily SBP voting information 

- **Parameters**: 

  * `Index`: Index of days. `0` stands for the first launching day
 

- **Returns**: 

`Array&lt;VoteDetails&gt;`
  1. `Name`: `string`  SBP name
  2. `Balance`: `big.Int`  Total number of votes
  3. `Addr`: `map`  Map of "voting address: voting amount"
  
- **Example**:

::: demo

```json tab:Request
{
    "jsonrpc": "2.0",
    "id": 17,
    "method":"vote_getVoteDetails",
    "params":[0]
}
```

```json tab:Response
{
    "jsonrpc": "2.0",
    "id": 17,
    "result": [
        {
            "Name": "Vite_SBP01",
            "Balance": 1.4292529985394005795698375e+25,
            "Type": null,
            "CurrentAddr": "vite_9065ff0e14ebf983e090cde47d59fe77d7164b576a6d2d0eda",
            "RegisterList": [
                "vite_9065ff0e14ebf983e090cde47d59fe77d7164b576a6d2d0eda"
            ],
            "Addr": {
                "vite_002c698c0f89662679d03eb65344cea6ed18ab64cd3562e399": 153173820149812829427,
                "vite_002f9cc5863815d27679b3a5e4f47675e680926a7ae5365d5e": 2.2637888780120236595891e+22,
                "vite_0032cc9274aa2b3de392cf8f0840ebae0367419d11219bcd7e": 0,
                "vite_003853b6b237b311a87029a669d589b19c97674d8b5473004f": 211999319045896173105,
                "vite_0047b193c7d7791a94de7e45b7febf6eac8139fd81695cfdb5": 27349565445264753118,
                "vite_dd8364662b3725ab89fff765091b8bc6a6e140adbfbfc3baca": 19926226954228583374
            }
        },
        {
            "Name": "Vite_SBP02",
            "Balance": 6.905516255516640791260504e+24,
            "Type": null,
            "CurrentAddr": "vite_995769283a01ba8d00258dbb5371c915df59c8657335bfb1b2",
            "RegisterList": [
                "vite_995769283a01ba8d00258dbb5371c915df59c8657335bfb1b2"
            ],
            "Addr": {
                "vite_013661a2b0ac7a7344b94308184105dfae64bb746aadfeb3eb": 1341892617448983441,
                "vite_0139fa07eccdd3945941d6bd376ffb67db771cfb5999439639": 83666851810677644147
            }
        }
   ]
}
```
:::
