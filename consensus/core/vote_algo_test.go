package core

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func TestAlgo_FilterVotes(t *testing.T) {

	now := time.Unix(1541640427, 0)
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		StakeAmount:            nil,
		ExpirationHeight:       0,
	})
	ag := NewAlgo(info)
	printResult(ag, 1000, 25)
	//printResult(ag, 100000, 26)
	//printResult(ag, 100000, 27)
	//printResult(ag, 100000, 28)
	//printResult(ag, 100000, 28)
	//printResult(ag, 100000, 99)
	//printResult(ag, 100000, 100)
	//printResult(ag, 100000, 101)

}

func TestAlgo_FilterVotes2(t *testing.T) {

	now := time.Unix(1541640427, 0)
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		StakeAmount:            nil,
		ExpirationHeight:       0,
	})
	ag := NewAlgo(info)
	var votes []*Vote
	for i := 0; i < 100; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(i))})
	}
	//result := make(map[string]uint64)
	hashH := &ledger.HashHeight{Height: 1}
	context := NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(0))
	actual := ag.FilterVotes(context)
	for _, v := range actual {
		print("\""+v.Name+"\"", ",")
	}
	println()
	expected := []string{"wj_99", "wj_98", "wj_97", "wj_96", "wj_95", "wj_94", "wj_92", "wj_91", "wj_90", "wj_89", "wj_88", "wj_87", "wj_86", "wj_85", "wj_84", "wj_83", "wj_82", "wj_81", "wj_80", "wj_79", "wj_78", "wj_77", "wj_75", "wj_49", "wj_27"}
	for _, v := range expected {
		print("\""+v+"\"", ",")
	}
	println()
	for i, v := range actual {
		if v.Name != expected[i] {
			t.Error("fail.")
		}
	}

}

func printResult(ag *algo, total uint64, cnt int) {
	println("-----------------------", strconv.FormatUint(total, 10), strconv.Itoa(cnt), "--------------------------------------------------")
	var votes []*Vote
	for i := 0; i < cnt; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(i))})
	}
	result := make(map[string]uint64)
	for j := uint64(0); j < total; j++ {
		hashH := &ledger.HashHeight{Height: j}
		tmp := ag.FilterVotes(NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(0)))
		for _, v := range tmp {
			result[v.Name] = result[v.Name] + 1
		}
	}
	sort.Sort(ByBalance(votes))
	for _, v := range votes {
		vv := result[v.Name]
		println(v.Name, (vv*10000)/total)
	}

	println("-------------------------------------------------------------------------")
}

