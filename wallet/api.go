package wallet

type ExternalAPI interface {
	ListAddress(v interface{}, reply *string) error

	NewAddress(pwd []string, reply *string) error
}
