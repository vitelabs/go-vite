package vnode

import "fmt"

func ExampleHostType_Is() {
	fmt.Println(HostIPv4.Is(HostIPv4))
	fmt.Println(HostIPv6.Is(HostIPv6))
	fmt.Println(HostIP.Is(HostIP))
	fmt.Println(HostDomain.Is(HostDomain))
	fmt.Println(HostIPv4.Is(HostIP))
	fmt.Println(HostIPv6.Is(HostIP))
	fmt.Println(HostIPv4.Is(HostIPv6))
	fmt.Println(HostIPv6.Is(HostIPv4))
	fmt.Println(HostIPv4.Is(HostDomain))
	fmt.Println(HostIPv6.Is(HostDomain))
	fmt.Println(HostDomain.Is(HostIP))
	fmt.Println(HostIP.Is(HostDomain))
	// Output:
	// true
	// true
	// true
	// true
	// true
	// true
	// false
	// false
	// false
	// false
	// false
	// false
}
