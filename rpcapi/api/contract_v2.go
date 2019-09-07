package api

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"sort"
	"time"
)

type ContractApi struct {
	chain chain.Chain
	vite  *vite.Vite
	cs    consensus.Consensus
	log   log15.Logger
}

func NewContractApi(vite *vite.Vite) *ContractApi {
	return &ContractApi{
		chain: vite.Chain(),
		vite:  vite,
		cs:    vite.Consensus(),
		log:   log15.New("module", "rpc_api/contract_api"),
	}
}

func (c ContractApi) String() string {
	return "ContractApi"
}

func (c *ContractApi) CreateContractAddress(address types.Address, height string, previousHash types.Hash) (*types.Address, error) {
	h, err := StringToUint64(height)
	if err != nil {
		return nil, err
	}
	addr := util.NewContractAddress(address, h, previousHash)
	return &addr, nil
}

type ContractInfo struct {
	Code            []byte    `json:"code"`
	Gid             types.Gid `json:"gid"`
	ConfirmTime     uint8     `json:"confirmTime"` // Deprecated: use responseLatency instead
	ResponseLatency uint8     `json:"responseLatency"`
	SeedCount       uint8     `json:"seedCount"` // Deprecated: use randomness instead
	RandomDegree    uint8     `json:"randomDegree"`
	QuotaRatio      uint8     `json:"quotaRatio"` // Deprecated: use quotaMultiplier instead
	QuotaMultiplier uint8     `json:"quotaMultiplier"`
}

func (c *ContractApi) GetContractInfo(addr types.Address) (*ContractInfo, error) {
	code, err := c.chain.GetContractCode(addr)
	if err != nil {
		return nil, err
	}
	meta, err := c.chain.GetContractMeta(addr)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, nil
	}
	return &ContractInfo{
		Code:            code,
		Gid:             meta.Gid,
		ConfirmTime:     meta.SendConfirmedTimes,
		ResponseLatency: meta.SendConfirmedTimes,
		SeedCount:       meta.SeedConfirmedTimes,
		RandomDegree:    meta.SeedConfirmedTimes,
		QuotaRatio:      meta.QuotaRatio,
		QuotaMultiplier: meta.QuotaRatio,
	}, nil
}

type CallOffChainMethodParam struct {
	SelfAddr          types.Address  `json:"selfAddr"` // Deprecated: use address field instead
	Addr              *types.Address `json:"address"`
	OffChainCode      string         `json:"offchainCode"`      // Deprecated: use code field instead
	OffChainCodeBytes []byte         `json:"offchainCodeBytes"` // Deprecated: use code field instead
	Code              []byte         `json:"code"`
	Data              []byte         `json:"data"`
}