func TestBalance(t *testing.T) {
	dd := []string{
		"LetsVite,vite_b9351a94769d85453d7ebd8dce868e5eaa56c7c5a23216ce92,3910400015688531595374642",
		"BayernM,vite_5809101f1839992b53802d9c0eb9c0373febcc18289c736c19,420243504974620182933216",
		"Japan_Node,vite_1e244aea1d5a25382453fd0d4f7a3a9384cc01260fc358818e,1084616846496855465416047",
		"SwissVite.org,vite_d12cfc15515e56d289ff0d17dadc10e1ceeca9129063119a80,2891022437962100440334346",
		"Vite_SBP04,vite_aadf6275ecbf07181c5c37c7d709aebf06553e470345d3f699,26753633261033966362026",
		"ZVnode,vite_7c655aca252d6f8b3788449bb39b330b32397b549b0a2d1000,4054377910458030929791466",
		"vite.bi23,vite_87db7da2d747698fd8f7e2f8f45078001a5dca83149192b21e,2768380758001284905736080",
		"Vite_SBP05,vite_c1d11e6eda9a9b80e388a38e0ac541cbc3333736233b4eaaab,181125989237227150945646",
		"Bite,vite_9d8b73d3408ccbf1f6efce89a1563660c5b1c16aa3dec131e3,57972358647551390734781",
		"Super_vite,vite_d4672160eabc76770cb0cf8c4fbc8bc3004c2a5c84829007b8,4290333627320320221813831",
		"XinShengZ_4,vite_94badf80abab06dc1cdb4d21038a6799040bb2feb154f730cb,7869906424307306398470993",
		"IC_Node,vite_e79efe2964388d40b75dc44f6b3e05956c5f4c90f8a6263f98,543836996277406988947339",
		"loopring_node,vite_785c872f703666b6088767d9809fa8aaf932d0e385f02a0c92,10361274561675094116150924",
		"Elegance.Vite,vite_adee0a2e60c50273ef54a042593628b590564ca921d95e73b8,4467906338249346654646166",
		"ZLnode,vite_1d46ef813f218a2c7fab8b4a71b6176220a9c8a7a417dee341,301935692839985501395649",
		"MoonVite,vite_af4e623adc61ab661fd2c5dbbce23b12eaadc92153c032cd64,120227737178540418263796",
		"FutureFund,vite_12bd06e0318062feaa343dcd48ed783aa902e2abbfeed6a26e,620776467304465823964058",
		"DreamFund,vite_33916a1b849611e0d97d6b335d19527b01c1421c305d4054f7,8047765009869376595407",
		"XS_Fund,vite_8370865362e739fb71615b8b33f9e394d85743093bdfaede6c,4130815931728211721495791",
		"Loopnest,vite_6e2e54c8c3f620a31a358650b3d92973c9d729b300a427ba52,4465189433271168305966543",
		"N4Q.org,vite_1dfe68a6e6f3d7970cad2d0b92d21c23fdf6c847b28d7f2f89,3563872748603725348846173",
		"Vite.Casino,vite_a1ffbed38cfcdb0f4bb0b79168c1b7970e774120d351aed4be,4871656581241977006438",
		"Beauty.Vite,vite_aa7f76480db9e4072231d52f7b7abcccc8197b217a2dd5e818,4749680570505292561953018",
		"BladeMaster,vite_b6fe7fb14765d8cc15b72816dcd1bc058eab07765edead449b,5653647909374539429507",
		"ViteTI,vite_1d656d6796c6894d18c1a64ac2b78e6f082e76808572578e8f,3267816729865509233063436",
		"V666.fun,vite_cdd075a61de71a076c94bf4fd45e5b6bc3c48ac65db912bc57,7370794440927834363402813",
		"N4Y,vite_4a04e8b6ba89f741c6945868a010ba82b22ca993ab89c6225d,3558095386062175892334531",
		"dragon_vite,vite_ea6804bf5ec9f57ab46f8c5e5457aa4c763dd78b796b9bec54,1039873585510615884535792",
		"VGATE,vite_e7a01e66d920c6c5ce82e9353ca57267f6534e357bbee63063,109861451370167240156698",
		"StarPoint,vite_3ae6649db0f94239f5256c4bcd3b10c24738fbe5d24c00f809,5160293497327321023805",
		"Chinese node,vite_dff5ee13c87ed2f205ef87d820b3cd8e97c181b1bb6781c602,4115961998543327241669872",
		"Tenzor_Node,vite_0307e7f8f7dd7a5566c8fd5ba583b6354bc2d5bf0c91f5957c,32087281701362594276312",
		"chainhedge_node,vite_0ab98c623471d3512bb6cbe06c243e977ab6e91c6abffd914e,4466450824154741380268416",
		"Korea_node,vite_4bcee903088951aeb55f16bfa5aade79cbe93d7d98216b3e86,4510714425096124153797911",
		"ViteDelegate.com,vite_286ee869c19fae545e1c6ea6d2272a74af7977351950d1a121,104703587117312067705500",
		"vite.NO1,vite_82d129931823a8a47275baf4c5c853713716800220b2ceaafd,3400832900890043082238238",
		"vite.vip,vite_1164f7ddda038466e510e26a6feba9214f6e7e2643baa4f016,2756678795977726580325816",
		"vite_cool,vite_10513d54e0c38a304ad9e7902c82277328b4df76dd31871f37,3233778617610514587549479",
		"Vite_SBP02,vite_995769283a01ba8d00258dbb5371c915df59c8657335bfb1b2,104825400902822720749786",
		"Vite_SBP03,vite_9b33ce9fc70f14407db75cfa8453680f364e6674c7cc1fb785,52243131325871362182925",
		"Vite_SBP01,vite_9065ff0e14ebf983e090cde47d59fe77d7164b576a6d2d0eda,1871844780538136069616380",
		"Pandora,vite_c38760d672f05ce5e1aef97884ecf3085b9ebe10000be47fb7,4269877874666831862566260",
		"VulcanNode,vite_62c8cbd276de2d086d48ba15b8b20ad8096021daa7a7f622cb,601299656907600443724869",
		"Vite.Singapore,vite_047c39256cb6d80eaa7aef39c8c60a39c38da69ea776a1af58,5160841607797117402278",
		"hashfin,vite_7deebe4bc6e3b3c0975b8424c935beb9deb48c8d42c769afd4,3656666000000000000000000",
		"UVA,vite_35160e6804adafffb51dd5569c35d6f9f25faea43292f3b0e7,5468919681523366434156",
		"V.Morgen,vite_165a295e214421ef1276e79990533953e901291d29b2d4851f,3308239329824973302397835",
		"XinShengZ_3,vite_aa8c1fa66636479ab725e2f32f9b0e17df76ae0b6adfc7675f,4467069881460490510939389",
		"HashQuark,vite_7adabd80763b42543078bcfd24888489fb790f159dc1129aef,1000396362576293440912179",
	}

	var votes []*Vote

	for _, v := range dd {
		split := strings.Split(v, ",")
		vote := &Vote{}
		vote.Name = split[0]
		vote.Addr = types.HexToAddressPanic(split[1])
		vote.Balance, _ = big.NewInt(0).SetString(split[2], 10)
		votes = append(votes, vote)
	}

	sort.Sort(ByBalance(votes))

	for k, v := range votes {
		fmt.Println(k, v.Name, v.Balance)
	}
}

