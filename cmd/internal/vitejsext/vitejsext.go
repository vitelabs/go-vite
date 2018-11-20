// package vitejsext contains gvite specific vite.js extensions.
package vitejsext

var Modules = map[string]string{
	"wallet":    Wallet_JS,
	"p2p":       P2P_JS,
	"onroad":    OnRoad_JS,
	"contracts": Contracts_JS,
	"ledger":    Ledger_JS,
}

const Wallet_JS = `
web3._extend({
	property: 'wallet',
	methods: [
		new web3._extend.Method({
			name: 'listAddress',
			getter: 'wallet_listAddress'
		}),
		new web3._extend.Property({
			name: 'newAddress',
			call: 'wallet_newAddress',
			params: 1
		}),
		new web3._extend.Method({
			name: 'status',
			getter: 'wallet_status'
		}),
		new web3._extend.Method({
			name: 'unlockAddress',
			call: 'wallet_unlockAddress',
			params: 3
		}),
		new web3._extend.Method({
			name: 'lockAddress',
			call: 'wallet_lockAddress',
			params: 1
		}),
		new web3._extend.Method({
			name: 'reloadAndFixAddressFile',
			getter: 'wallet_reloadAndFixAddressFile'
		}),
		new web3._extend.Method({
			name: 'isMayValidKeystoreFile',
			call: 'wallet_isMayValidKeystoreFile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getDataDir',
			getter: 'wallet_getDataDir'
		}),
		new web3._extend.Method({
			name: 'createTxWithPassphrase',
			call: 'wallet_createTxWithPassphrase',
			params: 6
		})
	]
});
`
const P2P_JS = `
web3._extend({
	property: 'p2p',
	methods: [
		new web3._extend.Method({
			name: 'networkAvailable',
			getter: 'p2p_networkAvailable'
		}),
		new web3._extend.Property({
			name: 'peersCount',
			getter: 'p2p_peersCount'
		})
	]
});
`
const OnRoad_JS = `
web3._extend({
	property: 'onroad',
	methods: [
		new web3._extend.Method({
			name: 'networkAvailable',
			call: 'onroad_getOnroadBlocksByAddress',
			params: 3
		}),
		new web3._extend.Property({
			name: 'getAccountOnroadInfo',
			call: 'onroad_getAccountOnroadInfo',
			params: 1
		}),
		new web3._extend.Method({
			name: 'listWorkingAutoReceiveWorker',
			getter: 'onroad_listWorkingAutoReceiveWorker'
		}),
		new web3._extend.Method({
			name: 'startAutoReceive',
			call: 'onroad_startAutoReceive',
			params: 2
		}),
		new web3._extend.Method({
			name: 'stopAutoReceive',
			call: 'onroad_stopAutoReceive',
			params: 1
		}),
	]
});
`

const Ledger_JS = `
web3._extend({
	property: 'ledger',
	methods: [
	]
});
`

const Contracts_JS = `
web3._extend({
	property: 'contracts',
	methods: [
		new web3._extend.Method({
			name: 'getPledgeData',
			call: 'contracts_getPledgeData',
			params: 3
		}),
		new web3._extend.Property({
			name: 'getCancelPledgeData',
			call: 'contracts_getCancelPledgeData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getMintageData',
			getter: 'contracts_getMintageData'
		}),
		new web3._extend.Method({
			name: 'getMintageCancelPledgeData',
			call: 'contracts_getMintageCancelPledgeData',
			params: 2
		}),
		new web3._extend.Method({
			name: 'getCreateContractToAddress',
			call: 'contracts_getCreateContractToAddress',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRegisterData',
			call: 'contracts_getRegisterData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getCancelRegisterData',
			call: 'contracts_getCancelRegisterData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRewardData',
			call: 'contracts_getRewardData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getUpdateRegistrationData',
			call: 'contracts_getUpdateRegistrationData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getVoteData',
			call: 'contracts_getVoteData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getCancelVoteData',
			call: 'contracts_getCancelVoteData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getConditionRegisterOfPledge',
			call: 'contracts_getConditionRegisterOfPledge',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getConditionVoteOfDefault',
			call: 'contracts_getConditionVoteOfDefault',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getConditionVoteOfKeepToken',
			call: 'contracts_getConditionVoteOfKeepToken',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getCreateConsensusGroupData',
			call: 'contracts_getCreateConsensusGroupData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getCancelConsensusGroupData',
			call: 'contracts_getCancelConsensusGroupData',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getReCreateConsensusGroupData',
			call: 'contracts_getReCreateConsensusGroupData',
			params: 1
		}),
	]
});
`
