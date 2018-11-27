package remote

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
