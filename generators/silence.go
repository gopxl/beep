package generators

import "github.com/gopxl/beep"

// Silence returns a Streamer which streams num samples of silence. If num is negative, silence is
// streamed forever.
func Silence(num int) beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		if num == 0 {
			return 0, false
		}
		if 0 < num && num < len(samples) {
			samples = samples[:num]
		}
		for i := range samples {
			samples[i] = [2]float64{}
		}
		if num > 0 {
			num -= len(samples)
		}
		return len(samples), true
	})
}
