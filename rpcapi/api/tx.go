package api

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
	"go.uber.org/atomic"
)

type Tx struct {
	vite *vite.Vite
	N    int
	M    int
}

var InitContractAddr = []string{
	"vite_def8c040e89f6039391fc43186c17f4ebb3c9aabf29041591a",
	"vite_156f12ac5a82fd3c4d0296f0c0ad4d6a67201f567eb03dde27",
	"vite_4344ab62d690465dcc93c874cbb82b1a2a3d5f212cac3c5d6a",
	"vite_fd88114af47a8472066714902a77ae7258196e4746ceb43cee",
	"vite_a6b16c36dab69eebe7c51cd18d4b3fdd9420bd849ce50f4858",
	"vite_2bd8bf9a82f1f13fb82c67e48c8999f8137c275df4e9871246",
	"vite_62fdcd39af744eafaa347573d4dffef0f8ecc69e6778d2765b",
	"vite_6170bed1ae72bb03e0e143021fb94226319d22ce0d0a87644c",
	"vite_ab6c0a72ee3b0d0c27a4dc8abaf40b352577100b10e971f1c3",
	"vite_43d9621b116f757c47dac06e2afa18eca5d7d3ae766ef09015",
	"vite_dbefec7ec8073910281a887377ce384aaadb3fd69ec19928be",
	"vite_b5a31ca076eb40987aea864c088aa545c312799a3035c177c5",
	"vite_c828227bc479ff81542b118e20a103ddc7c9dcb7886d6900c3",
	"vite_9d4243aa885991352ddf300ba059fee4eb4f1506bc0b0c7185",
	"vite_8274113499ad56e7e76d7a98b5d0b6ed3f074792ed650f661f",
	"vite_39e414329ba310e8f73d967ef396b8ed0bced14a4a18278205",
	"vite_f80f66f3aae027704d27c4935c941d7729210813053d3e0992",
	"vite_70db980fd8d339191a90cbde2c9dd238da93b1196d197e965b",
	"vite_4c6f327f2d743a244d21fad68cf654e76daffb34a04379284d",
	"vite_009d03a59f174e6d66bc3d282a923172a01de49c6ea8adc0e0",
	"vite_ddf2e87c66a1ced6c6536786f70f7d9c5e5af9e5e2f872067f",
	"vite_83715dc00c95a80de381f90d622a4ad44c486192a978049969",
	"vite_f3a0045ed3d849ea784af1d675d8128618674a591cb525a5f4",
	"vite_6a6189f14888792f5ef6eb7a8f884456bc693f4093fe4275c7",
	"vite_e06da81b4ee318ddafb87251f9f0ff7b1070f597c9e4c81775",
	"vite_9a3a2b8cfc12404b5bd05cb91b9c69c67d7f42d11dd8ff1c46",
	"vite_08ac3f733ffd8a414773bd06b0e0912b90d6411a3f70e6a6b1",
	"vite_5a9b248b0c854dc3ce4236e74102f3d2e2785f561c5140af5a",
	"vite_7ed6e706b7952fda2120d735ad2725ee0fc5dc41acd8e3e6e3",
	"vite_8733f708d428429e999d7ee25208d92102abe60b9ff02bfbaa",
	"vite_63ff2ef50dde59eb750446281f455471425cb0a9eaddca46a7",
	"vite_6e7eb342ed8d991ee623d1c7341adc78900214e9c63fc4aaaf",
	"vite_06c64d43e6e52e113eeff3273a24097d816ffc2abf23686218",
	"vite_979aaa04ba50e47e00d7bd7d4dbe09541792f1bb3f33b5a9d0",
	"vite_c0651bf6d49296d573af28c3f2aedfe30097db6b7ed92c1681",
	"vite_11c223edb933fd5c03955d53b5d3a84af64fba2c7a71674a19",
	"vite_fcc16dd08fa8df076c968df7cfd2adbe8206c10133f365ac42",
	"vite_a8ecac899469eed596bcdb32c7f7de0f81208f2537c1a7e125",
	"vite_e84b1d27a1d8e823b9184471804d44e4b7fc553f80c98918e6",
	"vite_d3e62898d5301f1410fa503fdec17e8ab02c4e7545177e9373",
	"vite_13b5e5ef0096947c0e7c770114a52ae9fc2c13867875d919ea",
	"vite_f805ec6425f0e88fc739a2cce8b36b0eaa1aaf8d247ad2bd60",
	"vite_313e4e9d8f1b825c5b16bdc2bd774f4831e92a3ea720a95503",
	"vite_e277a6c17a51b297800681216fdb78aa658d1cc271556246c5",
	"vite_3d68690d46f282ac7d6056257394fe3f9580f2dfece8a8ceb8",
	"vite_0623bf1f556e9516b9cc687eda29b4c5647b1c3f9a86f13c83",
	"vite_b75ffd7c7676abf3aa0b77af552134e149bb639abf44f1e345",
	"vite_93764c516ea055d5f5c75bc2e45c955f0420bdda60809841e0",
	"vite_1dd604fc3a0c24cf042ff31ce3b1a1d469719fab1c8ea85f27",
	"vite_4f0334ed66bb16e58f14cec3d56b3fd2b18a75594ac5ee51fb",
	"vite_2161aaf7c00a7393222e50a51bd3805b3ec1b935a3bcd7a075",
	"vite_08db30b61b00ebc9ee136967bf8cfc90fda4be8e78c16cdf5f",
	"vite_bcae9d01f5e96204f7f023ce92cae99b2317fe03690d703e30",
	"vite_f99061eda91d9d6faa5c6992f6ab99b31bdc44772028c8092c",
	"vite_814469298eda9857d11181fd84d773f4d3faab48e386cc9433",
	"vite_1943fc17b6a13208912ce3d37c7cc3a549f476ed4d07363c4c",
	"vite_6cef941d141a4c3d6dca84ddf2f3bbc9bb525fef96c84fc687",
	"vite_3bfb9884ff2051b78e219ccadbd3d4be0291461e3b40ceac02",
	"vite_17ae97b13d47d658dec2d6ed4ef9fc2b0ea59ae43f388f638f",
	"vite_f850aa81f2cc8ca561bb24de0a01f92aadc9c97ed46aec4729",
	"vite_6aaa32fccd9e52f9515431fd6f9e1e32d1dbb592c5c2e1213a",
	"vite_36ab4797d72d181bd9ca5bc70bfe73f2fd8dce8c342516736e",
	"vite_4b4ffefea8281d1816340a8e652786140efcbb136ecc1aedc1",
	"vite_83d0acf77da011a8e490cbbb0de7f42b2ba40f2da611fc7ab5",
	"vite_6d47ae037e7ea28b00ea39682de4d11dff95cbb5d15b7b9206",
	"vite_077a6e2a47a124d46dccc445c063e988361ef301ee05b6532f",
	"vite_dd6f7327485d08c2e94a7980a0d7dcb0ea6fadc36c41b9e0dd",
	"vite_6a02ce6060052eb3bf461bb1181acdaf2cdf815d69eb018eda",
	"vite_d84f286877ad18f01342081b40d1fb67fa8d1135217a896df1",
	"vite_fe4db81af9f75178adc606fcabc04b9261f2ce2482d07a3a97",
	"vite_43ab250ace3b8d51ec44204124fe77e342b279b5af2675f427",
	"vite_f7689b1753309191ee8010e3ab73d172e9414b52b961c5f808",
	"vite_8f9823436f1768527e310eb437b0b0663b5b14bf1dde213d67",
	"vite_7860810f4f01e591e96b977b526ea38c05ca9ed1458c5aa7c9",
	"vite_73a788abd33a9b88d28feb398164067d260058b15e1cb1cd48",
	"vite_aeb8a64183cd2cfb22174487d50b4b2d450382608a0a969d23",
	"vite_3794a275a039dca0e9c23295157c10479ece24e8ffcea52107",
	"vite_b519618aad01af6b932666d04d392382dbe76b3b9b8ccb2beb",
	"vite_51dc7d994b0815e2a234ae97a455589d1d9addfdd74b1c5a5f",
	"vite_8d3b868ae9cae1c1c757f2e05e2e66684b907e3e85ed909997",
	"vite_83a9bd967aac6a922d865d6c6851fe2182d1117e9719b265fa",
	"vite_3a2be49a956932ccf961d2f1ebd49503f829292cf063a6be37",
	"vite_aeac4c06dc91251fe5539a20f990c582820706b3ca1c78ac6d",
	"vite_9d1cc38e50a8c8548c1d437ef2436a268556161dfd690f890c",
	"vite_3fd92d0f943179e16c5c6262c49f92ee0c5c0b424cad86d65f",
	"vite_8a04a78e5beae8601d1c7dd0021fa08c7d110471ffc6816181",
	"vite_308bbe7023599d578185318e38caccd9bbbf5cc4ed014ba31b",
	"vite_b2fe7aabcfe443ce00a5bfcc0e8a0a0f6d43899627493ba177",
	"vite_1ac42299338661ab8ff6d446954ef00ff8e45929aa842ff9ab",
	"vite_ba531176a1c3aa866f2d35b25a1e16d8d434377ea65637a5b2",
	"vite_44a27a6048ba8ae0ba0000dd5a4b5de37f427940f53a1a501a",
	"vite_f3c95e89321932ebd0f68daec632da721e7a899171f95cb81f",
	"vite_dd87ef206aa2e8988d2595b79fb41543019f7e5826a90d755e",
	"vite_7a71a5acd055b1f82d705ccffd112e05b3afdf4d2f95c9e1ea",
	"vite_88f5e223a569afc3795e438d4811bc1bbc7cafcb538d7d3b3b",
	"vite_721dc08100b11853a449d03d1fde683c508ef20ca665bd0f04",
	"vite_556813733e3a097104ae007f30df6ec3a620e5ec46dce4a86a",
	"vite_cf00e3d419c0c8bd38c5d389d98174b1e93db75109dd1aa409",
	"vite_75eb7e77c1ad8fac45a0f122ccca436473dc2bdfcd27650917",
	"vite_39f6e2e65101ec67f83eabcf8614b06ce50dc900715228570a",
}

