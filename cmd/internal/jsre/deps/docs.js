require = (function () {
    function r(e, n, t) {
        function o(i, f) {
            if (!n[i]) {
                if (!e[i]) {
                    var c = "function" == typeof require && require;
                    if (!f && c) return c(i, !0);
                    if (u) return u(i, !0);
                    // var a = new Error("Cannot find module '" + i + "'"); throw a.code = "MODULE_NOT_FOUND", a
                } var p = n[i] = { exports: {} }; e[i][0].call(p.exports, function (r) { var n = e[i][1][r]; return o(n || r) }, p, p.exports, r, e, n, t)
            } return n[i].exports
        } for (var u = "function" == typeof require && require, i = 0; i < t.length; i++)o(t[i]); return o
    }
    return r
})
    ()({
        "docs": [function (require, module, exports) {
            "use strict";

            Object.defineProperty(exports, "__esModule", {
                value: true
            });
            exports.default = void 0;
            var _default = {
                //-----------down--------- 在此处修改
                help: {// vite.help
                    "wallet": {
                        "listEntropyFilesInStandardDir": "return sth",
                        "listAllEntropyFiles": "",
                        "unlock": "",
                        "lock": "",
                        "listEntropyStoreAddresses": "",
                        "newMnemonicAndEntropyStore": "",
                        "deriveForIndexPath": "",
                        "recoverEntropyStoreFromMnemonic": "",
                        "globalCheckAddrUnlocked": "",
                        "isAddrUnlocked": "",
                        "isUnlocked": "",
                        "findAddr": "",
                        "globalFindAddr": "",
                        "createTxWithPassphrase": "",
                        "addEntropyStore": ""
                    },
                    "net": {
                        "syncInfo": "",
                        "peers": ""
                    },
                    "onroad": {
                        "getOnroadBlocksByAddress": "",
                        "getAccountOnroadInfo": "",
                        "listWorkingAutoReceiveWorker": "",
                        "startAutoReceive": "",
                        "stopAutoReceive": ""
                    },
                    "contract": {
                        "getCreateContractToAddress": ""
                    },
                    "pledge": {
                        "getPledgeData": "",
                        "getCancelPledgeData": "",
                        "getPledgeQuota": "",
                        "getPledgeList": ""
                    },
                    "register": {
                        "getSignDataForRegister": "",
                        "getRegisterData": "",
                        "getCancelRegisterData": "",
                        "getRewardData": "",
                        "getUpdateRegistrationData": ""
                    },
                    "vote": {
                        "getVoteData": "",
                        "getCancelVoteData": ""
                    },
                    "mintage": {
                        "getMintageData": "",
                        "getMintageCancelPledgeData": ""
                    },
                    "consensusGroup": {
                        "getConditionRegisterOfPledge": "",
                        "getConditionVoteOfDefault": "",
                        "getConditionVoteOfKeepToken": "",
                        "getCreateConsensusGroupData": "",
                        "getCancelConsensusGroupData": "",
                        "getReCreateConsensusGroupData": ""
                    },
                    "ledger": {
                        "getBlocksByAccAddr": "",
                        "getAccountByAccAddr": "",
                        "getLatestSnapshotChainHash": "",
                        "getLatestBlock": "",
                        "getTokenMintage": "",
                        "getBlocksByHash": "",
                        "getSnapshotChainHeight": "",
                        "getFittestSnapshotHash": ""
                    },
                    "tx": {
                        "sendRawTx": ""
                    }
                },

                "wallet": {//vite.wallet_listEntropyFilesInStandardDir.help
                    "listEntropyFilesInStandardDir": "return sth",
                    "listAllEntropyFiles": "",
                    "unlock": "",
                    "lock": "",
                    "listEntropyStoreAddresses": "",
                    "newMnemonicAndEntropyStore": "",
                    "deriveForIndexPath": "",
                    "recoverEntropyStoreFromMnemonic": "",
                    "globalCheckAddrUnlocked": "",
                    "isAddrUnlocked": "",
                    "isUnlocked": "",
                    "findAddr": "",
                    "globalFindAddr": "",
                    "createTxWithPassphrase": "",
                    "addEntropyStore": ""
                },
                "net": {
                    "syncInfo": "",
                    "peers": ""
                },
                "onroad": {
                    "getOnroadBlocksByAddress": "",
                    "getAccountOnroadInfo": "",
                    "listWorkingAutoReceiveWorker": "",
                    "startAutoReceive": "",
                    "stopAutoReceive": ""
                },
                "contract": {
                    "getCreateContractToAddress": ""
                },
                "pledge": {
                    "getPledgeData": "",
                    "getCancelPledgeData": "",
                    "getPledgeQuota": "",
                    "getPledgeList": ""
                },
                "register": {
                    "getSignDataForRegister": "",
                    "getRegisterData": "",
                    "getCancelRegisterData": "",
                    "getRewardData": "",
                    "getUpdateRegistrationData": ""
                },
                "vote": {
                    "getVoteData": "",
                    "getCancelVoteData": ""
                },
                "mintage": {
                    "getMintageData": "",
                    "getMintageCancelPledgeData": ""
                },
                "consensusGroup": {
                    "getConditionRegisterOfPledge": "",
                    "getConditionVoteOfDefault": "",
                    "getConditionVoteOfKeepToken": "",
                    "getCreateConsensusGroupData": "",
                    "getCancelConsensusGroupData": "",
                    "getReCreateConsensusGroupData": ""
                },
                "ledger": {
                    "getBlocksByAccAddr": "",
                    "getAccountByAccAddr": "",
                    "getLatestSnapshotChainHash": "",
                    "getLatestBlock": "",
                    "getTokenMintage": "",
                    "getBlocksByHash": "",
                    "getSnapshotChainHeight": "",
                    "getFittestSnapshotHash": ""
                },
                "tx": {
                    "sendRawTx": ""
                }                //^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ 在以上地方修改^^^^^^^^^^^ 
            };
            exports.default = _default;

        }, {}]
    }, {}, ["docs"]);
