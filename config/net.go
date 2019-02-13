package config

type Net struct {
	Single      bool     `json:"Single"`
	FileAddress string   `json:"FileAddress"`
	Topology    []string `json:"Topology"`
	Topic       string   `json:"Topic"`
	Interval    int64    `json:"Interval"`
	TopoEnabled bool     `json:"TopoEnabled"`
}
