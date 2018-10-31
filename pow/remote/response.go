package remote

type ResponseJson struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Error string      `json:"error"`
	Msg   string      `json:"msg"`
}

type workGenerateResult struct {
	Work string `json:"work"`
}

type workValidateResult struct {
	Valid string `json:"valid"`
}

type workCancelResult struct {
}