func TestAlgo_FilterVotes3(t *testing.T) {

	now := time.Unix(1541640427, 0)
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		StakeAmount:            nil,
		ExpirationHeight:       0,
	})
	ag := NewAlgo(info)
	var votes []*Vote
	for i := 0; i < 100; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(100 - i))})
	}
	//result := make(map[string]uint64)
	hashH := &ledger.HashHeight{Height: 1}
	actual := ag.FilterVotes(NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(0)))
	sort.Sort(ByBalance(actual))
	for _, v := range actual {
		println("\""+v.Name+"\"", v.Balance.String(), ",")
	}

}

func TestAlgo_FilterBySuccessRate(t *testing.T) {
	a := &algo{}

	var groupA []*Vote
	for i := 10; i <= 35; i++ {
		vote := &Vote{
			Name:    fmt.Sprintf("s%d", i),
			Addr:    common.MockAddress(i),
			Balance: big.NewInt(10),
			Type:    nil,
		}
		groupA = append(groupA, vote)
	}

	var groupB []*Vote
	for i := 40; i < 50; i++ {
		vote := &Vote{
			Name:    fmt.Sprintf("s%d", i),
			Addr:    common.MockAddress(i),
			Balance: big.NewInt(10),
			Type:    nil,
		}
		groupB = append(groupB, vote)
	}

	sort.Sort(ByBalance(groupA))
	sort.Sort(ByBalance(groupB))

	successRate := make(map[types.Address]int32)

	for _, v := range groupA {
		successRate[v.Addr] = 1000000
	}
	for _, v := range groupB {
		successRate[v.Addr] = 800001
	}

	successRate[common.MockAddress(24)] = 0
	resultA, resultB := a.filterBySuccessRate(groupA, groupB, nil, successRate)

	nameA := []string{"s10", "s11", "s12", "s13", "s14", "s15", "s16", "s17", "s18", "s19", "s20", "s21", "s22", "s23", "s25", "s26", "s27", "s28", "s29", "s30", "s31", "s32", "s33", "s34", "s35", "s40"}
	for k, v := range resultA {
		assert.Equal(t, nameA[k], v.Name)
	}

	nameB := []string{"s42", "s43", "s44", "s45", "s46", "s47", "s48", "s49", "s24", "s41"}

	for k, v := range resultB {
		assert.Equal(t, nameB[k], v.Name)
	}

	successRate[common.MockAddress(23)] = 0
	resultA, resultB = a.filterBySuccessRate(groupA, groupB, nil, successRate)

	nameA = []string{"s10", "s11", "s12", "s13", "s14", "s15", "s16", "s17", "s18", "s19", "s20", "s21", "s22", "s25", "s26", "s27", "s28", "s29", "s30", "s31", "s32", "s33", "s34", "s35", "s41", "s40"}
	nameB = []string{"s42", "s43", "s44", "s45", "s46", "s47", "s48", "s49", "s24", "s23"}
	for k, v := range resultA {
		assert.Equal(t, nameA[k], v.Name)
	}

	for k, v := range resultB {
		assert.Equal(t, nameB[k], v.Name)
	}

	successRate[common.MockAddress(22)] = 0
	successRate[common.MockAddress(40)] = 0
	resultA, resultB = a.filterBySuccessRate(groupA, groupB, nil, successRate)

	nameA = []string{"s10", "s11", "s12", "s13", "s14", "s15", "s16", "s17", "s18", "s19", "s20", "s21", "s25", "s26", "s27", "s28", "s29", "s30", "s31", "s32", "s33", "s34", "s35", "s22", "s42", "s41"}
	nameB = []string{"s43", "s44", "s45", "s46", "s47", "s48", "s49", "s40", "s24", "s23"}
	for k, v := range resultA {
		assert.Equal(t, nameA[k], v.Name)
	}

	for k, v := range resultB {
		assert.Equal(t, nameB[k], v.Name)
	}
	for _, v := range resultA {
		fmt.Printf("%s \t", v.Name)
	}
	fmt.Println()

	for _, v := range resultB {
		fmt.Printf("%s \t", v.Name)
	}
	fmt.Println()
}
