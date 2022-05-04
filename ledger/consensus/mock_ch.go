// Code generated by MockGen. DO NOT EDIT.
// Source: ledger/consensus/chain_rw.go

// Package consensus is a generated GoMock package.
package consensus

import (
	big "math/big"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	leveldb "github.com/syndtr/goleveldb/leveldb"
	types "github.com/vitelabs/go-vite/v2/common/types"
	interfaces "github.com/vitelabs/go-vite/v2/interfaces"
	core "github.com/vitelabs/go-vite/v2/interfaces/core"
)

// MockChain is a mock of Chain interface.
type MockChain struct {
	ctrl     *gomock.Controller
	recorder *MockChainMockRecorder
}

// MockChainMockRecorder is the mock recorder for MockChain.
type MockChainMockRecorder struct {
	mock *MockChain
}

// NewMockChain creates a new mock instance.
func NewMockChain(ctrl *gomock.Controller) *MockChain {
	mock := &MockChain{ctrl: ctrl}
	mock.recorder = &MockChainMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockChain) EXPECT() *MockChainMockRecorder {
	return m.recorder
}

// GetAllRegisterList mocks base method.
func (m *MockChain) GetAllRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllRegisterList", snapshotHash, gid)
	ret0, _ := ret[0].([]*types.Registration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllRegisterList indicates an expected call of GetAllRegisterList.
func (mr *MockChainMockRecorder) GetAllRegisterList(snapshotHash, gid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllRegisterList", reflect.TypeOf((*MockChain)(nil).GetAllRegisterList), snapshotHash, gid)
}

// GetConfirmedBalanceList mocks base method.
func (m *MockChain) GetConfirmedBalanceList(addrList []types.Address, tokenID types.TokenTypeId, sbHash types.Hash) (map[types.Address]*big.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConfirmedBalanceList", addrList, tokenID, sbHash)
	ret0, _ := ret[0].(map[types.Address]*big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConfirmedBalanceList indicates an expected call of GetConfirmedBalanceList.
func (mr *MockChainMockRecorder) GetConfirmedBalanceList(addrList, tokenID, sbHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConfirmedBalanceList", reflect.TypeOf((*MockChain)(nil).GetConfirmedBalanceList), addrList, tokenID, sbHash)
}

// GetConsensusGroupList mocks base method.
func (m *MockChain) GetConsensusGroupList(snapshotHash types.Hash) ([]*types.ConsensusGroupInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConsensusGroupList", snapshotHash)
	ret0, _ := ret[0].([]*types.ConsensusGroupInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetConsensusGroupList indicates an expected call of GetConsensusGroupList.
func (mr *MockChainMockRecorder) GetConsensusGroupList(snapshotHash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConsensusGroupList", reflect.TypeOf((*MockChain)(nil).GetConsensusGroupList), snapshotHash)
}

// GetContractMeta mocks base method.
func (m *MockChain) GetContractMeta(contractAddress types.Address) (*core.ContractMeta, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContractMeta", contractAddress)
	ret0, _ := ret[0].(*core.ContractMeta)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContractMeta indicates an expected call of GetContractMeta.
func (mr *MockChainMockRecorder) GetContractMeta(contractAddress interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContractMeta", reflect.TypeOf((*MockChain)(nil).GetContractMeta), contractAddress)
}

// GetGenesisSnapshotBlock mocks base method.
func (m *MockChain) GetGenesisSnapshotBlock() *core.SnapshotBlock {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGenesisSnapshotBlock")
	ret0, _ := ret[0].(*core.SnapshotBlock)
	return ret0
}

// GetGenesisSnapshotBlock indicates an expected call of GetGenesisSnapshotBlock.
func (mr *MockChainMockRecorder) GetGenesisSnapshotBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGenesisSnapshotBlock", reflect.TypeOf((*MockChain)(nil).GetGenesisSnapshotBlock))
}

// GetLastUnpublishedSeedSnapshotHeader mocks base method.
func (m *MockChain) GetLastUnpublishedSeedSnapshotHeader(producer types.Address, beforeTime time.Time) (*core.SnapshotBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastUnpublishedSeedSnapshotHeader", producer, beforeTime)
	ret0, _ := ret[0].(*core.SnapshotBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLastUnpublishedSeedSnapshotHeader indicates an expected call of GetLastUnpublishedSeedSnapshotHeader.
func (mr *MockChainMockRecorder) GetLastUnpublishedSeedSnapshotHeader(producer, beforeTime interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastUnpublishedSeedSnapshotHeader", reflect.TypeOf((*MockChain)(nil).GetLastUnpublishedSeedSnapshotHeader), producer, beforeTime)
}

// GetLatestSnapshotBlock mocks base method.
func (m *MockChain) GetLatestSnapshotBlock() *core.SnapshotBlock {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestSnapshotBlock")
	ret0, _ := ret[0].(*core.SnapshotBlock)
	return ret0
}

// GetLatestSnapshotBlock indicates an expected call of GetLatestSnapshotBlock.
func (mr *MockChainMockRecorder) GetLatestSnapshotBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestSnapshotBlock", reflect.TypeOf((*MockChain)(nil).GetLatestSnapshotBlock))
}

// GetRandomSeed mocks base method.
func (m *MockChain) GetRandomSeed(snapshotHash types.Hash, n int) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRandomSeed", snapshotHash, n)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetRandomSeed indicates an expected call of GetRandomSeed.
func (mr *MockChainMockRecorder) GetRandomSeed(snapshotHash, n interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRandomSeed", reflect.TypeOf((*MockChain)(nil).GetRandomSeed), snapshotHash, n)
}

// GetRegisterList mocks base method.
func (m *MockChain) GetRegisterList(snapshotHash types.Hash, gid types.Gid) ([]*types.Registration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRegisterList", snapshotHash, gid)
	ret0, _ := ret[0].([]*types.Registration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRegisterList indicates an expected call of GetRegisterList.
func (mr *MockChainMockRecorder) GetRegisterList(snapshotHash, gid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRegisterList", reflect.TypeOf((*MockChain)(nil).GetRegisterList), snapshotHash, gid)
}

// GetSnapshotBlockByHash mocks base method.
func (m *MockChain) GetSnapshotBlockByHash(hash types.Hash) (*core.SnapshotBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSnapshotBlockByHash", hash)
	ret0, _ := ret[0].(*core.SnapshotBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSnapshotBlockByHash indicates an expected call of GetSnapshotBlockByHash.
func (mr *MockChainMockRecorder) GetSnapshotBlockByHash(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSnapshotBlockByHash", reflect.TypeOf((*MockChain)(nil).GetSnapshotBlockByHash), hash)
}

// GetSnapshotBlockByHeight mocks base method.
func (m *MockChain) GetSnapshotBlockByHeight(height uint64) (*core.SnapshotBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSnapshotBlockByHeight", height)
	ret0, _ := ret[0].(*core.SnapshotBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSnapshotBlockByHeight indicates an expected call of GetSnapshotBlockByHeight.
func (mr *MockChainMockRecorder) GetSnapshotBlockByHeight(height interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSnapshotBlockByHeight", reflect.TypeOf((*MockChain)(nil).GetSnapshotBlockByHeight), height)
}

// GetSnapshotHeaderBeforeTime mocks base method.
func (m *MockChain) GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*core.SnapshotBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSnapshotHeaderBeforeTime", timestamp)
	ret0, _ := ret[0].(*core.SnapshotBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSnapshotHeaderBeforeTime indicates an expected call of GetSnapshotHeaderBeforeTime.
func (mr *MockChainMockRecorder) GetSnapshotHeaderBeforeTime(timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSnapshotHeaderBeforeTime", reflect.TypeOf((*MockChain)(nil).GetSnapshotHeaderBeforeTime), timestamp)
}

// GetSnapshotHeadersAfterOrEqualTime mocks base method.
func (m *MockChain) GetSnapshotHeadersAfterOrEqualTime(endHashHeight *core.HashHeight, startTime *time.Time, producer *types.Address) ([]*core.SnapshotBlock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSnapshotHeadersAfterOrEqualTime", endHashHeight, startTime, producer)
	ret0, _ := ret[0].([]*core.SnapshotBlock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSnapshotHeadersAfterOrEqualTime indicates an expected call of GetSnapshotHeadersAfterOrEqualTime.
func (mr *MockChainMockRecorder) GetSnapshotHeadersAfterOrEqualTime(endHashHeight, startTime, producer interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSnapshotHeadersAfterOrEqualTime", reflect.TypeOf((*MockChain)(nil).GetSnapshotHeadersAfterOrEqualTime), endHashHeight, startTime, producer)
}

// GetVoteList mocks base method.
func (m *MockChain) GetVoteList(snapshotHash types.Hash, gid types.Gid) ([]*types.VoteInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVoteList", snapshotHash, gid)
	ret0, _ := ret[0].([]*types.VoteInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVoteList indicates an expected call of GetVoteList.
func (mr *MockChainMockRecorder) GetVoteList(snapshotHash, gid interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVoteList", reflect.TypeOf((*MockChain)(nil).GetVoteList), snapshotHash, gid)
}

// IsGenesisSnapshotBlock mocks base method.
func (m *MockChain) IsGenesisSnapshotBlock(hash types.Hash) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsGenesisSnapshotBlock", hash)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsGenesisSnapshotBlock indicates an expected call of IsGenesisSnapshotBlock.
func (mr *MockChainMockRecorder) IsGenesisSnapshotBlock(hash interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsGenesisSnapshotBlock", reflect.TypeOf((*MockChain)(nil).IsGenesisSnapshotBlock), hash)
}

// NewDb mocks base method.
func (m *MockChain) NewDb(dbDir string) (*leveldb.DB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewDb", dbDir)
	ret0, _ := ret[0].(*leveldb.DB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewDb indicates an expected call of NewDb.
func (mr *MockChainMockRecorder) NewDb(dbDir interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewDb", reflect.TypeOf((*MockChain)(nil).NewDb), dbDir)
}

// Register mocks base method.
func (m *MockChain) Register(listener interfaces.EventListener) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Register", listener)
}

// Register indicates an expected call of Register.
func (mr *MockChainMockRecorder) Register(listener interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Register", reflect.TypeOf((*MockChain)(nil).Register), listener)
}

// UnRegister mocks base method.
func (m *MockChain) UnRegister(listener interfaces.EventListener) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnRegister", listener)
}

// UnRegister indicates an expected call of UnRegister.
func (mr *MockChainMockRecorder) UnRegister(listener interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnRegister", reflect.TypeOf((*MockChain)(nil).UnRegister), listener)
}
