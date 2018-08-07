package api

type TypesApis interface {
	IsValidHexAddress(addrs []string, reply *string) error
	IsValidHexTokenTypeId(ttis []string, reply *string) error
}
