package api_interface

type CommonApis interface {
	IsValidHexAddress(addrs []string, reply *string) error
	IsValidHexTokenTypeId(ttis []string, reply *string) error
}
