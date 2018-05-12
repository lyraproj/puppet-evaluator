package utils

func Int64Pow(base, exp int64) int64 {
	if base == 0 || exp <= 0 {
		return 0
	}
	if base == 1 {
		return 1
	}
	result := int64(1)
	if base < 0 {
		base = -base
		if exp&1 == 1 {
			result = -result
		}
	}
	for exp > 0 {
		if exp&1 == 1 {
			result *= base
		}
		base *= base
		exp >>= 1
	}
	return result
}
