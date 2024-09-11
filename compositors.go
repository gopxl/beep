package beep

import (
	"fmt"
	"math"

	"github.com/pkg/errors"
)

// Take returns a Streamer which streams at most num samples from s.
//
// The returned Streamer propagates s's errors through Err.
func Take(num int, s Streamer) Streamer {
	return &take{
		s:       s,
		remains: num,
	}
}

type take struct {
	s       Streamer
	remains int
}

func (t *take) Stream(samples [][2]float64) (n int, ok bool) {
	if t.remains <= 0 {
		return 0, false
	}
	toStream := min(t.remains, len(samples))
	n, ok = t.s.Stream(samples[:toStream])
	t.remains -= n
	return n, ok
}

func (t *take) Err() error {
	return t.s.Err()
}

// Loop takes a StreamSeeker and plays it count times. If count is negative, s is looped infinitely.
//
// The returned Streamer propagates s's errors.
//
// Deprecated: use Loop2 instead. A call to Loop can be rewritten as follows:
// - beep.Loop(-1, s) -> beep.Loop2(s)
// - beep.Loop(0, s) -> no longer supported, use beep.Ctrl instead.
// - beep.Loop(3, s) -> beep.Loop2(s, beep.LoopTimes(2))
// Note that beep.LoopTimes takes the number of repeats instead of the number of total plays.
func Loop(count int, s StreamSeeker) Streamer {
	return &loop{
		s:       s,
		remains: count,
	}
}

type loop struct {
	s       StreamSeeker
	remains int
}

func (l *loop) Stream(samples [][2]float64) (n int, ok bool) {
	if l.remains == 0 || l.s.Err() != nil {
		return 0, false
	}
	for len(samples) > 0 {
		sn, sok := l.s.Stream(samples)
		if !sok {
			if l.remains > 0 {
				l.remains--
			}
			if l.remains == 0 {
				break
			}
			err := l.s.Seek(0)
			if err != nil {
				return n, true
			}
			continue
		}
		samples = samples[sn:]
		n += sn
	}
	return n, true
}

func (l *loop) Err() error {
	return l.s.Err()
}

type LoopOption func(opts *loop2)

// LoopTimes sets how many times the source stream will repeat. If a section is defined
// by LoopStart, LoopEnd, or LoopBetween, only that section will repeat.
// A value of 0 plays the stream or section once (no repetition); 1 plays it twice, and so on.
func LoopTimes(times int) LoopOption {
	if times < 0 {
		panic("invalid argument to LoopTimes; times cannot be negative")
	}
	return func(loop *loop2) {
		loop.remains = times
	}
}

// LoopStart sets the position in the source stream to which it returns (using Seek())
// after reaching the end of the stream or the position set using LoopEnd. The samples
// before this position are played once before the loop begins.
func LoopStart(pos int) LoopOption {
	if pos < 0 {
		panic("invalid argument to LoopStart; pos cannot be negative")
	}
	return func(loop *loop2) {
		loop.start = pos
	}
}

// LoopEnd sets the position (exclusive) in the source stream up to which the stream plays
// before returning (seeking) back to the start of the stream or the position set by LoopStart.
// The samples after this position are played once after looping completes.
func LoopEnd(pos int) LoopOption {
	if pos < 0 {
		panic("invalid argument to LoopEnd; pos cannot be negative")
	}
	return func(loop *loop2) {
		loop.end = pos
	}
}

// LoopBetween sets both the LoopStart and LoopEnd positions simultaneously, specifying
// the section of the stream that will be looped.
func LoopBetween(start, end int) LoopOption {
	return func(opts *loop2) {
		LoopStart(start)(opts)
		LoopEnd(end)(opts)
	}
}

// Loop2 takes a StreamSeeker and repeats it according to the specified options. If no LoopTimes
// option is provided, the stream loops indefinitely. LoopStart, LoopEnd, or LoopBetween can define
// a specific section of the stream to loop. Samples before the start and after the end positions
// are played once before and after the looping section, respectively.
//
// The returned Streamer propagates any errors from s.
func Loop2(s StreamSeeker, opts ...LoopOption) (Streamer, error) {
	l := &loop2{
		s:       s,
		remains: -1, // indefinitely
		start:   0,
		end:     math.MaxInt,
	}
	for _, opt := range opts {
		opt(l)
	}

	n := s.Len()
	if l.start >= n {
		return nil, errors.New(fmt.Sprintf("invalid argument to Loop2; start position %d must be smaller than the source streamer length %d", l.start, n))
	}
	if l.start >= l.end {
		return nil, errors.New(fmt.Sprintf("invalid argument to Loop2; start position %d must be smaller than the end position %d", l.start, l.end))
	}
	l.end = min(l.end, n)

	return l, nil
}

type loop2 struct {
	s       StreamSeeker
	remains int // number of seeks remaining.
	start   int // start position in the stream where looping begins. Samples before this position are played once before the first loop.
	end     int // end position in the stream where looping ends and restarts from `start`.
	err     error
}

func (l *loop2) Stream(samples [][2]float64) (n int, ok bool) {
	if l.err != nil {
		return 0, false
	}
	for len(samples) > 0 {
		toStream := len(samples)
		if l.remains != 0 {
			samplesUntilEnd := l.end - l.s.Position()
			if samplesUntilEnd <= 0 {
				// End of loop, reset the position and decrease the loop count.
				if l.remains > 0 {
					l.remains--
				}
				if err := l.s.Seek(l.start); err != nil {
					l.err = err
					return n, true
				}
				continue
			}
			// Stream only up to the end of the loop.
			toStream = min(samplesUntilEnd, toStream)
		}

		sn, sok := l.s.Stream(samples[:toStream])
		n += sn
		if sn < toStream || !sok {
			l.err = l.s.Err()
			return n, n > 0
		}
		samples = samples[sn:]
	}
	return n, true
}

func (l *loop2) Err() error {
	return l.err
}

// Seq takes zero or more Streamers and returns a Streamer which streams them one by one without pauses.
//
// Seq does not propagate errors from the Streamers.
func Seq(s ...Streamer) Streamer {
	i := 0
	return StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		for i < len(s) && len(samples) > 0 {
			sn, sok := s[i].Stream(samples)
			samples = samples[sn:]
			n, ok = n+sn, ok || sok
			if !sok {
				i++
			}
		}
		return n, ok
	})
}

// Mix takes zero or more Streamers and returns a Streamer which streams them mixed together.
//
// Mix does not propagate errors from the Streamers.
func Mix(s ...Streamer) Streamer {
	return &Mixer{
		streamers:     s,
		stopWhenEmpty: true,
	}
}

// Dup returns two Streamers which both stream the same data as the original s. The two Streamers
// can't be used concurrently without synchronization.
func Dup(s Streamer) (t, u Streamer) {
	var tBuf, uBuf [][2]float64
	return &dup{&tBuf, &uBuf, s}, &dup{&uBuf, &tBuf, s}
}

type dup struct {
	myBuf, itsBuf *[][2]float64
	s             Streamer
}

func (d *dup) Stream(samples [][2]float64) (n int, ok bool) {
	buf := *d.myBuf
	n = copy(samples, buf)
	ok = len(buf) > 0
	buf = buf[n:]
	samples = samples[n:]
	*d.myBuf = buf

	if len(samples) > 0 {
		sn, sok := d.s.Stream(samples)
		n += sn
		ok = ok || sok
		*d.itsBuf = append(*d.itsBuf, samples[:sn]...)
	}

	return n, ok
}

func (d *dup) Err() error {
	return d.s.Err()
}
