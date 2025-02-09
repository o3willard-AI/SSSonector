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
