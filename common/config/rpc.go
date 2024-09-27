package config

type Rpc struct {
	RPCEnabled       bool     `json:"RPCEnabled"`
	IPCEnabled       bool     `json:"IPCEnabled"`
	WSEnabled        bool     `json:"WSEnabled"`
	IPCPath          string   `json:"IPCPath"`
	HttpHost         string   `json:"HttpHost"`
	HttpPort         int      `json:"HttpPort"`
	HttpVirtualHosts []string `json:"HttpVirtualHosts"`
	WSHost           string   `json:"WSHost"`
	WSPort           int      `json:"WSPort"`
}
