package flac

import (
	"fmt"
	"io"

	"github.com/mewkiz/flac"
	"github.com/pkg/errors"

	"github.com/gopxl/beep"
)

// Decode takes a Reader containing audio data in FLAC format and returns a StreamSeekCloser,
// which streams that audio. The Seek method will panic if r is not io.Seeker.
//
// Do not close the supplied Reader, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
//
// Deprecated: Decode has been replaced with DecodeReader and DecodeReadSeeker.
func Decode(r io.Reader) (beep.StreamSeekCloser, beep.Format, error) {
	d := SeekableDecoder{d: Decoder{r: r}}
	var err error

	rs, seeker := r.(io.ReadSeeker)
	if seeker {
		d.d.stream, err = flac.NewSeek(rs)
		d.d.seekEnabled = true
	} else {
		d.d.stream, err = flac.New(r)
	}

	if err != nil {
		if closer, ok := r.(io.Closer); ok {
			if err != nil {
				closer.Close()
			}
		}
		return nil, beep.Format{}, errors.Wrap(err, "flac")
	}

	return &d, d.d.format(), nil
}

func DecodeReader(r io.ReadCloser) (*Decoder, beep.Format, error) {
	d := Decoder{r: r}
	var err error
	d.stream, err = flac.New(r)
	if err != nil {
		r.Close()
		return nil, beep.Format{}, errors.Wrap(err, "flac")
	}

	return &d, d.format(), nil
}

func DecodeReadSeeker(r io.ReadSeekCloser) (*SeekableDecoder, beep.Format, error) {
	d := SeekableDecoder{d: Decoder{r: r, seekEnabled: true}}
	var err error
	d.d.stream, err = flac.NewSeek(r)
	if err != nil {
		r.Close()
		return nil, beep.Format{}, errors.Wrap(err, "flac")
	}

	return &d, d.d.format(), nil
}

type Decoder struct {
	r           io.Reader
	stream      *flac.Stream
	buf         [][2]float64
	pos         int
	err         error
	seekEnabled bool
}

func (d *Decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}
	// Copy samples from buffer.
	j := 0
	for i := range samples {
		if j >= len(d.buf) {
			// refill buffer.
			if err := d.refill(); err != nil {
				d.pos += n
				if err == io.EOF {
					return n, n > 0
				}
				d.err = err
				return 0, false
			}
			j = 0
		}
		samples[i] = d.buf[j]
		j++
		n++
	}
	d.buf = d.buf[j:]
	d.pos += n
	return n, true
}

// refill decodes audio samples to fill the decode buffer.
func (d *Decoder) refill() error {
	// Empty buffer.
	d.buf = d.buf[:0]
	// Parse audio frame.
	frame, err := d.stream.ParseNext()
	if err != nil {
		return err
	}
	// Expand buffer size if needed.
	n := len(frame.Subframes[0].Samples)
	if cap(d.buf) < n {
		d.buf = make([][2]float64, n)
	} else {
		d.buf = d.buf[:n]
	}
	// Decode audio samples.
	bps := d.stream.Info.BitsPerSample
	nchannels := d.stream.Info.NChannels
	s := 1 << (bps - 1)
	q := 1 / float64(s)
	switch {
	case bps == 8 && nchannels == 1:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(int8(frame.Subframes[0].Samples[i])) * q
			d.buf[i][1] = float64(int8(frame.Subframes[0].Samples[i])) * q
		}
	case bps == 16 && nchannels == 1:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(int16(frame.Subframes[0].Samples[i])) * q
			d.buf[i][1] = float64(int16(frame.Subframes[0].Samples[i])) * q
		}
	case bps == 24 && nchannels == 1:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(int32(frame.Subframes[0].Samples[i])) * q
			d.buf[i][1] = float64(int32(frame.Subframes[0].Samples[i])) * q
		}
	case bps == 8 && nchannels >= 2:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(int8(frame.Subframes[0].Samples[i])) * q
			d.buf[i][1] = float64(int8(frame.Subframes[1].Samples[i])) * q
		}
	case bps == 16 && nchannels >= 2:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(int16(frame.Subframes[0].Samples[i])) * q
			d.buf[i][1] = float64(int16(frame.Subframes[1].Samples[i])) * q
		}
	case bps == 24 && nchannels >= 2:
		for i := 0; i < n; i++ {
			d.buf[i][0] = float64(frame.Subframes[0].Samples[i]) * q
			d.buf[i][1] = float64(frame.Subframes[1].Samples[i]) * q
		}
	default:
		panic(fmt.Errorf("support for %d bits-per-sample and %d channels combination not yet implemented", bps, nchannels))
	}
	return nil
}

func (d *Decoder) Err() error {
	return d.err
}

func (d *Decoder) Len() int {
	return int(d.stream.Info.NSamples)
}

func (d *Decoder) Position() int {
	return d.pos
}

func (d *Decoder) Close() error {
	if closer, ok := d.r.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return errors.Wrap(err, "flac")
		}
	}
	return nil
}

func (d *Decoder) format() beep.Format {
	return beep.Format{
		SampleRate:  beep.SampleRate(d.stream.Info.SampleRate),
		NumChannels: int(d.stream.Info.NChannels),
		Precision:   int(d.stream.Info.BitsPerSample / 8),
	}
}

type SeekableDecoder struct {
	d Decoder
}

func (sd *SeekableDecoder) Stream(samples [][2]float64) (n int, ok bool) {
	return sd.d.Stream(samples)
}

func (sd *SeekableDecoder) Err() error {
	return sd.d.Err()
}

func (sd *SeekableDecoder) Len() int {
	return sd.d.Len()
}

func (sd *SeekableDecoder) Position() int {
	return sd.d.Position()
}

func (sd *SeekableDecoder) Close() error {
	return sd.d.Close()
}

// Seek seeks to the start of the frame containing the given absolute sample number.
func (sd *SeekableDecoder) Seek(p int) error {
	if !sd.d.seekEnabled {
		return errors.New("flac.decoder.Seek: not enabled")
	}

	pos, err := sd.d.stream.Seek(uint64(p))
	sd.d.pos = int(pos)
	return err
}
