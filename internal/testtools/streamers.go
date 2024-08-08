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

type ErrorStreamer struct {
	Error error
}

func (e ErrorStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	return 0, false
}

func (e ErrorStreamer) Err() error {
	return e.Error
}
