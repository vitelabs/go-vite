package remote

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/log15"
	"io/ioutil"
	"net/http"
	"strconv"
)

var (
	powClientLog = log15.New("module", "pow_request")
	requestUrl   string
)

const (
	ApiActionGenerate = "/api/generate_work"
	ApiActionValidate = "/api/validate_work"
	ApiActionCancel   = "/api/cancel_work"
)

func InitUrl(ip string) {
	requestUrl = "http://" + ip + ":6007"
}

type workGenerate struct {
	DataHash  string `json:"hash"`
	Threshold string `json:"threshold"`
}

type workValidate struct {
	DataHash  string `json:"hash"`
	Threshold string `json:"threshold"`
	Work      string `json:"work"`
}

type workCancel struct {
	DataHash string `json:"hash"`
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
	powClientLog.Info("Response Status:", resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	//fmt.Println(helper.BytesToString(body))

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

func GenerateWork(dataHash []byte, threshold uint64) ([]byte, error) {
	wg := &workGenerate{
		Threshold: strconv.FormatUint(threshold, 16),
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
	work := []byte(workResult.Work)
	if len(work) < 8 {
		powClientLog.New("the length of work generated is less than eight")
		return helper.LeftPadBytes(work, 8), nil
	}
	return work, nil
}

func VaildateWork(dataHash []byte, threshold uint64, work []byte) (bool, error) {
	wg := &workValidate{
		DataHash:  hex.EncodeToString(dataHash),
		Threshold: strconv.FormatUint(threshold, 16),
		Work:      hex.EncodeToString(work),
	}
	bytesData, err := json.Marshal(wg)
	if err != nil {
		return false, err
	}
	validateResult := &workValidateResult{}
	if err := httpRequest(requestUrl+ApiActionValidate, bytesData, validateResult); err != nil {
		return false, err
	}
	if validateResult.Valid == "1" {
		return true, nil
	} else {
		return false, nil
	}
}

func CancelWork(dataHash []byte) error {
	wg := &workCancel{
		DataHash: hex.EncodeToString(dataHash),
	}
	bytesData, err := json.Marshal(wg)
	if err != nil {
		return err
	}
	if err := httpRequest(requestUrl+ApiActionCancel, bytesData, workCancelResult{}); err != nil {
		return err
	}
	return nil
}
