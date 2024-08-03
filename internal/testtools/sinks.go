package testtools

import "github.com/gopxl/beep/v2"

// Collect drains Streamer s and returns all the samples it streamed.
func Collect(s beep.Streamer) [][2]float64 {
	var (
		result [][2]float64
		buf    [479][2]float64
	)
	for {
		n, ok := s.Stream(buf[:])
		if !ok {
			return result
		}
		result = append(result, buf[:n]...)
	}
}

// CollectNum collects num samples from the streamer in chunks and returns them all.
func CollectNum(num int, s beep.Streamer) [][2]float64 {
	return Collect(beep.Take(num, s))
}
