package api

import (
	"sort"
	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
)

type RegisterApi struct {
	chain chain.Chain
	cs    consensus.Consensus
	log   log15.Logger
}

func NewRegisterApi(vite *vite.Vite) *RegisterApi {
	return &RegisterApi{
		chain: vite.Chain(),
		cs:    vite.Consensus(),
		log:   log15.New("module", "rpc_api/register_api"),
	}
}

func (r RegisterApi) String() string {
	return "RegisterApi"
}

// Private
func (r *RegisterApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameRegister, gid, name, nodeAddr)
}

// Private
func (r *RegisterApi) GetCancelRegisterData(gid types.Gid, name string) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameRevoke, gid, name)
}

// Private
func (r *RegisterApi) GetRewardData(gid types.Gid, name string, beneficialAddr types.Address) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameWithdrawReward, gid, name, beneficialAddr)
}

// Private
func (r *RegisterApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameUpdateBlockProducingAddress, gid, name, nodeAddr)
}

type RegistrationInfo struct {
	Name                  string        `json:"name"`
	NodeAddr              types.Address `json:"nodeAddr"`
	PledgeAddr            types.Address `json:"pledgeAddr"`
	RewardWithdrawAddress types.Address `json:"rewardWithdrawAddress"`
	PledgeAmount          string        `json:"pledgeAmount"`
	WithdrawHeight        string        `json:"withdrawHeight"`
	WithdrawTime          int64         `json:"withdrawTime"`
	CancelTime            int64         `json:"cancelTime"`
}

type byRegistrationExpirationHeight []*types.Registration

func (a byRegistrationExpirationHeight) Len() int      { return len(a) }
func (a byRegistrationExpirationHeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byRegistrationExpirationHeight) Less(i, j int) bool {
	if a[i].ExpirationHeight == a[j].ExpirationHeight {
		if a[i].RevokeTime == a[j].RevokeTime {
			return a[i].Name > a[j].Name
		} else {
			return a[i].RevokeTime > a[j].RevokeTime
		}
	}
	return a[i].ExpirationHeight > a[j].ExpirationHeight
}

// Deprecated: use contract_getSBPList
func (r *RegisterApi) GetRegistrationList(gid types.Gid, pledgeAddr types.Address) ([]*RegistrationInfo, error) {
	db, err := getVmDb(r.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	list, err := abi.GetRegistrationList(db, gid, pledgeAddr)
	if err != nil {
		return nil, err
	}
	targetList := make([]*RegistrationInfo, len(list))
	if len(list) > 0 {
		sort.Sort(byRegistrationExpirationHeight(list))
		for i, info := range list {
			targetList[i] = &RegistrationInfo{
				Name:                  info.Name,
				NodeAddr:              info.BlockProducingAddress,
				PledgeAddr:            info.StakeAddress,
				RewardWithdrawAddress: info.RewardWithdrawAddress,
				PledgeAmount:          *bigIntToString(info.Amount),
				WithdrawHeight:        Uint64ToString(info.ExpirationHeight),
				WithdrawTime:          getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.ExpirationHeight),
				CancelTime:            info.RevokeTime,
			}
		}
	}
	return targetList, nil
}

type Reward struct {
	BlockReward      string `json:"blockReward"`
	VoteReward       string `json:"voteReward"`
	TotalReward      string `json:"totalReward"`
	BlockNum         string `json:"blockNum"`
	ExpectedBlockNum string `json:"expectedBlockNum"`
	Drained          bool   `json:"drained"`
}

func ToReward(source *contracts.Reward) *Reward {
	if source == nil {
		return &Reward{TotalReward: "0",
			VoteReward:       "0",
			BlockReward:      "0",
			BlockNum:         "0",
			ExpectedBlockNum: "0"}
	} else {
		return &Reward{TotalReward: *bigIntToString(source.TotalReward),
			VoteReward:       *bigIntToString(source.VoteReward),
			BlockReward:      *bigIntToString(source.BlockReward),
			BlockNum:         Uint64ToString(source.BlockNum),
			ExpectedBlockNum: Uint64ToString(source.ExpectedBlockNum)}
	}
}

