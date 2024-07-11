// Package midi implements audio data decoding in MIDI format.
package midi

import (
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/samhocevar/go-meltysynth/meltysynth"

	"github.com/gopxl/beep"
)

const (
	midiSampleRate  = 44100
	midiNumChannels = 2
	midiPrecision   = 4
)

// Read a sound font containing instruments. A sound font is required in order to play MIDI files.
func NewSoundFont(r io.Reader) (*SoundFont, error) {
	sf, err := meltysynth.NewSoundFont(r)
	if err != nil {
		return nil, err
	}
	return &SoundFont{sf}, nil
}

type SoundFont struct {
	sf *meltysynth.SoundFont
}

// Decode takes a ReadCloser containing audio data in MIDI format and returns a StreamSeekCloser,
// which streams that audio. The Seek method will panic if rc is not io.Seeker.
//
// Do not close the supplied ReadSeekCloser, instead, use the Close method of the returned
// StreamSeekCloser when you want to release the resources.
func Decode(rc io.ReadCloser, sf *SoundFont) (s beep.StreamSeekCloser, format beep.Format, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "midi")
		}
	}()
	settings := meltysynth.NewSynthesizerSettings(midiSampleRate)
	synth, err := meltysynth.NewSynthesizer(sf.sf, settings)
	if err != nil {
		return nil, beep.Format{}, err
	}
	mf, err := meltysynth.NewMidiFile(rc)
	if err != nil {
		return nil, beep.Format{}, err
	}
	seq := meltysynth.NewMidiFileSequencer(synth)
	seq.Play(mf /*loop*/, false)
	format = beep.Format{
		SampleRate:  beep.SampleRate(midiSampleRate),
		NumChannels: midiNumChannels,
		Precision:   midiPrecision,
	}
	return &decoder{rc, synth, mf, seq, nil}, format, nil
}

type decoder struct {
	closer io.Closer
	synth  *meltysynth.Synthesizer
	mf     *meltysynth.MidiFile
	seq    *meltysynth.MidiFileSequencer
	err    error
}

func (d *decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}
	sampleCount := d.Len() - d.Position()
	if sampleCount > len(samples) {
		sampleCount = len(samples)
	}
	left := make([]float32, sampleCount)
	right := make([]float32, sampleCount)
	d.seq.Render(left, right)
	for i := range left {
		samples[i][0] = float64(left[i])
		samples[i][1] = float64(right[i])
	}
	return sampleCount, sampleCount > 0
}

func (d *decoder) Err() error {
	return d.err
}

func (d *decoder) Len() int {
	return int(d.mf.GetLength().Seconds() * midiSampleRate)
}

func (d *decoder) Position() int {
	return int(d.seq.Pos().Seconds() * midiSampleRate)
}

func (d *decoder) Seek(p int) error {
	if p < 0 || d.Len() < p {
		return fmt.Errorf("midi: seek position %v out of range [%v, %v]", p, 0, d.Len())
	}
	d.seq.Seek(time.Duration(float64(time.Second) * float64(p) / float64(midiSampleRate)))
	return nil
}

func (d *decoder) Close() error {
	err := d.closer.Close()
	if err != nil {
		return errors.Wrap(err, "midi")
	}
	d.seq.Stop()
	return nil
}
