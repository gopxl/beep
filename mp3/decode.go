// Package mp3 implements audio data decoding in MP3 format.
package mp3

import (
	"fmt"
	"io"

	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/pkg/errors"

	"github.com/gopxl/beep"
)

const (
	gomp3NumChannels   = 2
	gomp3Precision     = 2
	gomp3BytesPerFrame = gomp3NumChannels * gomp3Precision
)

// Decode takes an io.Reader containing audio data in MP3 format and returns a StreamSeeker,
// which streams that audio. The Seek method will panic if rc is not io.Seeker.
func Decode(r io.Reader) (s beep.StreamSeeker, format beep.Format, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "mp3")
		}
	}()
	d, err := gomp3.NewDecoder(r)
	if err != nil {
		return nil, beep.Format{}, err
	}
	format = beep.Format{
		SampleRate:  beep.SampleRate(d.SampleRate()),
		NumChannels: gomp3NumChannels,
		Precision:   gomp3Precision,
	}
	return &decoder{d, format, 0, nil}, format, nil
}

type decoder struct {
	d   *gomp3.Decoder
	f   beep.Format
	pos int
	err error
}

func (d *decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}
	var tmp [gomp3BytesPerFrame]byte
	for i := range samples {
		dn, err := d.d.Read(tmp[:])
		if dn == len(tmp) {
			samples[i], _ = d.f.DecodeSigned(tmp[:])
			d.pos += dn
			n++
			ok = true
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			d.err = errors.Wrap(err, "mp3")
			break
		}
	}
	return n, ok
}

func (d *decoder) Err() error {
	return d.err
}

func (d *decoder) Len() int {
	return int(d.d.Length()) / gomp3BytesPerFrame
}

func (d *decoder) Position() int {
	return d.pos / gomp3BytesPerFrame
}

func (d *decoder) Seek(p int) error {
	if p < 0 || d.Len() < p {
		return fmt.Errorf("mp3: seek position %v out of range [%v, %v]", p, 0, d.Len())
	}
	_, err := d.d.Seek(int64(p)*gomp3BytesPerFrame, io.SeekStart)
	if err != nil {
		return errors.Wrap(err, "mp3")
	}
	d.pos = p * gomp3BytesPerFrame
	return nil
}
