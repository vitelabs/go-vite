package fork

type ActiveChecker interface {
	IsForkActive(point ForkPointItem) bool
}
