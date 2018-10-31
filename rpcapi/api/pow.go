package api

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pow"
	"io/ioutil"
	"math/big"
	"net/http"
)

const requestUrl = "http://134.175.140.182:6007"

const ApiActionGenerate = "/api/generate_work"

type Pow struct {
}

type workGenerate struct {
	DataHash  string `json:"hash"`
	Threshold string `json:"threshold"`
}

func (p Pow) GetPowNonce(difficulty string, data types.Hash) ([]byte, error) {
	log.Info("GetPowNonce")
	//
	//wreq := workGenerate{
	//	DataHash:  data.Hex(),
	//	Threshold: "FFFFFFC000000000000000000000000000000000000000000000000000000000",
	//}
	//reqBytes, e := json.Marshal(wreq)
	//if e != nil {
	//	return nil, e
	//}
	//
	//resp, err := http.Post("", "application/json", bytes.NewReader(reqBytes))
	//if err != nil {
	//	return nil, e
	//}
	//if resp.StatusCode != 200 {
	//	return nil, errors.New("error pow server return ")
	//}

	work, e := GenerateWork(data.Bytes(), difficulty)
	if e != nil {
		return nil, e
	}

	nonceStr := *work
	nonceBig, ok := new(big.Int).SetString(nonceStr, 16)
	if !ok {
		return nil, errors.New("wrong nonce str")
	}
	nonceUint64 := nonceBig.Uint64()
	nn := make([]byte, 8)
	binary.LittleEndian.PutUint64(nn[:], nonceUint64)

	bd, ok := new(big.Int).SetString(difficulty, 16)
	if !ok {
		return nil, errors.New("wrong nonce difficulty")
	}

	if !pow.CheckPowNonce(bd, nn, data.Bytes()) {
		return nil, errors.New("check nonce failed")
	}

	return nn, nil
}

type ResponseJson struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
	Msg   string      `json:"msg"`
}

type workGenerateResult struct {
	Work string `json:"work"`
}

func httpRequest(requestPath string, bytesData []byte, responseInterface interface{}) error {
	req, err := http.NewRequest("POST", requestPath, bytes.NewReader(bytesData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseJson := &ResponseJson{Data: responseInterface}
	if err := json.Unmarshal(body, responseJson); err != nil {
		return err
	}
	if responseJson.Code != 0 {
		return errors.New(responseJson.Error)
	}
	responseInterface = responseJson.Data
	return nil
}

func GenerateWork(dataHash []byte, threshold string) (*string, error) {
	wg := &workGenerate{
		Threshold: threshold,
		DataHash:  hex.EncodeToString(dataHash),
	}
	bytesData, err := json.Marshal(wg)
	if err != nil {
		return nil, err
	}
	workResult := &workGenerateResult{}
	if err := httpRequest(requestUrl+ApiActionGenerate, bytesData, workResult); err != nil {
		return nil, err
	}

	return &workResult.Work, nil
}
