package common

import (
	"fmt"
	"testing"
)

func TestMockAddress(t *testing.T) {
	address := MockAddress(99)
	fmt.Println(address.String())
}

func TestMockHash(t *testing.T) {
	hash := MockHash(99)
	fmt.Println(hash.String())
}
