package config

type NodeReward struct {
	RewardAddr string  `json:"RewardAddr"`
	Name       string  `json:"Name"`
	SecretPub  *string `json:"SecretPub"`
}
