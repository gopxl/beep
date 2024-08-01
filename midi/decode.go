// Package midi implements audio data decoding in MIDI format.
package midi

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/samhocevar/go-meltysynth/meltysynth"

	"github.com/gopxl/beep"
)

const (
	midiNumChannels = 2
	midiPrecision   = 4
)

// NewSoundFont reads a sound font containing instruments. A sound font is required in order to play MIDI files.
//
// NewSoundFont closes the supplied ReadCloser.
func NewSoundFont(r io.ReadCloser) (*SoundFont, error) {
	sf, err := meltysynth.NewSoundFont(r)
	if err != nil {
		return nil, err
	}
	err = r.Close()
	if err != nil {
		return nil, err
	}
	return &SoundFont{sf}, nil
}

type SoundFont struct {
	sf *meltysynth.SoundFont
}

// Decode takes an io.Reader containing audio data in MIDI format and a SoundFont to synthesize the sounds
// and returns a beep.StreamSeeker, which streams the audio.
//
// The io.Reader can be closed immediately after the call to midi.Decode because the data
// will be loaded into memory.
func Decode(r io.Reader, sf *SoundFont, sampleRate beep.SampleRate) (s beep.StreamSeeker, format beep.Format, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "midi")
		}
	}()

	settings := meltysynth.NewSynthesizerSettings(int32(sampleRate))
	synth, err := meltysynth.NewSynthesizer(sf.sf, settings)
	if err != nil {
		return nil, beep.Format{}, err
	}

	mf, err := meltysynth.NewMidiFile(r)
	if err != nil {
		return nil, beep.Format{}, err
	}

	seq := meltysynth.NewMidiFileSequencer(synth)
	seq.Play(mf /*loop*/, false)

	format = beep.Format{
		SampleRate:  sampleRate,
		NumChannels: midiNumChannels,
		Precision:   midiPrecision,
	}

	return &decoder{
		synth:      synth,
		mf:         mf,
		seq:        seq,
		sampleRate: sampleRate,
		bufLeft:    make([]float32, 512),
		bufRight:   make([]float32, 512),
	}, format, nil
}

type decoder struct {
	synth             *meltysynth.Synthesizer
	mf                *meltysynth.MidiFile
	seq               *meltysynth.MidiFileSequencer
	sampleRate        beep.SampleRate
	bufLeft, bufRight []float32
	err               error
}

func (d *decoder) Stream(samples [][2]float64) (n int, ok bool) {
	if d.err != nil {
		return 0, false
	}

	samplesLeft := d.Len() - d.Position()
	if len(samples) > samplesLeft {
		samples = samples[:samplesLeft]
	}

	for len(samples) > 0 {
		cn := len(d.bufLeft)
		if cn > len(samples) {
			cn = len(samples)
		}

		d.seq.Render(d.bufLeft[:cn], d.bufRight[:cn])
		for i := 0; i < cn; i++ {
			samples[i][0] = float64(d.bufLeft[i])
			samples[i][1] = float64(d.bufRight[i])
		}

		samples = samples[cn:]
		n += cn
	}

	return n, n > 0
}

func (d *decoder) Err() error {
	return d.err
}

func (d *decoder) Len() int {
	return d.sampleRate.N(d.mf.GetLength())
}

func (d *decoder) Position() int {
	return d.sampleRate.N(d.seq.Pos())
}

func (d *decoder) Seek(p int) error {
	if p < 0 || d.Len() < p {
		return fmt.Errorf("midi: seek position %v out of range [%v, %v]", p, 0, d.Len())
	}
	d.seq.Seek(d.sampleRate.D(p))
	return nil
}
