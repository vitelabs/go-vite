package api_interface

type CommonApis interface {
	IsValidHexAddress(addrs []string, reply *string) error
	IsValidHexTokenTypeId(ttis []string, reply *string) error
	// if it  exists a log dir it will return it else return empty string
	LogDir(noop interface{}, reply *string) error
}
