package util

import "cmp"

func Clamp[T cmp.Ordered](x, minV, maxV T) T {
	return max(min(x, maxV), minV)
}
