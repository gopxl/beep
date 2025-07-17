package flac

import (
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
	d.hasFixedBlockSize = d.frame.HasFixedBlockSize

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

	hasFixedBlockSize bool
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

		//toFill := min(samplesLeft, len(samples))
		toFill := samplesLeft
		if toFill > len(samples) {
			toFill = len(samples)
		}
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

	if numChannels == 1 {
		samples1 := frame.Subframes[0].Samples[start:]
		for i := 0; i < num; i++ {
			v := float64(samples1[i]) * q
			into[i][0] = v
			into[i][1] = v
		}
	} else {
		samples1 := frame.Subframes[0].Samples[start:]
		samples2 := frame.Subframes[1].Samples[start:]
		for i := 0; i < num; i++ {
			into[i][0] = float64(samples1[i]) * q
			into[i][1] = float64(samples2[i]) * q
		}
	}
}

func (d *decoder) Err() error {
	return d.err
}

func (d *decoder) Len() int {
	return int(d.stream.Info.NSamples)
}

func (d *decoder) Position() int {
	if d.frame == nil {
		return d.Len()
	}

	// Temporary workaround until https://github.com/mewkiz/flac/pull/73 is resolved.
	if d.hasFixedBlockSize {
		return int(d.frame.Num)*int(d.stream.Info.BlockSizeMax) + d.posInFrame
	}

	return int(d.frame.SampleNumber()) + d.posInFrame
}

func (d *decoder) Seek(p int) error {
	if !d.seekEnabled {
		return errors.New("flac.decoder.Seek: not enabled")
	}

	// Temporary workaround until https://github.com/mewkiz/flac/pull/73 is resolved.
	// frame.SampleNumber() doesn't work for the last frame of a fixed block size stream
	// with the result that seeking to that frame doesn't work either. Therefore, if such
	// a seek is requested, we seek to one of the frames before it and consume until the
	// desired position is reached.
	if d.hasFixedBlockSize {
		lastFrameStartLowerBound := d.Len() - int(d.stream.Info.BlockSizeMax)
		if p >= lastFrameStartLowerBound {
			// Seek to & consume an earlier frame.
			_, err := d.stream.Seek(uint64(lastFrameStartLowerBound - 1))
			if err != nil {
				return errors.Wrap(err, "flac")
			}
			for {
				d.frame, err = d.stream.ParseNext()
				if err != nil {
					return errors.Wrap(err, "flac")
				}
				// Calculate the frame start position manually, because this doesn't
				// work for the last frame.
				frameStart := d.frame.Num * uint64(d.stream.Info.BlockSizeMax)
				if frameStart+uint64(d.frame.BlockSize) >= d.stream.Info.NSamples {
					// Found the desired frame.
					d.posInFrame = p - int(frameStart)
					return nil
				}
			}
		}
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

	return nil
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
