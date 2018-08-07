package impl

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/rpc/api"
	"time"
)

func NewMockLedger() api.LedgerApi {
	return &MockLedgerImpl{}
}

type MockLedgerImpl struct {
}

func (MockLedgerImpl) String() string {
	return "MockLedgerImpl"
}

func (MockLedgerImpl) CreateTxWithPassphrase(params *api.SendTxParms, reply *string) error {
	p, _ := json.Marshal(params)
	log.Debug(string(p))

	time.Sleep(10 * time.Second)
	return nil
}

func (MockLedgerImpl) GetBlocksByAccAddr(params *api.GetBlocksParams, reply *string) error {
	log.Debug("GetBlocksByAccAddr")
	p, _ := json.Marshal(params)
	log.Debug(string(p))

	s := []api.SimpleBlock{
		{
			Timestamp: uint64(time.Now().Unix()),
			Amount:    "123",
			FromAddr:  "vite_2c760b7163dcac330a32787a46779b56f6e6c6ffe68112090e",
			ToAddr:    "vite_8bca915b96022801d3f809bdb9133077c22dd640df06fced28",
			Status:    0,
			Hash:      "111",
		},
		{
			Timestamp: uint64(time.Now().Unix()),
			Amount:    "333",
			FromAddr:  "vite_b7d95cc00fd89f8f94cda547a9ec686ae0c3714921e1867dd9",
			ToAddr:    "vite_d308c5e857e2fa537be50f4aaa71abeb15155de930c6eb175d",
			Status:    1,
			Hash:      "222",
		},
		{
			Timestamp: uint64(time.Now().Unix()),
			Amount:    "666",
			FromAddr:  "vite_b7d95cc00fd89f8f94cda547a9ec686ae0c3714921e1867dd9",
			ToAddr:    "vite_8bca915b96022801d3f809bdb9133077c22dd640df06fced28",
			Status:    2,
			Hash:      "333",
		},
	}
	return easyJsonReturn(s, reply)
}

func (MockLedgerImpl) GetUnconfirmedBlocksByAccAddr(params *api.GetBlocksParams, reply *string) error {
	log.Debug("GetUnconfirmedBlocksByAccAddr")

	p, _ := json.Marshal(params)
	log.Debug(string(p))

	s := []api.SimpleBlock{
		{
			Timestamp: uint64(time.Now().Unix()),
			Amount:    "123",
			FromAddr:  "vite_2c760b7163dcac330a32787a46779b56f6e6c6ffe68112090e",
			ToAddr:    "vite_8bca915b96022801d3f809bdb9133077c22dd640df06fced28",
			Status:    0,
			Hash:      "111",
		},
		{
			Timestamp: uint64(time.Now().Unix()),
			Amount:    "333",
			FromAddr:  "vite_b7d95cc00fd89f8f94cda547a9ec686ae0c3714921e1867dd9",
			ToAddr:    "vite_d308c5e857e2fa537be50f4aaa71abeb15155de930c6eb175d",
			Status:    1,
			Hash:      "222",
		},
		{
			Timestamp: uint64(time.Now().Unix()),
			Amount:    "666",
			FromAddr:  "vite_b7d95cc00fd89f8f94cda547a9ec686ae0c3714921e1867dd9",
			ToAddr:    "vite_8bca915b96022801d3f809bdb9133077c22dd640df06fced28",
			Status:    2,
			Hash:      "333",
		},
	}

	return easyJsonReturn(s, reply)
}

func (MockLedgerImpl) GetAccountByAccAddr(addr []string, reply *string) error {
	log.Debug("GetAccountByAccAddr")
	return easyJsonReturn(api.GetAccountResponse{

		Addr: "vite_b7d95cc00fd89f8f94cda547a9ec686ae0c3714921e1867dd9 ",
		BalanceInfos: []api.BalanceInfo{
			{
				TokenSymbol: "vite",
				TokenName:   "vite",
				TokenTypeId: "tti_133",
				Balance:     "111",
			},
			{
				TokenSymbol: "tt",
				TokenName:   "tt",
				TokenTypeId: "tti_99",
				Balance:     "123",
			},
		},
		BlockHeight: "123",
	}, reply)
}

func (MockLedgerImpl) GetUnconfirmedInfo(addr []string, reply *string) error {
	log.Debug("GetUnconfirmedInfo")
	return easyJsonReturn(api.GetUnconfirmedInfoResponse{
		Addr: "vite_8bca915b96022801d3f809bdb9133077c22dd640df06fced28",
		BalanceInfos: []api.BalanceInfo{
			{
				TokenSymbol: "vite",
				TokenName:   "vite",
				TokenTypeId: "tti_133",
				Balance:     "666",
			},
			{
				TokenSymbol: "tt",
				TokenName:   "tt",
				TokenTypeId: "tti_99",
				Balance:     "888",
			},
		},
		UnConfirmedBlocksLen: "0",
	}, reply)
}

func (MockLedgerImpl) GetInitSyncInfo(noop interface{}, reply *string) error {
	log.Debug("GetInitSyncInfo")
	return easyJsonReturn(api.InitSyncResponse{
		StartHeight:   "100",
		TargetHeight:  "200",
		CurrentHeight: "200",
	}, reply)
}
