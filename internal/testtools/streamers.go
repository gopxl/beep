package testtools

import (
	"github.com/gopxl/beep"
	"math/rand"
)

// RandomDataStreamer generates numSamples random samples and returns a StreamSeeker to stream them.
func RandomDataStreamer(numSamples int) (s beep.StreamSeeker, data [][2]float64) {
	data = make([][2]float64, numSamples)
	for i := range data {
		data[i][0] = rand.Float64()*2 - 1
		data[i][1] = rand.Float64()*2 - 1
	}
	return &dataStreamer{data, 0}, data
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
