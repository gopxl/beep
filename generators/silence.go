package generators

import "github.com/gopxl/beep"

// Silence returns a Streamer which streams num samples of silence. If num is negative, silence is
// streamed forever.
func Silence(num int) beep.Streamer {
	if num < 0 {
		return beep.StreamerFunc(func(samples [][2]float64) (m int, ok bool) {
			for i := range samples {
				samples[i] = [2]float64{}
			}
			return len(samples), true
		})
	}

	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		if num <= 0 {
			return 0, false
		}
		if num < len(samples) {
			samples = samples[:num]
		}
		for i := range samples {
			samples[i] = [2]float64{}
		}
		num -= len(samples)

		return len(samples), true
	})
}
