package config

type KafkaProducer struct {
	BrokerList []string
	Topic      string
}

type Chain struct {
	KafkaProducers []*KafkaProducer
	OpenBlackBlock bool
	LedgerGcRetain uint64
	GenesisFile    string
	LedgerGc       bool
}
