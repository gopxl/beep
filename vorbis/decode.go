// Package vorbis implements audio data decoding in oggvorbis format.
package vorbis

import (
	"io"

	"github.com/jfreymuth/oggvorbis"
	"github.com/pkg/errors"

	"github.com/gopxl/beep"
)

const (
	govorbisPrecision = 2
)

// Decode takes a ReadCloser containing audio data in ogg/vorbis format and returns a StreamSeekCloser,
// which streams that audio. The Seek method will panic if rc is not io.Seeker.
//
// Do not close the supplied ReadSeekCloser, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
func Decode(rc io.ReadCloser) (s beep.StreamSeekCloser, format beep.Format, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "ogg/vorbis")
		}
	}()
	d, err := oggvorbis.NewReader(rc)
	if err != nil {
		return nil, beep.Format{}, err
	}

	channels := d.Channels()
	if channels > 2 {
		channels = 2
	}

	format = beep.Format{
		SampleRate:  beep.SampleRate(d.SampleRate()),
		NumChannels: channels,
		Precision:   govorbisPrecision,
	}

	return &decoder{rc, d, make([]float32, d.Channels()), nil}, format, nil
}

type decoder struct {
	closer io.Closer
	d      *oggvorbis.Reader
	tmp    []float32
	err    error
}

func (d *decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}

	// https://xiph.org/vorbis/doc/vorbisfile/ov_read.html
	// https://xiph.org/vorbis/doc/Vorbis_I_spec.html#x1-810004.3.9
	var leftChannelIndex, rightChannelIndex int
	switch d.d.Channels() {
	case 0:
		d.err = errors.New("ogg/vorbis: invalid channel count: 0")
		return 0, false
	case 1:
		leftChannelIndex = 0
		rightChannelIndex = 0
	case 2:
	case 4:
		leftChannelIndex = 0
		rightChannelIndex = 1
	case 3:
	case 5:
	case 6:
	case 7:
	case 8:
	default:
		leftChannelIndex = 0
		rightChannelIndex = 2
	}

	for i := range samples {
		dn, err := d.d.Read(d.tmp)
		if dn == 0 {
			break
		}
		if dn < len(d.tmp) {
			d.err = errors.New("ogg/vorbis: could only read part of a frame")
			return 0, false
		}
		if err == io.EOF {
			return 0, false
		}
		if err != nil {
			d.err = errors.Wrap(err, "ogg/vorbis")
			return 0, false
		}

		samples[i][0] = float64(d.tmp[leftChannelIndex])
		samples[i][1] = float64(d.tmp[rightChannelIndex])
		n++
	}
	return n, n > 0
}

func (d *decoder) Err() error {
	return d.err
}

func (d *decoder) Len() int {
	return int(d.d.Length())
}

func (d *decoder) Position() int {
	return int(d.d.Position())
}

func (d *decoder) Seek(p int) error {
	err := d.d.SetPosition(int64(p))
	if err != nil {
		return errors.Wrap(err, "ogg/vorbis")
	}
	return nil
}

func (d *decoder) Close() error {
	err := d.closer.Close()
	if err != nil {
		return errors.Wrap(err, "ogg/vorbis")
	}
	return nil
}
