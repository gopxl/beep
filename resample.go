package beep

import (
	"fmt"
	"math"
)

const resamplerSingleBufferSize = 512

// Resample takes a Streamer which is assumed to stream at the old sample rate and returns a
// Streamer, which streams the data from the original Streamer resampled to the new sample rate.
//
// This is, for example, useful when mixing multiple Streamer with different sample rates, either
// through a beep.Mixer, or through a speaker. Speaker has a constant sample rate. Thus, playing
// Streamer which stream at a different sample rate will lead to a changed speed and pitch of the
// playback.
//
//	sr := beep.SampleRate(48000)
//	speaker.Init(sr, sr.N(time.Second/2))
//	speaker.Play(beep.Resample(3, format.SampleRate, sr, s))
//
// In the example above, the original sample rate of the source is format.SampleRate. We want to play
// it at the speaker's native sample rate and thus we need to resample.
//
// The quality argument specifies the quality of the resampling process. Higher quality implies
// worse performance. Values below 1 or above 64 are invalid and Resample will panic. Here's a table
// for deciding which quality to pick.
//
//	quality | use case
//	--------|---------
//	1       | very high performance, on-the-fly resampling, low quality
//	3-4     | good performance, on-the-fly resampling, good quality
//	6       | higher CPU usage, usually not suitable for on-the-fly resampling, very good quality
//	>6      | even higher CPU usage, for offline resampling, very good quality
//
// Sane quality values are usually below 16. Higher values will consume too much CPU, giving
// negligible quality improvements.
//
// Resample propagates errors from s.
func Resample(quality int, old, new SampleRate, s Streamer) *Resampler {
	return ResampleRatio(quality, float64(old)/float64(new), s)
}

// ResampleRatio is same as Resample, except it takes the ratio of the old and the new sample rate,
// specifically, the old sample rate divided by the new sample rate. Aside from correcting the
// sample rate, this can be used to change the speed of the audio. For example, resampling at the
// ratio of 2 and playing at the original sample rate will cause doubled speed in playback.
func ResampleRatio(quality int, ratio float64, s Streamer) *Resampler {
	if quality < 1 || 64 < quality {
		panic(fmt.Errorf("resample: invalid quality: %d", quality))
	}
	if ratio <= 0 || math.IsInf(ratio, 0) || math.IsNaN(ratio) {
		panic(fmt.Errorf("resample: invalid ratio: %f", ratio))
	}
	return &Resampler{
		s:     s,
		ratio: ratio,
		buf1:  make([][2]float64, resamplerSingleBufferSize),
		buf2:  make([][2]float64, resamplerSingleBufferSize),
		pts:   make([]point, quality*2),
		// The initial value of `off` is set so that the current position is just behind the end
		// of buf2:
		//   current position (0) - len(buf2) = -resamplerSingleBufferSize
		// When the Stream() method is called for the first time, it will determine that neither
		// buf1 nor buf2 contain the required samples because they are both in the past relative to
		// the chosen `off` value. As a result, buf2 will be filled with samples, and `off` will be
		// incremented by resamplerSingleBufferSize, making `off` equal to 0. This will align the
		// start of buf2 with the current position.
		off: -resamplerSingleBufferSize,
		pos: 0.0,
		end: math.MaxInt,
	}
}

// Resampler is a Streamer created by Resample and ResampleRatio functions. It allows dynamic
// changing of the resampling ratio, which can be useful for dynamically changing the speed of
// streaming.
type Resampler struct {
	s          Streamer     // the original streamer
	ratio      float64      // old sample rate / new sample rate
	buf1, buf2 [][2]float64 // buf1 contains previous buf2, new data goes into buf2, buf1 is because interpolation might require old samples
	pts        []point      // pts is for points used for interpolation
	off        int          // off is the position of the start of buf2 in the original data
	pos        float64      // pos is the current position in the resampled data
	end        int          // end is the position after the last sample in the original data
}

// Stream streams the original audio resampled according to the current ratio.
func (r *Resampler) Stream(samples [][2]float64) (n int, ok bool) {
	for len(samples) > 0 {
		// Calculate the current position in the original data.
		wantPos := r.pos * r.ratio

		// Determine the quality*2 closest sample positions for the interpolation.
		// The window has length len(r.pts) and is centered around wantPos.
		windowStart := int(wantPos) - (len(r.pts)-1)/2 // (inclusive)
		windowEnd := int(wantPos) + len(r.pts)/2 + 1   // (exclusive)

		// Prepare the buffers.
		for windowEnd > r.off+resamplerSingleBufferSize {
			// We load into buf1.
			sn, _ := r.s.Stream(r.buf1)
			if sn < len(r.buf1) {
				r.end = r.off + resamplerSingleBufferSize + sn
			}

			// Swap buffers.
			r.buf1, r.buf2 = r.buf2, r.buf1
			r.off += resamplerSingleBufferSize
		}

		// Exit when wantPos is after the end of the original data.
		if int(wantPos) >= r.end {
			return n, n > 0
		}

		// Adjust the window to be within the available buffers.
		windowStart = max(windowStart, 0)
		windowEnd = min(windowEnd, r.end)

		// For each channel...
		for c := range samples[0] {
			// Get the points.
			numPts := windowEnd - windowStart
			pts := r.pts[:numPts]
			for i := range pts {
				x := windowStart + i
				var y float64
				if x < r.off {
					// Sample is in buf1.
					offBuf1 := r.off - resamplerSingleBufferSize
					y = r.buf1[x-offBuf1][c]
				} else {
					// Sample is in buf2.
					y = r.buf2[x-r.off][c]
				}
				pts[i] = point{
					X: float64(x),
					Y: y,
				}
			}

			// Calculate the resampled sample using polynomial interpolation from the
			// quality*2 closest samples.
			samples[0][c] = lagrange(pts, wantPos)
		}

		samples = samples[1:]
		n++
		r.pos++
	}

	return n, true
}

// Err propagates the original Streamer's errors.
func (r *Resampler) Err() error {
	return r.s.Err()
}

// Ratio returns the current resampling ratio.
func (r *Resampler) Ratio() float64 {
	return r.ratio
}

// SetRatio sets the resampling ratio. This does not cause any glitches in the stream.
func (r *Resampler) SetRatio(ratio float64) {
	if ratio <= 0 || math.IsInf(ratio, 0) || math.IsNaN(ratio) {
		panic(fmt.Errorf("resample: invalid ratio: %f", ratio))
	}
	r.pos *= r.ratio / ratio
	r.ratio = ratio
}

// lagrange calculates the value at x of a polynomial of order len(pts)+1 which goes through all
// points in pts
func lagrange(pts []point, x float64) (y float64) {
	y = 0.0
	for j := range pts {
		l := 1.0
		for m := range pts {
			if j == m {
				continue
			}
			l *= (x - pts[m].X) / (pts[j].X - pts[m].X)
		}
		y += pts[j].Y * l
	}
	return y
}

type point struct {
	X, Y float64
}
