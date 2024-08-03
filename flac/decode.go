package flac

import (
	"fmt"
	"io"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/pkg/errors"

	"github.com/gopxl/beep/v2"
)

// Decode takes a Reader containing audio data in FLAC format and returns a StreamSeekCloser,
// which streams that audio. The Seek method will panic if r is not io.Seeker.
//
// Do not close the supplied Reader, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
func Decode(r io.Reader) (s beep.StreamSeekCloser, format beep.Format, err error) {
	d := decoder{r: r}
	defer func() { // hacky way to always close r if an error occurred
		if closer, ok := d.r.(io.Closer); ok {
			if err != nil {
				closer.Close()
			}
		}
	}()

	rs, seeker := r.(io.ReadSeeker)
	if seeker {
		d.stream, err = flac.NewSeek(rs)
		d.seekEnabled = true
	} else {
		d.stream, err = flac.New(r)
	}
	if err != nil {
		return nil, beep.Format{}, errors.Wrap(err, "flac")
	}

	// Read the first frame
	d.frame, err = d.stream.ParseNext()
	if err != nil {
		return nil, beep.Format{}, errors.Wrap(err, "flac")
	}

	format = beep.Format{
		SampleRate:  beep.SampleRate(d.stream.Info.SampleRate),
		NumChannels: int(d.stream.Info.NChannels),
		Precision:   int(d.stream.Info.BitsPerSample / 8),
	}
	return &d, format, nil
}

type decoder struct {
	r           io.Reader
	stream      *flac.Stream
	frame       *frame.Frame
	posInFrame  int
	err         error
	seekEnabled bool
}

func (d *decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil || d.frame == nil {
		return 0, false
	}

	for len(samples) > 0 {
		samplesLeft := int(d.frame.BlockSize) - d.posInFrame
		if samplesLeft <= 0 {
			// Read next frame
			var err error
			d.frame, err = d.stream.ParseNext()
			if err != nil {
				d.frame = nil
				if err == io.EOF {
					return n, n > 0
				}
				d.err = errors.Wrap(err, "flac")
				return 0, false
			}
			d.posInFrame = 0
			continue
		}

		toFill := min(samplesLeft, len(samples))
		d.decodeFrameRangeInto(d.frame, d.posInFrame, toFill, samples)
		d.posInFrame += toFill
		n += toFill
		samples = samples[toFill:]
	}

	return n, true
}

// decodeFrameRangeInto decodes the samples frame from the position `start` up to `start + num`
// and stores them in Beep's format into the provided slice `into`.
func (d *decoder) decodeFrameRangeInto(frame *frame.Frame, start, num int, into [][2]float64) {
	bps := d.stream.Info.BitsPerSample
	numChannels := d.stream.Info.NChannels
	s := 1 << (bps - 1)
	q := 1 / float64(s)
	switch {
	case bps == 8 && numChannels == 1:
		for i := 0; i < num; i++ {
			into[i][0] = float64(frame.Subframes[0].Samples[start+i]) * q
			into[i][1] = into[i][0]
		}
	case bps == 16 && numChannels == 1:
		for i := 0; i < num; i++ {
			into[i][0] = float64(frame.Subframes[0].Samples[start+i]) * q
			into[i][1] = into[i][0]
		}
	case bps == 24 && numChannels == 1:
		for i := 0; i < num; i++ {
			into[i][0] = float64(frame.Subframes[0].Samples[start+i]) * q
			into[i][1] = into[i][0]
		}
	case bps == 8 && numChannels >= 2:
		for i := 0; i < num; i++ {
			into[i][0] = float64(frame.Subframes[0].Samples[start+i]) * q
			into[i][1] = float64(frame.Subframes[1].Samples[start+i]) * q
		}
	case bps == 16 && numChannels >= 2:
		for i := 0; i < num; i++ {
			into[i][0] = float64(frame.Subframes[0].Samples[start+i]) * q
			into[i][1] = float64(frame.Subframes[1].Samples[start+i]) * q
		}
	case bps == 24 && numChannels >= 2:
		for i := 0; i < num; i++ {
			into[i][0] = float64(frame.Subframes[0].Samples[start+i]) * q
			into[i][1] = float64(frame.Subframes[1].Samples[start+i]) * q
		}
	default:
		panic(fmt.Errorf("flac: support for %d bits-per-sample and %d channels combination not yet implemented", bps, numChannels))
	}
}

func (d *decoder) Err() error {
	return d.err
}

func (d *decoder) Len() int {
	return int(d.stream.Info.NSamples)
}

func (d *decoder) Position() int {
	return int(d.frame.SampleNumber()) + d.posInFrame
}

func (d *decoder) Seek(p int) error {
	if !d.seekEnabled {
		return errors.New("flac.decoder.Seek: not enabled")
	}

	// d.stream.Seek() doesn't seek to the exact position p, instead
	// it seeks to the start of the frame p is in. The frame position
	// is returned and stored in pos.
	pos, err := d.stream.Seek(uint64(p))
	if err != nil {
		return errors.Wrap(err, "flac")
	}
	d.posInFrame = p - int(pos)

	d.frame, err = d.stream.ParseNext()
	if err != nil {
		return errors.Wrap(err, "flac")
	}

	return err
}

func (d *decoder) Close() error {
	if closer, ok := d.r.(io.Closer); ok {
		err := closer.Close()
		if err != nil {
			return errors.Wrap(err, "flac")
		}
	}
	return nil
}
