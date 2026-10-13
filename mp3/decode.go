// Package mp3 implements audio data decoding in MP3 format.
package mp3

import (
	"fmt"
	"io"

	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/pkg/errors"

	"github.com/gopxl/beep/v2"
)

const (
	gomp3NumChannels   = 2
	gomp3Precision     = 2
	gomp3BytesPerFrame = gomp3NumChannels * gomp3Precision
)

// Decode takes a ReadCloser containing audio data in MP3 format and returns a StreamSeekCloser,
// which streams that audio. The Seek method will panic if rc is not io.Seeker.
//
// When the first frame holds a Xing, Info or VBRI header, the header frame is not streamed.
// When it also holds a LAME tag, the encoder delay, the decoder delay and the end padding are
// not streamed either, so gapless tracks play without a gap. The end padding is only trimmed
// when rc is an io.Seeker, because the length of the stream is unknown otherwise.
//
// Do not close the supplied ReadSeekCloser, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
func Decode(rc io.ReadCloser) (s beep.StreamSeekCloser, format beep.Format, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "mp3")
		}
	}()
	r, g := readGapless(rc)
	d, err := gomp3.NewDecoder(r)
	if err != nil {
		return nil, beep.Format{}, err
	}
	format = beep.Format{
		SampleRate:  beep.SampleRate(d.SampleRate()),
		NumChannels: gomp3NumChannels,
		Precision:   gomp3Precision,
	}
	dec := &decoder{closer: rc, d: d, f: format, length: int(d.Length())}
	skip := g.skip * gomp3BytesPerFrame
	padding := g.padding * gomp3BytesPerFrame
	if skip > 0 && (dec.length < 0 || dec.length-skip-padding > 0) {
		if _, err := io.CopyN(io.Discard, d, int64(skip)); err != nil && err != io.EOF {
			return nil, beep.Format{}, err
		}
		dec.skip = skip
		if dec.length >= 0 {
			dec.length -= skip + padding
		}
	}
	return dec, format, nil
}

type decoder struct {
	closer io.Closer
	d      *gomp3.Decoder
	f      beep.Format
	pos    int // bytes streamed since the first track sample
	skip   int // bytes of gapless delay before the first track sample
	length int // bytes of track samples; -1 when unknown
	err    error
}

func (d *decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}
	var tmp [gomp3BytesPerFrame]byte
	for i := range samples {
		if d.length >= 0 && d.pos >= d.length {
			break
		}
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
	if d.length < 0 {
		return 0
	}
	return d.length / gomp3BytesPerFrame
}

func (d *decoder) Position() int {
	return d.pos / gomp3BytesPerFrame
}

func (d *decoder) Seek(p int) error {
	if p < 0 || d.Len() < p {
		return fmt.Errorf("mp3: seek position %v out of range [%v, %v]", p, 0, d.Len())
	}
	_, err := d.d.Seek(int64(p)*gomp3BytesPerFrame+int64(d.skip), io.SeekStart)
	if err != nil {
		return errors.Wrap(err, "mp3")
	}
	d.pos = p * gomp3BytesPerFrame
	return nil
}

func (d *decoder) Close() error {
	err := d.closer.Close()
	if err != nil {
		return errors.Wrap(err, "mp3")
	}
	return nil
}
