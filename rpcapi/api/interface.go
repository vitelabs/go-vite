package api

import (
	"encoding/hex"
	"github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
)

func convert(params []string, arguments abi.Arguments) ([]interface{}, error) {
	if len(params) != len(arguments) {
		return nil, errors.New("argument size not match")
	}
	resultList := make([]interface{}, len(params))
	for i, argument := range arguments {
		result, err := convertOne(params[i], argument.Type)
		if err != nil {
			return nil, err
		}
		resultList[i] = result
	}
	return resultList, nil
}

func convertOne(param string, t abi.Type) (interface{}, error) {
	typeString := t.String()
	if strings.Contains(typeString, "[") {
		return convertToArray(param, t)
	} else if typeString == "bool" {
		return convertToBool(param)
	} else if strings.HasPrefix(typeString, "int") {
		return convertToInt(param, t.Size)
	} else if strings.HasPrefix(typeString, "uint") {
		return convertToUint(param, t.Size)
	} else if typeString == "address" {
		return types.HexToAddress(param)
	} else if typeString == "tokenId" {
		return types.HexToTokenTypeId(param)
	} else if typeString == "gid" {
		return types.HexToGid(param)
	} else if typeString == "string" {
		return param, nil
	} else if typeString == "bytes" {
		return convertToDynamicBytes(param)
	} else if strings.HasPrefix(typeString, "bytes") {
		return convertToFixedBytes(param, t.Size)
	}
	return nil, errors.New("unknown type " + typeString)
}

func convertToArray(param string, t abi.Type) (interface{}, error) {
	if t.Elem.Elem != nil {
		return nil, errors.New(t.String() + " type not supported")
	}
	elemList := []string{}
	if err := jsoniter.UnmarshalFromString(param, &elemList); err != nil {
		return nil, err
	}
	typeString := t.Elem.String()
	if typeString == "bool" {
		return convertToBoolArray(elemList)
	} else if strings.HasPrefix(typeString, "int") {
		return convertToIntArray(elemList, *t.Elem)
	} else if strings.HasPrefix(typeString, "uint") {
		return convertToUintArray(elemList, *t.Elem)
	} else if typeString == "address" {
		return convertToAddressArray(elemList)
	} else if typeString == "tokenId" {
		return convertToTokenIdArray(elemList)
	} else if typeString == "gid" {
		return convertToGidArray(elemList)
	} else if typeString == "string" {
		return elemList, nil
	} else if strings.HasPrefix(typeString, "bytes") {
		return convertToBytesArray(elemList, *t.Elem)
	}
	return nil, errors.New(typeString + " array type not supported")
}

func convertToBoolArray(elemList []string) (interface{}, error) {
	resultList := make([]bool, 0)
	for _, elem := range elemList {
		elemInterface, err := convertToBool(elem)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, elemInterface.(bool))
	}
	return resultList, nil
}

func convertToIntArray(elemList []string, t abi.Type) (interface{}, error) {
	size := t.Size
	if size == 8 {
		resultList := make([]int8, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToInt(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(int8))
		}
		return resultList, nil
	} else if size == 16 {
		resultList := make([]int16, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToInt(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(int16))
		}
		return resultList, nil
	} else if size == 32 {
		resultList := make([]int32, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToInt(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(int32))
		}
		return resultList, nil
	} else if size == 64 {
		resultList := make([]int64, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToInt(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(int64))
		}
		return resultList, nil
	} else {
		resultList := make([]*big.Int, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToInt(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(*big.Int))
		}
		return resultList, nil
	}
}
func convertToUintArray(elemList []string, t abi.Type) (interface{}, error) {
	size := t.Size
	if size == 8 {
		resultList := make([]uint8, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToUint(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(uint8))
		}
		return resultList, nil
	} else if size == 16 {
		resultList := make([]uint16, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToUint(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(uint16))
		}
		return resultList, nil
	} else if size == 32 {
		resultList := make([]uint32, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToUint(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(uint32))
		}
		return resultList, nil
	} else if size == 64 {
		resultList := make([]uint64, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToUint(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(uint64))
		}
		return resultList, nil
	} else {
		resultList := make([]*big.Int, 0)
		for _, elem := range elemList {
			elemInterface, err := convertToUint(elem, size)
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, elemInterface.(*big.Int))
		}
		return resultList, nil
	}
}
func convertToAddressArray(elemList []string) (interface{}, error) {
	resultList := make([]types.Address, 0)
	for _, elem := range elemList {
		elemInterface, err := types.HexToAddress(elem)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, elemInterface)
	}
	return resultList, nil
}
func convertToTokenIdArray(elemList []string) (interface{}, error) {
	resultList := make([]types.TokenTypeId, 0)
	for _, elem := range elemList {
		elemInterface, err := types.HexToTokenTypeId(elem)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, elemInterface)
	}
	return resultList, nil
}
func convertToGidArray(elemList []string) (interface{}, error) {
	resultList := make([]types.Gid, 0)
	for _, elem := range elemList {
		elemInterface, err := types.HexToGid(elem)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, elemInterface)
	}
	return resultList, nil
}
func convertToBytesArray(elemList []string, t abi.Type) (interface{}, error) {
	resultList := make([][]byte, 0)
	for _, elem := range elemList {
		elemInterface, err := convertToBytes(elem, t.Size)
		if err != nil {
			return nil, err
		}
		resultList = append(resultList, elemInterface.([]byte))
	}
	return resultList, nil
}

func convertToBool(param string) (interface{}, error) {
	if param == "true" {
		return true, nil
	} else {
		return false, nil
	}
	return nil, errors.New(param + " convert to bool failed")
}

func convertToInt(param string, size int) (interface{}, error) {
	bigInt, ok := new(big.Int).SetString(param, 0)
	if !ok || bigInt.BitLen() > size-1 {
		return nil, errors.New(param + " convert to int failed")
	}
	if size == 8 {
		return int8(bigInt.Int64()), nil
	} else if size == 16 {
		return int16(bigInt.Int64()), nil
	} else if size == 32 {
		return int32(bigInt.Int64()), nil
	} else if size == 64 {
		return int64(bigInt.Int64()), nil
	} else {
		return bigInt, nil
	}
}

func convertToUint(param string, size int) (interface{}, error) {
	bigInt, ok := new(big.Int).SetString(param, 0)
	if !ok || bigInt.BitLen() > size {
		return nil, errors.New(param + " convert to uint failed")
	}
	if size == 8 {
		return uint8(bigInt.Uint64()), nil
	} else if size == 16 {
		return uint16(bigInt.Uint64()), nil
	} else if size == 32 {
		return uint32(bigInt.Uint64()), nil
	} else if size == 64 {
		return uint64(bigInt.Uint64()), nil
	} else {
		return bigInt, nil
	}
}

func convertToBytes(param string, size int) (interface{}, error) {
	if size == 0 {
		return convertToDynamicBytes(param)
	} else {
		return convertToFixedBytes(param, size)
	}
}

func convertToFixedBytes(param string, size int) (interface{}, error) {
	if !strings.HasPrefix(param, "0x") || len(param) != size*2+2 {
		return nil, errors.New(param + " is not valid bytes")
	}
	return hex.DecodeString(param[2:])
}
func convertToDynamicBytes(param string) (interface{}, error) {
	if !strings.HasPrefix(param, "0x") {
		return nil, errors.New(param + " is not valid bytes")
	}
	return hex.DecodeString(param[2:])
}
