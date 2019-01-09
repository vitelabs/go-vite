package config

type Vm struct {
	IsVmTest         bool `json:"IsVmTest"`
	IsUseVmTestParam bool `json:"IsUseVmTestParam"`
	IsVmDebug        bool `json:"IsVmDebug"`
}
