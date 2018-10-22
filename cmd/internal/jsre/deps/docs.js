require = (function () {
    function r(e, n, t) {
        function o(i, f) {
            if (!n[i]) {
                if (!e[i]) {
                    console.log('99999',e)
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
                helpContent: '任意api下使用api.help即可查看使用说明文档',
                //vite.help
                wallet: {
                    helpContent: '该命名空间包括钱包相关的api',
                    listAddress: 'vite.wallet_listAddress(<none>) @return <Array>HexAddress-string ',
                    newAddress: 'vite.wallet_newAddress(<password-string>) @return address-string',
                    status: '',
                    unlockAddress: '',
                    lockAddress: '',
                    reloadAndFixAddressFile: '',
                    isMayValidKeystoreFile: '',
                    getDataDir: '',
                    createTxWithPassphrase: ''
                },
                p2p: {
                    networkAvailable: '',
                    peersCount: ''
                },
                ledger: {
                    getBlocksByAccAddr: '',
                    getAccountByAccAddr: '',
                    getLatestSnapshotChainHash: '',
                    getLatestBlock: '',
                    getTokenMintage: '',
                    getBlocksByHash: '',
                    getSnapshotChainHeight: ''
                },
                onroad: {
                    getOnroadBlocksByAddress: '',
                    getAccountOnroadInfo: '',
                    listWorkingAutoReceiveWorker: '',
                    startAutoReceive: '',
                    stopAutoReceive: ''
                },
                contracts: {
                    getPledgeData: '',
                    getCancelPledgeData: '',
                    getMintageData: '',
                    getMintageCancelPledgeData: '',
                    getCreateContractToAddress: '',
                    getRegisterData: '',
                    getCancelRegisterData: '',
                    getRewardData: '',
                    getUpdateRegistrationData: '',
                    getVoteData: '',
                    getCancelVoteData: '',
                    getConditionRegisterOfPledge: '',
                    getConditionVoteOfDefault: '',
                    getConditionVoteOfKeepToken: '',
                    getCreateConsensusGroupData: '',
                    getCancelConsensusGroupData: '',
                    getReCreateConsensusGroupData: ''
                },
                pow: {
                    getPowNonce: ''
                },
                tx: {
                    sendRawTx: ''
                }
     //^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ 在以上地方修改^^^^^^^^^^^ 
            };
            exports.default = _default;

        }, {}]
    }, {}, ["docs"]);
