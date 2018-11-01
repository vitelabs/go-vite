package remote

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"io/ioutil"
	"math/big"
	"net/http"
)

var (
	powClientLog = log15.New("module", "pow_request")
	requestUrl   string
)

const (
	ApiActionGenerate = "/api/generate_work"
	ApiActionValidate = "/api/validate_work"
	ApiActionCancel   = "/api/cancel_work"
	// DefaultThreshold  = "FFFFFFC000000000000000000000000000000000000000000000000000000000"
)

func InitUrl(ip string) {
	requestUrl = "http://" + ip + ":6007"
}

func GenerateWork(dataHash []byte, threshold *big.Int) (*string, error) {
	wg := &workGenerate{
		Threshold: threshold.Text(16),
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

func VaildateWork(dataHash []byte, threshold *big.Int, work []byte) (bool, error) {
	//Threshold(uint): strconv.FormatUint(threshold, 16)
	wg := &workValidate{
		DataHash:  hex.EncodeToString(dataHash),
		Threshold: threshold.Text(16),
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
