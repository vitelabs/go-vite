package vm

// SafeMul returns multiplication result and whether overflow occurred.
func SafeMul(x, y uint64) (uint64, bool) {
	if x == 0 || y == 0 {
		return 0, false
	}
	return x * y, y > maxUint64/x
}

// SafeAdd returns the result and whether overflow occurred.
func SafeAdd(x, y uint64) (uint64, bool) {
	return x + y, y > maxUint64-x
}
