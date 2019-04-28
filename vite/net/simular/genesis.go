package main

const genesis = `
{
    "GenesisAccountAddress": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
    "ForkPoints": {},
    "ConsensusGroupInfo": {
        "ConsensusGroupInfoMap": {
            "00000000000000000001": {
                "NodeCount": 3,
                "Interval": 1,
                "PerCount": 3,
                "RandCount": 2,
                "RandRank": 100,
                "Repeat": 1,
                "CheckLevel": 0,
                "CountingTokenId": "tti_5649544520544f4b454e6e40",
                "RegisterConditionId": 1,
                "RegisterConditionParam": {
                    "PledgeAmount": 100000000000000000000000,
                    "PledgeHeight": 1,
                    "PledgeToken": "tti_5649544520544f4b454e6e40"
                },
                "VoteConditionId": 1,
                "VoteConditionParam": {},
                "Owner": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "PledgeAmount": 0,
                "WithdrawHeight": 1
            },
            "00000000000000000002": {
                "NodeCount": 3,
                "Interval": 3,
                "PerCount": 1,
                "RandCount": 2,
                "RandRank": 100,
                "Repeat": 48,
                "CheckLevel": 1,
                "CountingTokenId": "tti_5649544520544f4b454e6e40",
                "RegisterConditionId": 1,
                "RegisterConditionParam": {
                    "PledgeAmount": 100000000000000000000000,
                    "PledgeHeight": 1,
                    "PledgeToken": "tti_5649544520544f4b454e6e40"
                },
                "VoteConditionId": 1,
                "VoteConditionParam": {},
                "Owner": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "PledgeAmount": 0,
                "WithdrawHeight": 1
            }
        },
        "RegistrationInfoMap": {
            "00000000000000000001": {
                "s1": {
                    "NodeAddr": "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a",
                    "PledgeAddr": "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a",
                    "Amount": 100000000000000000000000,
                    "WithdrawHeight": 7776000,
                    "RewardTime": 1,
                    "CancelTime": 0,
                    "HisAddrList": ["vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a"]
                },
                "s2": {
                    "NodeAddr": "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25",
                    "PledgeAddr": "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25",
                    "Amount": 100000000000000000000000,
                    "WithdrawHeight": 7776000,
                    "RewardTime": 1,
                    "CancelTime": 0,
                    "HisAddrList": ["vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25"]
                },
                "s3": {
                    "NodeAddr": "vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a",
                    "PledgeAddr": "vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a",
                    "Amount": 100000000000000000000000,
                    "WithdrawHeight": 7776000,
                    "RewardTime": 1,
                    "CancelTime": 0,
                    "HisAddrList": ["vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a"]
                }
            }
        },
        "VoteStatusMap": {
            "00000000000000000001": {
                "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a": "s1",
                "vite_3d544acf90181a0449bb22b4cf02eb4f0bd3ce695da9abc9f8": "s1",
                "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25": "s2",
                "vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83": "s2",
                "vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a": "s3",
                "vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d": "s3"
            }
        }
    },
    "MintageInfo": {
        "TokenInfoMap": {
            "tti_5649544520544f4b454e6e40": {
                "TokenName": "Vite Token",
                "TokenSymbol": "VITE",
                "TotalSupply": 1000000000000000000000000000,
                "Decimals": 18,
                "Owner": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "PledgeAmount": 0,
                "PledgeAddr": "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
                "WithdrawHeight": 0,
                "MaxSupply": 115792089237316195423570985008687907853269984665640564039457584007913129639935,
                "OwnerBurnOnly": false,
                "IsReIssuable": true
            }
        },
        "LogList": [{
            "Data": "",
            "Topics": ["3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6", "000000000000000000000000000000000000000000005649544520544f4b454e"]
        }]
    },
    "PledgeInfo": {
        "PledgeBeneficialMap": {
            "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a": 1000000000000000000000,
            "vite_3d544acf90181a0449bb22b4cf02eb4f0bd3ce695da9abc9f8": 1000000000000000000000,
            "vite_805f96789429b8ae457a8d1423586a552db7eaa503741e1e8a": 1000000000000000000000,
            "vite_6fb5cb4f31d9baa92296920d304055b4d15dd91d08ec3ce344": 1000000000000000000000,
            "vite_8a6e2f4d0cbd12be6187dddc5aa00c2deb8b0efc9abb60fb59": 1000000000000000000000,
            "vite_e01fb307fd62b5e927d62c372837ad9669e04678899f3a87e4": 1000000000000000000000,
            "vite_d23015581d8d9628fbed001ec19ea8395efecdf712b73591e3": 1000000000000000000000,
            "vite_97506a3c2f7953c1db151ee6d5bad250fea1b1e2bc3887900a": 1000000000000000000000,
            "vite_3e9544e791a594252ae6a77ee57bd6cfe2364fede6d7f6ad10": 1000000000000000000000,
            "vite_f45503646d3c19834a5d55cf3087430f3edf9bb01a37917f6c": 1000000000000000000000,
            "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25": 1000000000000000000000,
            "vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83": 1000000000000000000000,
            "vite_a6059f0227861c3c8c6bb21c8fc85ac9f69d618aeb4f77bc4c": 1000000000000000000000,
            "vite_0211767ebbae78be48992bd1d92b9c6fc355e93449862bdaa7": 1000000000000000000000,
            "vite_8ec49c82a2a9d1c5ff5a9b84eac8788a9d193ddefc628852cc": 1000000000000000000000,
            "vite_cb3fa168df4f57a53880c06399ff480021dfe15fd2701410e4": 1000000000000000000000,
            "vite_28b0f39cd7b9a0d416ac9f96a3dfb420f08f0f82baa32caac7": 1000000000000000000000,
            "vite_8465e9f092bdf84a14d4e459251fed6fdb2baaca70ca4fdcd2": 1000000000000000000000,
            "vite_d57ed4838f64a158af205d362e0ef5f3603bc7c82e9cd7f280": 1000000000000000000000,
            "vite_389c44f54663c5ab45a385b782bd3406fd42df3b11b4713a88": 1000000000000000000000,
            "vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a": 1000000000000000000000,
            "vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d": 1000000000000000000000,
            "vite_06501648d1d4f6e3f55b6e54d5d4cafa1d22567067e829d860": 1000000000000000000000,
            "vite_e82e1bf245292d3f5505f41fbb50b41382b8f562a9cd30ad4e": 1000000000000000000000,
            "vite_004d7d2f8f1f18a7d69e2d28a13d0bb1d2c1361b91acf497dc": 1000000000000000000000,
            "vite_083dcd368c8f95904a523735dc68df821f22dac3b0c084ba8a": 1000000000000000000000,
            "vite_aaf1f8539a3abd1955de068a9e78255fee90039c465b13cbb5": 1000000000000000000000,
            "vite_721f7c4ca3148b35aeead851f9443d25d98370ef18a3df0901": 1000000000000000000000,
            "vite_ddf74c69fa61f0f52a0bb67f842ec6896f1f35801cda2a61df": 1000000000000000000000,
            "vite_e3f04896576a352cbbf2b4cd8c521bda2b724f81081bb44034": 1000000000000000000000
        }
    },
    "AccountBalanceMap": {
        "vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a": {
            "tti_5649544520544f4b454e6e40": 899999000000000000000000000
        },
        "vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_360232b0378111b122685a15e612143dc9a89cfa7e803f4b5a": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_3d544acf90181a0449bb22b4cf02eb4f0bd3ce695da9abc9f8": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_805f96789429b8ae457a8d1423586a552db7eaa503741e1e8a": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_6fb5cb4f31d9baa92296920d304055b4d15dd91d08ec3ce344": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_8a6e2f4d0cbd12be6187dddc5aa00c2deb8b0efc9abb60fb59": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_e01fb307fd62b5e927d62c372837ad9669e04678899f3a87e4": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_d23015581d8d9628fbed001ec19ea8395efecdf712b73591e3": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_97506a3c2f7953c1db151ee6d5bad250fea1b1e2bc3887900a": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_3e9544e791a594252ae6a77ee57bd6cfe2364fede6d7f6ad10": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_f45503646d3c19834a5d55cf3087430f3edf9bb01a37917f6c": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_ce18b99b46c70c8e6bf34177d0c5db956a8c3ea7040a1c1e25": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_999fb65a4553fcec4f68e7262a8014678ea0f594c49d843b83": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_a6059f0227861c3c8c6bb21c8fc85ac9f69d618aeb4f77bc4c": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_0211767ebbae78be48992bd1d92b9c6fc355e93449862bdaa7": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_8ec49c82a2a9d1c5ff5a9b84eac8788a9d193ddefc628852cc": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_cb3fa168df4f57a53880c06399ff480021dfe15fd2701410e4": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_28b0f39cd7b9a0d416ac9f96a3dfb420f08f0f82baa32caac7": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_8465e9f092bdf84a14d4e459251fed6fdb2baaca70ca4fdcd2": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_d57ed4838f64a158af205d362e0ef5f3603bc7c82e9cd7f280": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_389c44f54663c5ab45a385b782bd3406fd42df3b11b4713a88": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_40996a2ba285ad38930e09a43ee1bd0d84f756f65318e8073a": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_cdedd375b6001c2081e829d80f1b336c0564fb96f1e2d2e93d": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_06501648d1d4f6e3f55b6e54d5d4cafa1d22567067e829d860": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_e82e1bf245292d3f5505f41fbb50b41382b8f562a9cd30ad4e": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_004d7d2f8f1f18a7d69e2d28a13d0bb1d2c1361b91acf497dc": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_083dcd368c8f95904a523735dc68df821f22dac3b0c084ba8a": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_aaf1f8539a3abd1955de068a9e78255fee90039c465b13cbb5": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_721f7c4ca3148b35aeead851f9443d25d98370ef18a3df0901": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_ddf74c69fa61f0f52a0bb67f842ec6896f1f35801cda2a61df": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        },
        "vite_e3f04896576a352cbbf2b4cd8c521bda2b724f81081bb44034": {
            "tti_5649544520544f4b454e6e40": 100000000000000000000000000
        }
    }
}
`