func (c *ContractApi) CallOffChainMethod(param CallOffChainMethodParam) ([]byte, error) {
	if param.Addr != nil {
		param.SelfAddr = *param.Addr
	}
	prevHash, err := getPrevBlockHash(c.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(c.chain, &param.SelfAddr, &c.chain.GetLatestSnapshotBlock().Hash, prevHash)
	if err != nil {
		return nil, err
	}
	var codeBytes []byte
	if len(param.OffChainCode) > 0 {
		codeBytes, err = hex.DecodeString(param.OffChainCode)
		if err != nil {
			return nil, err
		}
	} else if len(param.OffChainCodeBytes) > 0 {
		codeBytes = param.OffChainCodeBytes
	} else {
		codeBytes = param.Code
	}
	return vm.NewVM(nil).OffChainReader(db, codeBytes, param.Data)
}

type QuotaInfo struct {
	CurrentQuota string  `json:"currentQuota"`
	MaxQuota     string  `json:"maxQuota"`
	StakeAmount  *string `json:"stakeAmount"`
}

func (p *ContractApi) GetQuotaByAccount(addr types.Address) (*QuotaInfo, error) {
	amount, q, err := p.chain.GetPledgeQuota(addr)
	if err != nil {
		return nil, err
	}
	return &QuotaInfo{
		CurrentQuota: Uint64ToString(q.Current()),
		MaxQuota:     Uint64ToString(q.PledgeQuotaPerSnapshotBlock() * util.OneRound),
		StakeAmount:  bigIntToString(amount)}, nil
}

type StakeInfoList struct {
	StakeAmount string       `json:"totalStakeAmount"`
	Count       int          `json:"totalStakeCount"`
	StakeList   []*StakeInfo `json:"stakeList"`
}

type StakeInfo struct {
	Amount           string        `json:"stakeAmount"`
	Beneficiary      types.Address `json:"beneficiary"`
	ExpirationHeight string        `json:"expirationHeight"`
	ExpirationTime   int64         `json:"expirationTime"`
	IsDelegated      bool          `json:"isDelegated"`
	DelegateAddress  types.Address `json:"delegateAddress"`
	StakeAddress     types.Address `json:"stakeAddress"`
	Bid              uint8         `json:"bid"`
}

func NewStakeInfo(addr types.Address, info *types.PledgeInfo, snapshotBlock *ledger.SnapshotBlock) *StakeInfo {
	return &StakeInfo{
		*bigIntToString(info.Amount),
		info.BeneficialAddr,
		Uint64ToString(info.WithdrawHeight),
		getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
		info.Agent,
		info.AgentAddress,
		addr,
		info.Bid}
}

func (p *ContractApi) GetStakeList(address types.Address, pageIndex int, pageSize int) (*StakeInfoList, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	list, amount, err := abi.GetPledgeInfoList(db, address)
	if err != nil {
		return nil, err
	}
	sort.Sort(byWithdrawHeight(list))
	startHeight, endHeight := pageIndex*pageSize, (pageIndex+1)*pageSize
	if startHeight >= len(list) {
		return &StakeInfoList{*bigIntToString(amount), len(list), []*StakeInfo{}}, nil
	}
	if endHeight > len(list) {
		endHeight = len(list)
	}
	targetList := make([]*StakeInfo, endHeight-startHeight)
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	for i, info := range list[startHeight:endHeight] {
		targetList[i] = NewStakeInfo(address, info, snapshotBlock)
	}
	return &StakeInfoList{*bigIntToString(amount), len(list), targetList}, nil
}

type StakeInfoListBySearchKey struct {
	StakingInfoList []*StakeInfo `json:"stakeList"`
	LastKey         string       `json:"lastSearchKey"`
}

func (p *ContractApi) GetStakeListBySearchKey(snapshotHash types.Hash, lastKey string, size uint64) (*StakeInfoListBySearchKey, error) {
	lastKeyBytes, err := hex.DecodeString(lastKey)
	if err != nil {
		return nil, err
	}
	list, lastKeyBytes, err := p.chain.GetPledgeListByPage(snapshotHash, lastKeyBytes, size)
	if err != nil {
		return nil, err
	}
	targetList := make([]*StakeInfo, len(list))
	snapshotBlock := p.chain.GetLatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	for i, info := range list {
		targetList[i] = NewStakeInfo(info.PledgeAddress, info, snapshotBlock)
	}
	return &StakeInfoListBySearchKey{targetList, hex.EncodeToString(lastKeyBytes)}, nil
}

func (p *ContractApi) GetRequiredStakeAmount(qStr string) (*string, error) {
	q, err := StringToUint64(qStr)
	if err != nil {
		return nil, err
	}
	amount, err := quota.CalcPledgeAmountByQuota(q)
	if err != nil {
		return nil, err
	}
	return bigIntToString(amount), nil
}

type StakeQueryParams struct {
	StakeAddress    types.Address `json:"stakeAddress"`
	DelegateAddress types.Address `json:"delegateAddress"`
	BeneficialAddr  types.Address `json:"beneficiary"`
	Bid             uint8         `json:"bid"`
}

func (p *ContractApi) GetDelegatedStakeInfo(params StakeQueryParams) (*StakeInfo, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	info, err := abi.GetPledgeInfo(db, params.StakeAddress, params.BeneficialAddr, params.DelegateAddress, true, params.Bid)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return NewStakeInfo(params.StakeAddress, info, snapshotBlock), nil
}

type SBPInfo struct {
	Name           string        `json:"name"`
	NodeAddr       types.Address `json:"blockProducingAddress"`
	PledgeAddr     types.Address `json:"stakeAddress"`
	PledgeAmount   string        `json:"stakeAmount"`
	WithdrawHeight string        `json:"expirationHeight"`
	WithdrawTime   int64         `json:"expirationTime"`
	CancelTime     int64         `json:"revokeTime"`
}

func newSBPInfo(info *types.Registration, sb *ledger.SnapshotBlock) *SBPInfo {
	return &SBPInfo{
		Name:           info.Name,
		NodeAddr:       info.NodeAddr,
		PledgeAddr:     info.PledgeAddr,
		PledgeAmount:   *bigIntToString(info.Amount),
		WithdrawHeight: Uint64ToString(info.WithdrawHeight),
		WithdrawTime:   getWithdrawTime(sb.Timestamp, sb.Height, info.WithdrawHeight),
		CancelTime:     info.CancelTime,
	}
}

func (r *ContractApi) GetSBPList(stakeAddress types.Address) ([]*SBPInfo, error) {
	db, err := getVmDb(r.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	list, err := abi.GetRegistrationList(db, types.SNAPSHOT_GID, stakeAddress)
	if err != nil {
		return nil, err
	}
	targetList := make([]*SBPInfo, len(list))
	if len(list) > 0 {
		sort.Sort(byRegistrationWithdrawHeight(list))
		for i, info := range list {
			targetList[i] = newSBPInfo(info, sb)
		}
	}
	return targetList, nil
}

type SBPReward struct {
	BlockReward      string `json:"blockProducingReward"`
	VoteReward       string `json:"votingReward"`
	TotalReward      string `json:"totalReward"`
	BlockNum         string `json:"producedBlocks"`
	ExpectedBlockNum string `json:"targetBlocks"`
	Drained          bool   `json:"allRewardWithdrawed"`
}

func ToSBPReward(source *contracts.Reward) *SBPReward {
	if source == nil {
		return &SBPReward{TotalReward: "0",
			VoteReward:       "0",
			BlockReward:      "0",
			BlockNum:         "0",
			ExpectedBlockNum: "0"}
	} else {
		return &SBPReward{TotalReward: *bigIntToString(source.TotalReward),
			VoteReward:       *bigIntToString(source.VoteReward),
			BlockReward:      *bigIntToString(source.BlockReward),
			BlockNum:         Uint64ToString(source.BlockNum),
			ExpectedBlockNum: Uint64ToString(source.ExpectedBlockNum)}
	}
}
func (r *ContractApi) GetSBPRewardPendingWithdrawal(name string) (*SBPReward, error) {
	db, err := getVmDb(r.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	info, err := abi.GetRegistration(db, types.SNAPSHOT_GID, name)
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
	_, _, reward, drained, err := contracts.CalcReward(util.NewVmConsensusReader(r.cs.SBPReader()), db, info, sb)
	if err != nil {
		return nil, err
	}
	result := ToSBPReward(reward)
	result.Drained = contracts.RewardDrained(reward, drained)
	return result, nil
}

type SBPRewardInfo struct {
	RewardMap map[string]*SBPReward `json:"rewardMap"`
	StartTime int64                 `json:"startTime"`
	EndTime   int64                 `json:"endTime"`
	Cycle     string                `json:"cycle"`
}

func (r *ContractApi) GetSBPRewardByTimestamp(timestamp int64) (*SBPRewardInfo, error) {
	db, err := getVmDb(r.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	m, index, err := contracts.CalcRewardByDay(db, util.NewVmConsensusReader(r.cs.SBPReader()), timestamp)
	if err != nil {
		return nil, err
	}
	rewardMap := make(map[string]*SBPReward, len(m))
	for name, reward := range m {
		rewardMap[name] = ToSBPReward(reward)
	}
	startTime, endTime := r.cs.SBPReader().GetDayTimeIndex().Index2Time(index)
	return &SBPRewardInfo{rewardMap, startTime.Unix(), endTime.Unix(), Uint64ToString(index)}, nil
}

func (r *ContractApi) GetSBPRewardByCycle(cycle string) (*SBPRewardInfo, error) {
	index, err := StringToUint64(cycle)
	if err != nil {
		return nil, err
	}
	db, err := getVmDb(r.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	m, err := contracts.CalcRewardByDayIndex(db, util.NewVmConsensusReader(r.cs.SBPReader()), index)
	if err != nil {
		return nil, err
	}
	rewardMap := make(map[string]*SBPReward, len(m))
	for name, reward := range m {
		rewardMap[name] = ToSBPReward(reward)
	}
	startTime, endTime := r.cs.SBPReader().GetDayTimeIndex().Index2Time(index)
	return &SBPRewardInfo{rewardMap, startTime.Unix(), endTime.Unix(), cycle}, nil
}

func (r *ContractApi) GetSBP(name string) (*SBPInfo, error) {
	db, err := getVmDb(r.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	info, err := abi.GetRegistration(db, types.SNAPSHOT_GID, name)
	if err != nil {
		return nil, err
	}
	sb, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	return newSBPInfo(info, sb), nil
}

type SBPVoteInfo struct {
	Name     string        `json:"sbpName"`
	NodeAddr types.Address `json:"blockProducingAddress"`
	VoteNum  string        `json:"votes"`
}

func (r *ContractApi) GetSBPVoteList() ([]*SBPVoteInfo, error) {
	head := r.chain.GetLatestSnapshotBlock()
	details, _, err := r.cs.API().ReadVoteMap((*head.Timestamp).Add(time.Second))
	if err != nil {
		return nil, err
	}
	var result []*SBPVoteInfo
	for _, v := range details {
		result = append(result, &SBPVoteInfo{v.Name, v.CurrentAddr, *bigIntToString(v.Balance)})
	}
	return result, nil
}

type VotedSBPInfo struct {
	Name       string `json:"blockProducerName"`
	NodeStatus uint8  `json:"status"`
	Balance    string `json:"votes"`
}

func (v *ContractApi) GetVotedSBP(addr types.Address) (*VotedSBPInfo, error) {
	db, err := getVmDb(v.chain, types.AddressConsensusGroup)
	if err != nil {
		return nil, err
	}
	voteInfo, err := abi.GetVote(db, types.SNAPSHOT_GID, addr)
	if err != nil {
		return nil, err
	}
	if voteInfo != nil {
		balance, err := v.chain.GetBalance(addr, ledger.ViteTokenId)
		if err != nil {
			return nil, err
		}
		active, err := abi.IsActiveRegistration(db, voteInfo.NodeName, types.SNAPSHOT_GID)
		if err != nil {
			return nil, err
		}
		if active {
			return &VotedSBPInfo{voteInfo.NodeName, NodeStatusActive, *bigIntToString(balance)}, nil
		} else {
			return &VotedSBPInfo{voteInfo.NodeName, NodeStatusInActive, *bigIntToString(balance)}, nil
		}
	}
	return nil, nil
}

type VoteDetail struct {
	Name            string                   `json:"blockProducerName"`
	VoteNum         string                   `json:"totalVotes"`
	CurrentAddr     types.Address            `json:"blockProducingAddress"`
	HistoryAddrList []types.Address          `json:"historyProducingAddresses"`
	VoteMap         map[types.Address]string `json:"addressVoteMap"`
}

func (v *ContractApi) GetSBPVoteDetailsByCycle(cycle string) ([]*VoteDetail, error) {
	t := time.Now()
	if len(cycle) > 0 {
		index, err := StringToUint64(cycle)
		if err != nil {
			return nil, err
		}
		_, etime := v.cs.SBPReader().GetDayTimeIndex().Index2Time(index)
		t = etime
	}
	details, _, err := v.cs.API().ReadVoteMap(t)
	if err != nil {
		return nil, err
	}
	list := make([]*VoteDetail, len(details))
	for i, detail := range details {
		voteMap := make(map[types.Address]string, len(detail.Addr))
		for k, v := range detail.Addr {
			voteMap[k] = *bigIntToString(v)
		}
		list[i] = &VoteDetail{
			Name:            detail.Name,
			VoteNum:         *bigIntToString(detail.Balance),
			CurrentAddr:     detail.CurrentAddr,
			HistoryAddrList: detail.RegisterList,
			VoteMap:         voteMap,
		}
	}
	return list, nil
}

type TokenInfoList struct {
	Count int             `json:"totalCount"`
	List  []*RpcTokenInfo `json:"tokenInfoList"`
}

type byName []*RpcTokenInfo

func (a byName) Len() int      { return len(a) }
func (a byName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool {
	if a[i].TokenName == a[j].TokenName {
		return a[i].TokenId.String() < a[j].TokenId.String()
	}
	return a[i].TokenName < a[j].TokenName
}

func (m *ContractApi) GetTokenInfoList(pageIndex int, pageSize int) (*TokenInfoList, error) {
	db, err := getVmDb(m.chain, types.AddressMintage)
	if err != nil {
		return nil, err
	}
	tokenMap, err := abi.GetTokenMap(db)
	if err != nil {
		return nil, err
	}
	listLen := len(tokenMap)
	tokenList := make([]*RpcTokenInfo, 0)
	for tokenId, tokenInfo := range tokenMap {
		tokenList = append(tokenList, RawTokenInfoToRpc(tokenInfo, tokenId))
	}
	sort.Sort(byName(tokenList))
	start, end := getRange(pageIndex, pageSize, listLen)
	return &TokenInfoList{listLen, tokenList[start:end]}, nil
}

func (m *ContractApi) GetTokenInfoById(tokenId types.TokenTypeId) (*RpcTokenInfo, error) {
	db, err := getVmDb(m.chain, types.AddressMintage)
	if err != nil {
		return nil, err
	}
	tokenInfo, err := abi.GetTokenById(db, tokenId)
	if err != nil {
		return nil, err
	}
	if tokenInfo != nil {
		return RawTokenInfoToRpc(tokenInfo, tokenId), nil
	}
	return nil, nil
}

func (m *ContractApi) GetTokenInfoListByOwner(owner types.Address) ([]*RpcTokenInfo, error) {
	db, err := getVmDb(m.chain, types.AddressMintage)
	if err != nil {
		return nil, err
	}
	tokenMap, err := abi.GetTokenMapByOwner(db, owner)
	if err != nil {
		return nil, err
	}
	tokenList := make([]*RpcTokenInfo, 0)
	for tokenId, tokenInfo := range tokenMap {
		tokenList = append(tokenList, RawTokenInfoToRpc(tokenInfo, tokenId))
	}
	return checkGenesisToken(db, owner, m.vite.Config().MintageInfo.TokenInfoMap, tokenList)
}
