package config

type KafkaProducer struct {
	BrokerList []string
	Topic      string
}

type Chain struct {
	KafkaProducers []*KafkaProducer
	OpenBlackBlock bool
}