func NewTxApi(vite *vite.Vite) *Tx {
	tx := &Tx{
		vite: vite,
		N:    3,
		M:    0,
	}
	if vite.Producer() == nil {
		return tx
	}
	coinbase := vite.Producer().GetCoinBase()

	manager, err := vite.WalletManager().GetEntropyStoreManager(coinbase.String())
	if err != nil {
		panic(err)
	}

	var fromAddrs []types.Address
	var fromHexPrivKeys []string

	{
		for i := uint32(0); i < uint32(10); i++ {
			_, key, err := manager.DeriveForIndexPath(i)
			if err != nil {
				panic(err)
			}
			binKey, err := key.PrivateKey()
			if err != nil {
				panic(err)
			}

			pubKey, err := key.PublicKey()
			if err != nil {
				panic(err)
			}

			address, err := key.Address()
			if err != nil {
				panic(err)
			}

			fmt.Printf("%s hex public key:%s\n", address, hex.EncodeToString(pubKey))

			hexKey := hex.EncodeToString(binKey)
			fromHexPrivKeys = append(fromHexPrivKeys, hexKey)
			fromAddrs = append(fromAddrs, *address)

		}
	}

	toAddr := types.AddressConsensusGroup
	amount := string("0")

	ss := []string{
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnMxAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnMyAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnMzAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnM0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"/cF/JQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAnM1AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	var datas [][]byte
	for _, v := range ss {
		byts, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			panic(err)
		}
		datas = append(datas, byts)
	}

	num := atomic.NewUint32(0)

	vite.Consensus().Subscribe(types.SNAPSHOT_GID, "api-auto-send", nil, func(e consensus.Event) {

		if num.Load() > 0 {
			fmt.Printf("something is loading[return].%s\n", time.Now())
			return
		}
		num.Add(1)
		defer num.Sub(1)
		snapshotBlock := vite.Chain().GetLatestSnapshotBlock()
		if snapshotBlock.Height < 10 {
			fmt.Println("latest height must >= 10.")
			return
		}

		state := vite.Net().Status().State
		if state != net.SyncDone {
			fmt.Printf("sync state: %s \n", state)
			return
		}

		for i := 0; i < tx.N; i++ {
			for k, v := range fromAddrs {
				addr := v
				key := fromHexPrivKeys[k]

				block, err := tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
					SelfAddr:     &addr,
					ToAddr:       &toAddr,
					TokenTypeId:  ledger.ViteTokenId,
					PrivateKey:   &key,
					Amount:       &amount,
					Data:         datas[rand.Intn(len(datas))],
					Difficulty:   nil,
					PreBlockHash: nil,
					BlockType:    2,
				})
				if err != nil {
					log.Error(fmt.Sprintf("send block err:%v\n", err))
					return
				} else {
					log.Info(fmt.Sprintf("send block:%s,%s,%s\n", block.AccountAddress, block.Height, block.Hash))
				}
			}

		}

		for i := 0; i < tx.M; i++ {
			for k, v := range fromAddrs {
				addr := v
				key := fromHexPrivKeys[k]

				mToAddr := types.HexToAddressPanic(InitContractAddr[rand.Intn(len(InitContractAddr))])
				block, err := tx.SendTxWithPrivateKey(SendTxWithPrivateKeyParam{
					SelfAddr:     &addr,
					ToAddr:       &mToAddr,
					TokenTypeId:  ledger.ViteTokenId,
					PrivateKey:   &key,
					Amount:       &amount,
					Difficulty:   nil,
					PreBlockHash: nil,
					BlockType:    2,
				})
				if err != nil {
					log.Error(fmt.Sprintf("send block err:%v\n", err))
					return
				} else {
					log.Info(fmt.Sprintf("send block:%s,%s,%s\n", block.AccountAddress, block.Height, block.Hash))
				}
			}
		}
	})

	return tx
}

