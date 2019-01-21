package dex

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	cabi "github.com/vitelabs/go-vite/vm/contracts/abi"
	dexproto "github.com/vitelabs/go-vite/vm/contracts/dex/proto"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"math/big"
	"strings"
)

var (
	fundKeyPrefix = []byte("fund:") // fund:types.Address

	feeAccKeyPrefix          = []byte("fee:") // fee:feeAccId feeAccId = PeriodIdByHeight
	feePeriodByHeight uint64 = 10

	vxFundKeyPrefix     = []byte("vxFund:") // vxFund:types.Address
	VxTokenBytes        = []byte{0, 0, 0, 0, 0, 1, 2, 3, 4, 5}
	VxDividendThreshold = new(big.Int).Mul(new(big.Int).Exp(helper.Big10, new(big.Int).SetUint64(uint64(18)), nil), big.NewInt(10)) // 18 : vx decimals, 10 amount
)

type UserFund struct {
	dexproto.Fund
}

func (df *UserFund) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.Fund)
}

func (df *UserFund) DeSerialize(fundData []byte) (dexFund *UserFund, err error) {
	protoFund := dexproto.Fund{}
	if err := proto.Unmarshal(fundData, &protoFund); err != nil {
		return nil, err
	} else {
		return &UserFund{protoFund}, nil
	}
}

type Fee struct {
	dexproto.FeeByPeriod
}

func (df *Fee) Serialize() (data []byte, err error) {
	return proto.Marshal(&df.FeeByPeriod)
}

func (df *Fee) DeSerialize(feeData []byte) (dexFee *Fee, err error) {
	protoFee := dexproto.FeeByPeriod{}
	if err := proto.Unmarshal(feeData, &protoFee); err != nil {
		return nil, err
	} else {
		return &Fee{protoFee}, nil
	}
}

type VxFunds struct {
	dexproto.VxFunds
}

func (dvf *VxFunds) Serialize() (data []byte, err error) {
	return proto.Marshal(&dvf.VxFunds)
}

func (dvf *VxFunds) DeSerialize(vxFundsData []byte) (*VxFunds, error) {
	protoVxFunds := dexproto.VxFunds{}
	if err := proto.Unmarshal(vxFundsData, &protoVxFunds); err != nil {
		return nil, err
	} else {
		return &VxFunds{protoVxFunds}, nil
	}
}

func CheckSettleActions(actions *dexproto.SettleActions) error {
	if actions == nil || len(actions.FundActions) == 0 && len(actions.FeeActions) == 0 {
		return fmt.Errorf("settle actions is emtpy")
	}
	for _, v := range actions.FundActions {
		if len(v.Address) != 20 {
			return fmt.Errorf("invalid address format for settle")
		}
		if len(v.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for settle")
		}
		if CmpToBigZero(v.IncAvailable) < 0 {
			return fmt.Errorf("negative incrAvailable for settle")
		}
		if CmpToBigZero(v.ReduceLocked) < 0 {
			return fmt.Errorf("negative reduceLocked for settle")
		}
		if CmpToBigZero(v.ReleaseLocked) < 0 {
			return fmt.Errorf("negative releaseLocked for settle")
		}
	}

	for _, fee := range actions.FeeActions {
		if len(fee.Token) != 10 {
			return fmt.Errorf("invalid tokenId format for fee settle")
		}
		if CmpToBigZero(fee.Amount) <= 0 {
			return fmt.Errorf("negative feeAmount for settle")
		}
	}
	return nil
}

func GetAccountByTokeIdFromFund(dexFund *UserFund, token types.TokenTypeId) (account *dexproto.Account, exists bool) {
	for _, a := range dexFund.Accounts {
		if bytes.Equal(token.Bytes(), a.Token) {
			return a, true
		}
	}
	account = &dexproto.Account{}
	account.Token = token.Bytes()
	account.Available = big.NewInt(0).Bytes()
	account.Locked = big.NewInt(0).Bytes()
	return account, false
}

func GetAccountFundInfo(dexFund *UserFund, tokenId *types.TokenTypeId) ([]*Account, error) {
	var dexAccount = make([]*Account, 0)
	if tokenId != nil {
		for _, v := range dexFund.Accounts {
			if bytes.Equal(tokenId.Bytes(), v.Token) {
				var acc = &Account{}
				acc.Deserialize(v)
				dexAccount = append(dexAccount, acc)
				break
			}
		}
	} else {
		for _, v := range dexFund.Accounts {
			var acc = &Account{}
			acc.Deserialize(v)
			dexAccount = append(dexAccount, acc)
		}
	}
	return dexAccount, nil
}

func GetUserFundFromStorage(storage vmctxt_interface.VmDatabase, address types.Address) (dexFund *UserFund, err error) {
	fundKey := GetUserFundKey(address)
	dexFund = &UserFund{}
	if fundBytes := storage.GetStorage(&types.AddressDexFund, fundKey); len(fundBytes) > 0 {
		if dexFund, err = dexFund.DeSerialize(fundBytes); err != nil {
			return nil, err
		}
	}
	return dexFund, nil
}

