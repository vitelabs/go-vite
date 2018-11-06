package abi

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"strings"
	"time"
)

const (
	jsonRegister = `
	[
		{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"Name":"name","type":"string"},{"name":"nodeAddr","type":"address"}]},
		{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
		{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"beneficialAddr","type":"address"}]},
		{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"nodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"rewardIndex","type":"uint64"},{"name":"cancelHeight","type":"uint64"},{"name":"hisAddrList","type":"address[]"}]},
		{"type":"variable","name":"hisName","inputs":[{"name":"name","type":"string"}]}
	]`

	MethodNameRegister           = "Register"
	MethodNameCancelRegister     = "CancelRegister"
	MethodNameReward             = "Reward"
	MethodNameUpdateRegistration = "UpdateRegistration"
	VariableNameRegistration     = "registration"
	VariableNameHisName          = "hisName"
)

var (
	ABIRegister, _ = abi.JSONToABIContract(strings.NewReader(jsonRegister))
)

type ParamRegister struct {
	Gid      types.Gid
	Name     string
	NodeAddr types.Address
}
type ParamCancelRegister struct {
	Gid  types.Gid
	Name string
}
type ParamReward struct {
	Gid            types.Gid
	Name           string
	BeneficialAddr types.Address
}

func GetRegisterKey(name string, gid types.Gid) []byte {
	return append(gid.Bytes(), types.DataHash([]byte(name)).Bytes()[types.GidSize:]...)
}

func GetHisNameKey(addr types.Address, gid types.Gid) []byte {
	return append(addr.Bytes(), gid.Bytes()...)
}

func IsRegisterKey(key []byte) bool {
	return len(key) == types.HashSize
}

func IsActiveRegistration(db StorageDatabase, name string, gid types.Gid) bool {
	if value := db.GetStorageBySnapshotHash(&AddressRegister, GetRegisterKey(name, gid), nil); len(value) > 0 {
		registration := new(types.Registration)
		if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil {
			return registration.IsActive()
		}
	}
	return false
}

func GetCandidateList(db StorageDatabase, gid types.Gid, snapshotHash *types.Hash) []*types.Registration {
	defer monitor.LogTime("vm", "GetCandidateList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIteratorBySnapshotHash(&AddressRegister, types.SNAPSHOT_GID.Bytes(), snapshotHash)
	} else {
		iterator = db.NewStorageIteratorBySnapshotHash(&AddressRegister, gid.Bytes(), snapshotHash)
	}
	registerList := make([]*types.Registration, 0)
	if iterator == nil {
		return registerList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if IsRegisterKey(key) {
			registration := new(types.Registration)
			if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil && registration.IsActive() {
				registerList = append(registerList, registration)
			}
		}
	}
	return registerList
}

func GetRegistrationList(db StorageDatabase, gid types.Gid, pledgeAddr types.Address) []*types.Registration {
	defer monitor.LogTime("vm", "GetRegistrationList", time.Now())
	var iterator vmctxt_interface.StorageIterator
	if gid == types.DELEGATE_GID {
		iterator = db.NewStorageIteratorBySnapshotHash(&AddressRegister, types.SNAPSHOT_GID.Bytes(), nil)
	} else {
		iterator = db.NewStorageIteratorBySnapshotHash(&AddressRegister, gid.Bytes(), nil)
	}
	registerList := make([]*types.Registration, 0)
	if iterator == nil {
		return registerList
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if IsRegisterKey(key) {
			registration := new(types.Registration)
			if err := ABIRegister.UnpackVariable(registration, VariableNameRegistration, value); err == nil && registration.PledgeAddr == pledgeAddr {
				registerList = append(registerList, registration)
			}
		}
	}
	return registerList
}