func (t *Tx) UpdateBenchMark(cnt int) (int, error) {
	t.N = cnt
	return t.N, nil
}

func (t Tx) SendRawTx(block *AccountBlock) error {
	log.Info("SendRawTx")
	if block == nil {
		return errors.New("empty block")
	}

	lb, err := block.LedgerAccountBlock()
	if err != nil {
		return err
	}
	// need to remove Later
	//if len(lb.Data) != 0 && !isBuiltinContracts(lb.ToAddress) {
	//	return ErrorNotSupportAddNot
	//}
	//
	//if len(lb.Data) != 0 && block.BlockType == ledger.BlockTypeReceive {
	//	return ErrorNotSupportRecvAddNote
	//}

	v := verifier.NewVerifier(nil, verifier.NewAccountVerifier(t.vite.Chain(), t.vite.Consensus()))
	latestSb := t.vite.Chain().GetLatestSnapshotBlock()
	if latestSb == nil {
		return errors.New("failed to get latest snapshotBlock")
	}
	result, err := v.VerifyRPCAccBlock(lb, &latestSb.Hash)
	if err != nil {
		return err
	}

	if result != nil {
		return t.vite.Pool().AddDirectAccountBlock(result.AccountBlock.AccountAddress, result)
	} else {
		return errors.New("generator gen an empty block")
	}
	return nil
}

