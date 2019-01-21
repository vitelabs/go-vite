package api

import (
	"fmt"
	"github.com/vitelabs/go-vite/vm/abi"
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	params:=[]string{
		"true",
		"-1",
		"-01",
		"-0x1",
		"1",
		"01",
		"0x1",
		"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a",
		"tti_5649544520544f4b454e6e40",
		"00000000000000000001",
		"test",
		"89520241000000000000000000000000000000000000000000000000000000000000007b",
		"000000000000000000000000000000000000000000000000000000000000007b",
		"[true,false]",
		"[true,false]",
		"[-1,-2]",
		"[-1,-2]",
		"[-1,-2]",
		"[-1,-2]",
		"[-1,-2]",
		"[-1,-2]",
		"[1,2]",
		"[1,2]",
		"[1,2]",
		"[1,2]",
		"[1,2]",
		"[1,2]",
		"[\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23\"]",
		"[\"vite_ab24ef68b84e642c0ddca06beec81c9acb1977bbd7da27a87a\",\"vite_56fd05b23ff26cd7b0a40957fb77bde60c9fd6ebc35f809c23\"]",
		"[\"tti_5649544520544f4b454e6e40\",\"tti_2d95b4ae402bbcf1429aa1e5\"]",
		"[\"tti_5649544520544f4b454e6e40\",\"tti_2d95b4ae402bbcf1429aa1e5\"]",
		"[\"00000000000000000001\",\"00000000000000000002\"]",
		"[\"00000000000000000001\",\"00000000000000000002\"]",
		"[\"test1\",\"test2\"]",
		"[\"test1\",\"test2\"]",
	}
	abiStr := "[{\"inputs\":[" +
		"{\"type\":\"bool\"}," +
		"{\"type\":\"int8\"}," +
		"{\"type\":\"int256\"}," +
		"{\"type\":\"int56\"}," +
		"{\"type\":\"uint8\"}," +
		"{\"type\":\"uint256\"}," +
		"{\"type\":\"uint56\"}," +
		"{\"type\":\"address\"}," +
		"{\"type\":\"tokenId\"}," +
		"{\"type\":\"gid\"}," +
		"{\"type\":\"string\"}," +
		"{\"type\":\"bytes\"}," +
		"{\"type\":\"bytes32\"}," +
		"{\"type\":\"bool[]\"}," +
		"{\"type\":\"bool[2]\"}," +
		"{\"type\":\"int8[]\"}," +
		"{\"type\":\"int256[]\"}," +
		"{\"type\":\"int56[]\"}," +
		"{\"type\":\"int8[2]\"}," +
		"{\"type\":\"int256[2]\"}," +
		"{\"type\":\"int56[2]\"}," +
		"{\"type\":\"uint8[]\"}," +
		"{\"type\":\"uint256[]\"}," +
		"{\"type\":\"uint56[]\"}," +
		"{\"type\":\"uint8[2]\"}," +
		"{\"type\":\"uint256[2]\"}," +
		"{\"type\":\"uint56[2]\"}," +
		"{\"type\":\"address[]\"}," +
		"{\"type\":\"address[2]\"}," +
		"{\"type\":\"tokenId[]\"}," +
		"{\"type\":\"tokenId[2]\"}," +
		"{\"type\":\"gid[]\"}," +
		"{\"type\":\"gid[2]\"}," +
		"{\"type\":\"string[]\"}," +
		"{\"type\":\"string[2]\"}" +
	"],\"name\":\"testFunction\",\"type\":\"function\"}]"
	abiContract, err := abi.JSONToABIContract(strings.NewReader(abiStr))
	if err != nil {
		t.Fatalf("convert abi failed, %v", err)
	}
	arguments, err := convert(params, abiContract.Methods["testFunction"].Inputs)
	if err != nil {
		t.Fatalf("convert arguments failed, %v", err)
	}
	fmt.Println(arguments)
}
