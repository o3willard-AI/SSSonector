package throttle

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// minInt returns the minimum of two int values
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two int values
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// roundUpToMultiple rounds up a number to the nearest multiple
func roundUpToMultiple(n, multiple int) int {
	if multiple == 0 {
		return n
	}
	remainder := n % multiple
	if remainder == 0 {
		return n
	}
	return n + multiple - remainder
}