func (t Tx) SendTxWithPrivateKey(param SendTxWithPrivateKeyParam) (*AccountBlock, error) {

	if param.Amount == nil {
		return nil, errors.New("amount is nil")
	}

	if param.SelfAddr == nil {
		return nil, errors.New("selfAddr is nil")
	}

	if param.ToAddr == nil && param.BlockType != ledger.BlockTypeSendCreate {
		return nil, errors.New("toAddr is nil")
	}

	if param.PrivateKey == nil {
		return nil, errors.New("privateKey is nil")
	}

	var d *big.Int = nil
	if param.Difficulty != nil {
		t, ok := new(big.Int).SetString(*param.Difficulty, 10)
		if !ok {
			return nil, ErrStrToBigInt
		}
		d = t
	}

	amount, ok := new(big.Int).SetString(*param.Amount, 10)
	if !ok {
		return nil, ErrStrToBigInt
	}

	var blockType byte
	if param.BlockType > 0 {
		blockType = param.BlockType
	} else {
		blockType = ledger.BlockTypeSendCall
	}

	msg := &generator.IncomingMessage{
		BlockType:      blockType,
		AccountAddress: *param.SelfAddr,
		ToAddress:      param.ToAddr,
		TokenId:        &param.TokenTypeId,
		Amount:         amount,
		Fee:            nil,
		Data:           param.Data,
		Difficulty:     d,
	}

	addrState, err := generator.GetAddressStateForGenerator(t.vite.Chain(), &msg.AccountAddress)
	if err != nil || addrState == nil {
		return nil, errors.New(fmt.Sprintf("failed to get addr state for generator, err:%v", err))
	}
	g, e := generator.NewGenerator(t.vite.Chain(), t.vite.Consensus(), msg.AccountAddress, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if e != nil {
		return nil, e
	}
	result, e := g.GenerateWithMessage(msg, &msg.AccountAddress, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
		var privkey ed25519.PrivateKey
		privkey, e := ed25519.HexToPrivateKey(*param.PrivateKey)
		if e != nil {
			return nil, nil, e
		}
		signData := ed25519.Sign(privkey, data)
		pubkey = privkey.PubByte()
		return signData, pubkey, nil
	})
	if e != nil {
		return nil, e
	}
	if result.Err != nil {
		return nil, result.Err
	}
	if result.VmBlock != nil {
		if err := t.vite.Pool().AddDirectAccountBlock(msg.AccountAddress, result.VmBlock); err != nil {
			return nil, err
		}
		return ledgerToRpcBlock(result.VmBlock.AccountBlock, t.vite.Chain())
	} else {
		return nil, errors.New("generator gen an empty block")
	}
}

