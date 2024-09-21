package testtools

import (
	"math/rand"

	"github.com/gopxl/beep/v2"
)

// RandomDataStreamer generates numSamples random samples and returns a StreamSeeker to stream them.
func RandomDataStreamer(numSamples int) (s beep.StreamSeeker, data [][2]float64) {
	data = make([][2]float64, numSamples)
	for i := range data {
		data[i] = [2]float64{
			rand.Float64()*2 - 1,
			rand.Float64()*2 - 1,
		}
	}
	return NewDataStreamer(data), data
}

// NewSequentialDataStreamer creates a streamer which streams samples with values {0, 0}, {1, 1}, {2, 2}, etc.
// Note that this aren't valid sample values in the range of [-1, 1], but it can nonetheless
// be useful for testing.
func NewSequentialDataStreamer(numSamples int) (s beep.StreamSeeker, data [][2]float64) {
	data = make([][2]float64, numSamples)
	for i := range data {
		data[i] = [2]float64{float64(i), float64(i)}
	}
	return NewDataStreamer(data), data
}

// NewDataStreamer creates a streamer which streams the given data.
func NewDataStreamer(data [][2]float64) (s beep.StreamSeeker) {
	return &dataStreamer{data, 0}
}

type dataStreamer struct {
	data [][2]float64
	pos  int
}

func (ds *dataStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if ds.pos >= len(ds.data) {
		return 0, false
	}
	n = copy(samples, ds.data[ds.pos:])
	ds.pos += n
	return n, true
}

func (ds *dataStreamer) Err() error {
	return nil
}

func (ds *dataStreamer) Len() int {
	return len(ds.data)
}

func (ds *dataStreamer) Position() int {
	return ds.pos
}

func (ds *dataStreamer) Seek(p int) error {
	ds.pos = p
	return nil
}

// NewErrorStreamer returns a streamer which errors immediately with the given err.
func NewErrorStreamer(err error) beep.StreamSeeker {
	return &ErrorStreamer{
		s: beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
			panic("unreachable")
		}),
		samplesLeft: 0,
		Error:       err,
	}
}

// NewDelayedErrorStreamer wraps streamer s but returns an error after numSamples have been streamed.
func NewDelayedErrorStreamer(s beep.Streamer, numSamples int, err error) beep.StreamSeeker {
	return &ErrorStreamer{
		s:           s,
		samplesLeft: numSamples,
		Error:       err,
	}
}

type ErrorStreamer struct {
	s           beep.Streamer
	samplesLeft int
	Error       error
}

func (e *ErrorStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if e.samplesLeft == 0 {
		return 0, false
	}

	//toStream := min(e.samplesLeft, len(samples))
	toStream := e.samplesLeft
	if toStream > len(samples) {
		toStream = len(samples)
	}
	n, ok = e.s.Stream(samples[:toStream])
	e.samplesLeft -= n

	return n, ok
}

func (e *ErrorStreamer) Err() error {
	if e.samplesLeft == 0 {
		return e.Error
	} else {
		return e.s.Err()
	}
}

func (e *ErrorStreamer) Seek(p int) error {
	if s, ok := e.s.(beep.StreamSeeker); ok {
		return s.Seek(p)
	}
	panic("source streamer is not a beep.StreamSeeker")
}

func (e *ErrorStreamer) Len() int {
	if s, ok := e.s.(beep.StreamSeeker); ok {
		return s.Len()
	}
	panic("source streamer is not a beep.StreamSeeker")
}

func (e *ErrorStreamer) Position() int {
	if s, ok := e.s.(beep.StreamSeeker); ok {
		return s.Position()
	}
	panic("source streamer is not a beep.StreamSeeker")
}

func NewSeekErrorStreamer(s beep.StreamSeeker, err error) *SeekErrorStreamer {
	return &SeekErrorStreamer{
		s:   s,
		err: err,
	}
}

type SeekErrorStreamer struct {
	s   beep.StreamSeeker
	err error
}

func (s *SeekErrorStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	return s.s.Stream(samples)
}

func (s *SeekErrorStreamer) Err() error {
	return s.s.Err()
}

func (s *SeekErrorStreamer) Len() int {
	return s.s.Len()
}

func (s *SeekErrorStreamer) Position() int {
	return s.s.Position()
}

func (s *SeekErrorStreamer) Seek(p int) error {
	return s.err
}