func SaveUserFundToStorage(storage vmctxt_interface.VmDatabase, address types.Address, dexFund *UserFund) error {
	if fundRes, err := dexFund.Serialize(); err == nil {
		storage.SetStorage(GetUserFundKey(address), fundRes)
		return nil
	} else {
		return err
	}
}

func GetAccountByAddressAndTokenId(storage vmctxt_interface.VmDatabase, address types.Address, token types.TokenTypeId) (account *dexproto.Account, exists bool, err error) {
	if dexFund, err := GetUserFundFromStorage(storage, address); err != nil {
		return nil, false, err
	} else {
		account, exists = GetAccountByTokeIdFromFund(dexFund, token)
		return account, exists, nil
	}
}

func GetUserFundKey(address types.Address) []byte {
	return append(fundKeyPrefix, address.Bytes()...)
}

func GetFeeFromStorage(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64) (dexFee *Fee, err error) {
	feeKey := GetFeeKeyForHeight(snapshotBlockHeight)
	dexFee = &Fee{}
	if feeBytes := storage.GetStorage(&types.AddressDexFund, feeKey); len(feeBytes) > 0 {
		if dexFee, err = dexFee.DeSerialize(feeBytes); err != nil {
			return nil, err
		} else {
			return dexFee, nil
		}
	} else {
		dexFee.Divided = false
		return dexFee, nil
	}
}

func SaveFeeToStorage(storage vmctxt_interface.VmDatabase, snapshotBlockHeight uint64, fee *Fee) error {
	feeKey := GetFeeKeyForHeight(snapshotBlockHeight)
	if feeBytes, err := proto.Marshal(fee); err == nil {
		storage.SetStorage(feeKey, feeBytes)
		return nil
	} else {
		return err
	}
}

func GetFeeKeyForHeight(height uint64) []byte {
	return append(feeAccKeyPrefix, GetPeriodIdBytesFromHeight(height)...)
}

func GetVxFundsFromStorage(storage vmctxt_interface.VmDatabase, address []byte) (vxFunds *VxFunds, err error) {
	vxFundsKey := GetVxFundsKey(address)
	vxFunds = &VxFunds{}
	if vxFundsBytes := storage.GetStorage(&types.AddressDexFund, vxFundsKey); len(vxFundsBytes) > 0 {
		if vxFunds, err = vxFunds.DeSerialize(vxFundsBytes); err != nil {
			return nil, err
		} else {
			return vxFunds, nil
		}
	} else {
		return vxFunds, nil
	}
}

func SaveVxFundsToStorage(storage vmctxt_interface.VmDatabase, address []byte, vxFunds *VxFunds) error {
	vxFundsKey := GetVxFundsKey(address)
	if vxFundsBytes, err := proto.Marshal(vxFunds); err == nil {
		storage.SetStorage(vxFundsKey, vxFundsBytes)
		return nil
	} else {
		return err
	}
}

func GetVxFundsKey(address []byte) []byte {
	return append(vxFundKeyPrefix, address...)
}

func IsValidVxAmountForDividend(amount *big.Int) bool {
	return amount.Cmp(VxDividendThreshold) >= 0
}

func FromBytesToTokenTypeId(bytes []byte) (tokenId *types.TokenTypeId, err error) {
	tokenId = &types.TokenTypeId{}
	if err := tokenId.SetBytes(bytes); err == nil {
		return tokenId, nil
	} else {
		return nil, err
	}
}

func GetTokenInfo(db vmctxt_interface.VmDatabase, tokenId types.TokenTypeId) (error, *types.TokenInfo) {
	if tokenInfo := cabi.GetTokenById(db, tokenId); tokenInfo == nil {
		return fmt.Errorf("token is invalid"), nil
	} else {
		return nil, tokenInfo
	}
}

func GetPeriodIdBytesFromHeight(height uint64) []byte {
	periodId := GetPeriodIdFromHeight(height)
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, periodId)
	return bs
}

func GetPeriodIdFromHeight(height uint64) uint64 {
	var periodIdByLastHeight uint64
	if int64(height)%int64(feePeriodByHeight) == 0 {
		periodIdByLastHeight = height
	} else {
		periodIdByLastHeight = ((height / feePeriodByHeight) + 1) * feePeriodByHeight
	}
	return periodIdByLastHeight
}

func ValidPrice(price string) bool {
	if len(price) == 0 {
		return false
	} else if pr, ok := new(big.Float).SetString(price); !ok || pr.Cmp(big.NewFloat(0)) <= 0 {
		return false
	} else {
		idx := strings.Index(price, ",")
		if idx > 0 && len(price)-idx >= 12 { // price max precision is 10 decimal
			return false
		}
	}
	return true
}