// Deprecated: use contract_getSBPRewardPendingWithdrawal
func (r *RegisterApi) GetAvailableReward(gid types.Gid, name string) (*Reward, error) {
	db, err := getVmDb(r.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	info, err := abi.GetRegistration(db, gid, name)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	_, _, reward, drained, err := contracts.CalcReward(util.NewVMConsensusReader(r.cs.SBPReader()), db, info, sb)
	if err != nil {
		return nil, err
	}
	result := ToReward(reward)
	result.Drained = contracts.RewardDrained(reward, drained)
	return result, nil
}

// Deprecated: use contract_getSBPRewardByTimestamp instead
func (r *RegisterApi) GetRewardByDay(gid types.Gid, timestamp int64) (map[string]*Reward, error) {
	db, err := getVmDb(r.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	m, _, err := contracts.CalcRewardByCycle(db, util.NewVMConsensusReader(r.cs.SBPReader()), timestamp)
	if err != nil {
		return nil, err
	}
	rewardMap := make(map[string]*Reward, len(m))
	for name, reward := range m {
		rewardMap[name] = ToReward(reward)
	}
	return rewardMap, nil
}

type RewardInfo struct {
	RewardMap map[string]*Reward `json:"rewardMap"`
	StartTime int64              `json:"startTime"`
	EndTime   int64              `json:"endTime"`
}

// Deprecated: use contract_getSBPRewardByCycle instead
func (r *RegisterApi) GetRewardByIndex(gid types.Gid, indexStr string) (*RewardInfo, error) {
	index, err := StringToUint64(indexStr)
	if err != nil {
		return nil, err
	}
	db, err := getVmDb(r.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	m, err := contracts.CalcRewardByIndex(db, util.NewVMConsensusReader(r.cs.SBPReader()), index)
	if err != nil {
		return nil, err
	}
	rewardMap := make(map[string]*Reward, len(m))
	for name, reward := range m {
		rewardMap[name] = ToReward(reward)
	}
	startTime, endTime := r.cs.SBPReader().GetDayTimeIndex().Index2Time(index)
	return &RewardInfo{rewardMap, startTime.Unix(), endTime.Unix()}, nil
}

// Deprecated: use contract_getSBP instead
func (r *RegisterApi) GetRegistration(name string, gid types.Gid) (*types.Registration, error) {
	db, err := getVmDb(r.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	return abi.GetRegistration(db, gid, name)
}

type RegistParam struct {
	Name string     `json:"name"`
	Gid  *types.Gid `json:"gid"`
}

// Deprecated
func (r *RegisterApi) GetRegisterPledgeAddrList(paramList []*RegistParam) ([]*types.Address, error) {
	if len(paramList) == 0 {
		return nil, nil
	}
	db, err := getVmDb(r.chain, types.AddressGovernance)
	if err != nil {
		return nil, err
	}
	addrList := make([]*types.Address, len(paramList))
	for k, v := range paramList {
		var r *types.Registration
		if v.Gid == nil || *v.Gid == types.DELEGATE_GID {
			if r, err = abi.GetRegistration(db, types.SNAPSHOT_GID, v.Name); err != nil {
				return nil, err
			}
		} else {
			if r, err = abi.GetRegistration(db, *v.Gid, v.Name); err != nil {
				return nil, err
			}
		}
		if r != nil {
			addrList[k] = &r.StakeAddress
		}
	}
	return addrList, nil
}

type CandidateInfo struct {
	Name     string        `json:"name"`
	NodeAddr types.Address `json:"nodeAddr"`
	VoteNum  string        `json:"voteNum"`
}

// Deprecated: usecontract_getSBPVoteList instead
func (r *RegisterApi) GetCandidateList() ([]*CandidateInfo, error) {
	head := r.chain.GetLatestSnapshotBlock()
	details, _, err := r.cs.API().ReadVoteMap((*head.Timestamp).Add(time.Second))
	if err != nil {
		return nil, err
	}
	var result []*CandidateInfo
	for _, v := range details {
		result = append(result, &CandidateInfo{v.Name, v.CurrentAddr, *bigIntToString(v.Balance)})
	}
	return result, nil
}
