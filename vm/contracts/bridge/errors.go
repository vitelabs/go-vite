package bridge

import "errors"

var (
	HeaderErr           = errors.New("header error")
	ParentHeaderErr     = errors.New("prev header error")
	HeaderDuplicatedErr = errors.New("header duplicated error")
	NextLatestBlockErr  = errors.New("next latest block error")
	InitErr             = errors.New("init error")
	SubmitErr           = errors.New("submit error")

	ProofReceiptErr = errors.New("proof receipt err")
)
