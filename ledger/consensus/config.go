package consensus

type ConsensusCfg struct {
	enablePuppet bool
}

func DefaultCfg() *ConsensusCfg {
	return &ConsensusCfg{
		enablePuppet: false,
	}
}

func Cfg(enablePuppet bool) *ConsensusCfg {
	return &ConsensusCfg{enablePuppet: enablePuppet}
}
