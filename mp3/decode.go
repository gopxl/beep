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

// Decode takes a ReadCloser containing audio data in MP3 format and returns a StreamSeekCloser,
// which streams that audio. The Seek method will panic if rc is not io.Seeker.
//
// Do not close the supplied ReadSeekCloser, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
//
// Deprecated: Decode has been replaced with DecodeReader and DecodeReadSeeker.
func Decode(rc io.ReadCloser) (ssc beep.StreamSeekCloser, format beep.Format, err error) {
	sc, format, err := DecodeReader(rc)
	if err != nil {
		return nil, beep.Format{}, err
	}

	// Even though rc may not be an io.Seeker, Decode will return a Seeker for backward compatibility.
	ssc = &seekWrapper{
		decoder: *sc.(*decoder),
	}
	return
}

// DecodeReader takes an io.ReadCloser containing audio data in MP3 format and returns a beep.StreamCloser,
// which streams that audio. See DecodeReadSeeker when Len() and Seek() functionality is required.
//
// Do not close the supplied StreamCloser, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
func DecodeReader(r io.ReadCloser) (sc beep.StreamCloser, format beep.Format, err error) {
	d, err := gomp3.NewDecoder(r)
	if err != nil {
		return nil, beep.Format{}, errors.Wrap(err, "mp3")
	}

	format = beep.Format{
		SampleRate:  beep.SampleRate(d.SampleRate()),
		NumChannels: gomp3NumChannels,
		Precision:   gomp3Precision,
	}
	sc = &decoder{
		closer: r,
		d:      d,
		f:      format,
		pos:    0,
		err:    nil,
	}
	return
}

// DecodeReadSeeker takes an ReadSeekCloser containing audio data in MP3 format and returns a beep.StreamSeekCloser,
// which streams that audio. See DecodeReader when the io.Reader isn't seekable.
//
// Do not close the supplied StreamCloser, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
func DecodeReadSeeker(rc io.ReadSeekCloser) (ssc beep.StreamSeekCloser, format beep.Format, err error) {
	var sc beep.StreamCloser
	sc, format, err = DecodeReader(rc)
	if err != nil {
		return nil, beep.Format{}, err
	}

	ssc = &seekWrapper{
		decoder: *sc.(*decoder),
	}
	return
}

type decoder struct {
	closer io.Closer
	d      *gomp3.Decoder
	f      beep.Format
	pos    int
	err    error
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

func (d *decoder) Position() int {
	return d.pos / gomp3BytesPerFrame
}

func (d *decoder) Close() error {
	err := d.closer.Close()
	if err != nil {
		return errors.Wrap(err, "mp3")
	}
	return nil
}

type seekWrapper struct {
	decoder
}

func (d *seekWrapper) Len() int {
	return int(d.d.Length()) / gomp3BytesPerFrame
}

func (d *seekWrapper) Seek(p int) error {
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
