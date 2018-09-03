package api

type TypesApis interface {
	IsValidHexAddress(addrs string) bool
	IsValidHexTokenTypeId(ttis string) bool
}
