package config

type Vm struct {
	IsVmTest            bool `json:"IsVmTest"`
	IsUseVmTestParam    bool `json:"IsUseVmTestParam"`
	IsUseQuotaTestParam bool `json:"IsUseQuotaTestParam"`
	IsVmDebug           bool `json:"IsVmDebug"`
}
