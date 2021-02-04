# genesis.json

This shows a full example of genesis.json. In this example, 2 SBPs are selected in each round and each SBP is responsible for producing 3 continuous blocks.
Vite SBP is selected based on how many votes the supernode has in amount of **VITE**. In the example, one SBP is constantly assigned as the supernode who has most votes, while another is randomly selected from supernodes ranking from 2 to 100.
Registering supernode of snapshot consensus group needs staking **VITE**. Here 1000000000000000000000000 **VITE** for 7776000 snapshots, approximately equivalent to 3 months, is required.

:::warning
The example is a complete configuration file. You can make your own config based on it. But do remember to remove the text comments following after "//". 
The only reason the comments are here is to explain certain configuration settings. They are not part of the genesis config and should always be removed before you use this example in real. 
:::
```
{
  "GenesisAccountAddress": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792", // Genesis account
  "ConsensusGroupInfo": { // Consensus group settings
    "ConsensusGroupInfoMap": {
      "00000000000000000001": { // Consensus group id. "00000000000000000001" stands for snapshot consensus group. "00000000000000000002" stands for delegated consensus group
        "NodeCount": 2, // Number of SBP(s) selected in each round
        "Interval": 1,  // Snapshot block interval
        "PerCount": 3,  // Number of blocks each SBP continuously produces
        "RandCount": 1, // Number of SBP(s) selected in random
        "RandRank": 100,  // Supernode ranking to which the random SBP(s) is selected
        "Repeat": 1,  // Number of round in which the consensus result takes effect. No need to change
        "CheckLevel": 0,  // Consensus verification level. No need to change
        "CountingTokenId": "tti_5649544520544f4b454e6e40",  // Token id in which the voting number is calculated. Default is VITE
        "RegisterConditionId": 1, // Registration type id. No need to change
        "RegisterConditionParam": {
          "PledgeAmount": 1000000000000000000000000, // Amount of staking upon supernode registration
          "PledgeToken": "tti_5649544520544f4b454e6e40",  // Staking token id
          "PledgeHeight": 7776000 // Staking period in number of snapshots
        },
        "VoteConditionId": 1, // Voting type id. No need to change
        "VoteConditionParam": {}, // No need to change
        "Owner": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792", // Owner of snapshot consensus group. No need to change
        "PledgeAmount": 0,  // Amount of staking for this consensus group. No need to change
        "WithdrawHeight": 1 // Staking period for this consensus group. No need to change
      },
      "00000000000000000002": {
        "NodeCount": 2,
        "Interval": 3,
        "PerCount": 1,
        "RandCount": 2,
        "RandRank": 100,
        "Repeat": 48,
        "CheckLevel": 1,
        "CountingTokenId": "tti_5649544520544f4b454e6e40",
        "RegisterConditionId": 1,
        "RegisterConditionParam": {
          "PledgeAmount": 1000000000000000000000000,
          "PledgeToken": "tti_5649544520544f4b454e6e40",
          "PledgeHeight": 7776000
        },
        "VoteConditionId": 1,
        "VoteConditionParam": {},
        "Owner": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792",
        "PledgeAmount": 0,
        "WithdrawHeight": 1
      }
    },
    "RegistrationInfoMap": {  // Supernode settings
      "00000000000000000001": { 
        "s1": { // Supernode name
          "NodeAddr": "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",  // Block producing address
          "PledgeAddr": "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906",  // Staking address
          "Amount": 1000000000000000000000000, // Staking amount
          "WithdrawHeight": 7776000,  // Staking period
          "RewardTime": 1,  // Starting block height from which mining rewards are available for retrieval
          "CancelTime": 0,  // Block height at which the supernode can be cancelled and the corresponding staking can be retrieved. 0 means the supernode can be cancelled immediately
          "HisAddrList": [  // Historical block producing addresses
            "vite_e41be57d38c796984952fad618a9bc91637329b5255cb18906"
          ]
        },
        "s2": {
          "NodeAddr": "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
          "PledgeAddr": "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
          "Amount": 1000000000000000000000000,
          "WithdrawHeight": 7776000,
          "RewardTime": 1,
          "CancelTime": 0,
          "HisAddrList": [
            "vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87"
          ]
        }
      }
    }
    "VoteStatusMap": {  // Voter settings
      "00000000000000000001":{  
        "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792":"s1", // Voter address and the name of supernode voted
        "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23":"s2"
      }
    }
  },
  "MintageInfo": {  // Token settings
    "TokenInfoMap": {
      "tti_5649544520544f4b454e6e40": { // Token id
        "TokenName": "Vite Token",  // Token name
        "TokenSymbol": "VITE",  // Token symbol
        "TotalSupply": 1000000000000000000000000000,  // Total supply. The total supply of VITE is 1e9 * 1e18
        "Decimals": 18, // Decimals
        "Owner": "vite_0000000000000000000000000000000000000004d28108e76b", // Token owner
        "PledgeAmount": 0,  // 铸币时抵押的vite金额
        "PledgeAddr": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792",  // 铸币时的抵押地址
        "WithdrawHeight": 0,  // 铸币时的抵押到期高度
        "MaxSupply": 115792089237316195423570985008687907853269984665640564039457584007913129639935,  // Maximum supply
        "OwnerBurnOnly": false, // Boolean flag representing if the token can only be burned by token owner. Re-issuable token only.
        "IsReIssuable": true  // Boolean flag representing if the token is re-issuable or not
      }
    }
  },
  "PledgeInfo": { // Quota settings
    "PledgeBeneficialMap": { 
      "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792": 1000000000000000000000,  // Quota recipient address and staking amount
      "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": 1000000000000000000000
    },
    "PledgeInfoMap": {  
      "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792": [  // Staking address
        {
          "Amount": 1000000000000000000000, // Staking amount
          "WithdrawHeight": 259200, // Staking period
          "BeneficialAddr": "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792" // Quota recipient address
        }
      ],
      "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": [
        {
          "Amount": 1000000000000000000000,
          "WithdrawHeight": 259200,
          "BeneficialAddr": "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23"
        }
      ]
    }
  },
  "AccountBalanceMap": {  // Account settings
    "vite_bb6ad02107a4422d6a324fd2e3707ad53cfed9359378a78792": {  // Address
      "tti_5649544520544f4b454e6e40": 899798000000000000000000000 // Token id and amount
    },
    "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": {
      "tti_5649544520544f4b454e6e40": 100000000000000000000000000
    },
    "vite_0000000000000000000000000000000000000004d28108e76b":{ 
      "tti_5649544520544f4b454e6e40": 200000000000000000000000
    },
    "vite_0000000000000000000000000000000000000003f6af7459b9":{ 
      "tti_5649544520544f4b454e6e40": 2000000000000000000000
    }
  }
}
```
