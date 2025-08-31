package effects

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep/v2"
)

// SmoothFader handles smoothly starting and stopping playback via quick but
// gentle fades to avoid popping. The general concept is described here:
// https://en.m.wikipedia.org/wiki/Hann_function
type SmoothFader struct {
	Streamer beep.Streamer

	// are we fading?
	fadingOut   atomic.Bool
	fadingIn    bool
	firstStream bool

	// true if stopped
	stopped bool

	// how many samples we've left to fade
	fadeIndex atomic.Int32

	hannWindow []float64
}

func NewSmoothFader(streamer beep.Streamer, sr beep.SampleRate) *SmoothFader {
	s := &SmoothFader{
		Streamer:    streamer,
		firstStream: true,
	}

	// pre-calculate _half_ a hann window: we can step forwards and
	// backwards (no need to calculate the entire window).
	N := sr.N(5 * time.Millisecond)
	s.hannWindow = make([]float64, N)
	for n := 0; n < len(s.hannWindow); n++ {
		s.hannWindow[n] = 0.5 * (1 - math.Cos(math.Pi*float64(n)/float64(N-1)))
	}

	return s
}

// Stream streams the wrapped Streamer with volume adjusted according to Base & Volume
// and whether we are fading in or out
func (s *SmoothFader) Stream(samples [][2]float64) (n int, ok bool) {
	// already stopped, nothing to do
	if s.stopped {
		return 0, false
	}

	// fade in if we are just starting and the first sample is
	// loud enough to pop!
	if s.firstStream && (math.Abs(samples[0][0]) > 0.01 || math.Abs(samples[0][1]) > 0.01) {
		s.fadeIndex.Store(0)
		s.fadingIn = true
	}
	s.firstStream = false

	n, ok = s.Streamer.Stream(samples)
	var gain float64
	for i := range samples[:n] {
		switch {
		case s.fadingIn:
			idx := s.fadeIndex.Load()
			gain = s.hannWindow[idx]
			s.fadeIndex.Add(2) // fade in twice as fast as we fade out
			if s.fadeIndex.Load() >= int32(len(s.hannWindow)) {
				// we are done with fading in
				s.fadingIn = false
				s.fadeIndex.Store(0)
			}
		case s.fadingOut.Load():
			// to fade out, we walk _backwards_ through the window
			idx := s.fadeIndex.Load()
			gain = s.hannWindow[idx]
			s.fadeIndex.Add(-1)
		default:
			gain = 1
		}

		samples[i][0] *= gain
		samples[i][1] *= gain

		if s.fadingOut.Load() && s.fadeIndex.Load() <= int32(0) {
			s.stopped = true
			return i, false
		}
	}
	return n, ok
}

// Stop fades out audio and stops streaming.
// Stop can be called from another thread.
func (s *SmoothFader) Stop() {
	s.fadeIndex.Store(int32(len(s.hannWindow) - 1))
	s.fadingOut.Store(true)
}

// Err propagates the wrapped Streamer's errors.
func (s *SmoothFader) Err() error {
	return s.Streamer.Err()
}