type SendTxWithPrivateKeyParam struct {
	SelfAddr     *types.Address    `json:"selfAddr"`
	ToAddr       *types.Address    `json:"toAddr"`
	TokenTypeId  types.TokenTypeId `json:"tokenTypeId"`
	PrivateKey   *string           `json:"privateKey"` //hex16
	Amount       *string           `json:"amount"`
	Data         []byte            `json:"data"` //base64
	Difficulty   *string           `json:"difficulty,omitempty"`
	PreBlockHash *types.Hash       `json:"preBlockHash,omitempty"`
	BlockType    byte              `json:"blockType"`
}

type CalcPoWDifficultyParam struct {
	SelfAddr types.Address `json:"selfAddr"`
	PrevHash types.Hash    `json:"prevHash"`

	BlockType byte           `json:"blockType"`
	ToAddr    *types.Address `json:"toAddr"`
	Data      []byte         `json:"data"`

	UsePledgeQuota bool `json:"usePledgeQuota"`
}

type CalcPoWDifficultyResult struct {
	QuotaRequired uint64 `json:"quota"`
	Difficulty    string `json:"difficulty"`
}

func (t Tx) CalcPoWDifficulty(param CalcPoWDifficultyParam) (result *CalcPoWDifficultyResult, err error) {
	latestBlock, err := t.vite.Chain().GetLatestAccountBlock(param.SelfAddr)
	if err != nil {
		return nil, err
	}
	if (latestBlock == nil && !param.PrevHash.IsZero()) ||
		(latestBlock != nil && latestBlock.Hash != param.PrevHash) {
		return nil, util.ErrChainForked
	}
	// get quota required
	block := &ledger.AccountBlock{
		BlockType:      param.BlockType,
		AccountAddress: param.SelfAddr,
		PrevHash:       param.PrevHash,
		Data:           param.Data,
	}
	if param.ToAddr != nil {
		block.ToAddress = *param.ToAddr
	} else if param.BlockType == ledger.BlockTypeSendCall {
		return nil, errors.New("toAddr is nil")
	}
	sb := t.vite.Chain().GetLatestSnapshotBlock()
	db, err := vm_db.NewVmDb(t.vite.Chain(), &param.SelfAddr, &sb.Hash, &param.PrevHash)
	if err != nil {
		return nil, err
	}
	quotaRequired, err := vm.GasRequiredForBlock(db, block)
	if err != nil {
		return nil, err
	}

	// get current quota
	var pledgeAmount *big.Int
	var q types.Quota
	if param.UsePledgeQuota {
		pledgeAmount, err = t.vite.Chain().GetPledgeBeneficialAmount(param.SelfAddr)
		if err != nil {
			return nil, err
		}
		q, err := quota.GetPledgeQuota(db, param.SelfAddr, pledgeAmount)
		if err != nil {
			return nil, err
		}
		if q.Current() >= quotaRequired {
			return &CalcPoWDifficultyResult{quotaRequired, ""}, nil
		}
	} else {
		pledgeAmount = big.NewInt(0)
		q = types.NewQuota(0, 0, 0, 0)
	}
	// calc difficulty if current quota is not enough
	canPoW, err := quota.CanPoW(db, block.AccountAddress)
	if err != nil {
		return nil, err
	}
	if !canPoW {
		return nil, util.ErrCalcPoWTwice
	}
	d, err := quota.CalcPoWDifficulty(quotaRequired, q)
	if err != nil {
		return nil, err
	}
	return &CalcPoWDifficultyResult{quotaRequired, d.String()}, nil
}
