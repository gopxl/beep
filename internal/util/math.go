package util

func ClampFloat64(x, minV, maxV float64) float64 {
	if x <= minV {
		return minV
	}
	if x >= maxV {
		return maxV
	}
	return x
}
