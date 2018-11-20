package config

type Producer struct {
	Producer         bool   `json:"Producer"`
	Coinbase         string `json:"Coinbase"`
	EntropyStorePath string `json:"EntropyStorePath"`
}

//func MergeMinerConfig(cfg *Miner) *Miner {
//	m := GlobalConfig.Miner
//
//	if cfg == nil {
//		return m
//	}
//
//	if cfg.Miner {
//		m.Miner = cfg.Miner
//	}
//
//	if cfg.Coinbase != "" {
//		m.Coinbase = cfg.Coinbase
//	}
//
//	if cfg.MinerInterval != 0 {
//		m.MinerInterval = cfg.MinerInterval
//	}
//
//	return m
//}
