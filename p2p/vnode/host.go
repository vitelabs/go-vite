package vnode

type HostType byte

const (
	HostIPv4   HostType = 1
	HostIPv6   HostType = 2
	HostIP     HostType = 3
	HostDomain HostType = 4
)

func (ht HostType) Is(ht2 HostType) bool {
	return ht&ht2 > 0
}
