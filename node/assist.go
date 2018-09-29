package node

func CopyConf(conf *Config) *Config {
	confCopy := *conf
	return &confCopy
}
