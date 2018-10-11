package config

type Net struct {
	Single   bool     `json:"Single"`
	FilePort uint16   `json:"Port"`
	Topology []string `json:"Topology"`
	Topic    string   `json:"Topic"`
	Interval int64    `json:"Interval"`
}
