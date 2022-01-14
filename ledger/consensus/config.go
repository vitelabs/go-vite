package consensus

type ConsensusCfg struct {
	EnablePuppet bool
}

func DefaultCfg() *ConsensusCfg {
	return &ConsensusCfg{
		EnablePuppet: false,
	}
}

func Cfg(enablePuppet bool) *ConsensusCfg {
	return &ConsensusCfg{EnablePuppet: enablePuppet}
}
