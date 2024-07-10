package beep_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/generators"
)

type bufferFormatTestCase struct {
	Name           string
	Precision      int
	NumChannels    int
	Signed         bool
	Bytes          []byte
	Samples        [2]float64
	SkipDecodeTest bool
}

var bufferFormatTests = []bufferFormatTestCase{
	// See https://gist.github.com/endolith/e8597a58bcd11a6462f33fa8eb75c43d
	// for an explanation about the asymmetry in sample encodings in WAV when
	// converting between ints and floats. Note that Beep does not follow the
	// suggested solution. Instead, integer samples are divided by 1 more, so
	// that the resulting float value falls within the range of -1.0 and 1.0.
	// This is similar to how some other tools do the conversion.
	{
		Name:        "1 channel 8bit WAV negative full scale",
		Precision:   1,
		NumChannels: 1,
		Signed:      false,
		Bytes:       []byte{0x00},
		Samples:     [2]float64{-1.0, -1.0},
	},
	{
		Name:        "1 channel 8bit WAV midpoint",
		Precision:   1,
		NumChannels: 1,
		Signed:      false,
		Bytes:       []byte{0x80},
		Samples:     [2]float64{0.0, 0.0},
	},
	{
		// Because the WAV integer range is asymmetric, converting it to float
		// by division will not result in an exactly 1.0 full scale float value.
		// It will be 1 least significant bit integer value lower. "1", converted
		// to float for an 8-bit WAV sample is 1 / (1 << 7).
		Name:        "1 channel 8bit WAV positive full scale minus 1 significant bit",
		Precision:   1,
		NumChannels: 1,
		Signed:      false,
		Bytes:       []byte{0xFF},
		Samples:     [2]float64{1.0 - (1.0 / (1 << 7)), 1.0 - (1.0 / (1 << 7))},
	},
	{
		Name:        "2 channel 8bit WAV full scale",
		Precision:   1,
		NumChannels: 2,
		Signed:      false,
		Bytes:       []byte{0x00, 0xFF},
		Samples:     [2]float64{-1.0, 1.0 - (1.0 / (1 << 7))},
	},
	{
		Name:        "1 channel 16bit WAV negative full scale",
		Precision:   2,
		NumChannels: 1,
		Signed:      true,
		Bytes:       []byte{0x00, 0x80},
		Samples:     [2]float64{-1.0, -1.0},
	},
	{
		Name:        "1 channel 16bit WAV midpoint",
		Precision:   2,
		NumChannels: 1,
		Signed:      true,
		Bytes:       []byte{0x00, 0x00},
		Samples:     [2]float64{0.0, 0.0},
	},
	{
		// Because the WAV integer range is asymmetric, converting it to float
		// by division will not result in an exactly 1.0 full scale float value.
		// It will be 1 least significant bit integer value lower. "1", converted
		// to float for an 16-bit WAV sample is 1 / (1 << 15).
		Name:        "1 channel 16bit WAV positive full scale minus 1 significant bit",
		Precision:   2,
		NumChannels: 1,
		Signed:      true,
		Bytes:       []byte{0xFF, 0x7F},
		Samples:     [2]float64{1.0 - (1.0 / (1 << 15)), 1.0 - (1.0 / (1 << 15))},
	},
	{
		Name:           "1 channel 8bit WAV float positive full scale clipping test",
		Precision:      1,
		NumChannels:    1,
		Signed:         false,
		Bytes:          []byte{0xFF},
		Samples:        [2]float64{1.0, 1.0},
		SkipDecodeTest: true,
	},
	{
		Name:           "1 channel 16bit WAV float positive full scale clipping test",
		Precision:      2,
		NumChannels:    1,
		Signed:         true,
		Bytes:          []byte{0xFF, 0x7F},
		Samples:        [2]float64{1.0, 1.0},
		SkipDecodeTest: true,
	},
}

func TestFormatDecode(t *testing.T) {
	for _, test := range bufferFormatTests {
		if test.SkipDecodeTest {
			continue
		}

		t.Run(test.Name, func(t *testing.T) {
			format := beep.Format{
				SampleRate:  44100,
				Precision:   test.Precision,
				NumChannels: test.NumChannels,
			}

			var sample [2]float64
			var n int
			if test.Signed {
				sample, n = format.DecodeSigned(test.Bytes)
			} else {
				sample, n = format.DecodeUnsigned(test.Bytes)
			}
			assert.Equal(t, len(test.Bytes), n)
			assert.Equal(t, test.Samples, sample)
		})
	}
}

func TestFormatEncode(t *testing.T) {
	for _, test := range bufferFormatTests {
		t.Run(test.Name, func(t *testing.T) {
			format := beep.Format{
				SampleRate:  44100,
				Precision:   test.Precision,
				NumChannels: test.NumChannels,
			}

			bytes := make([]byte, test.Precision*test.NumChannels)
			var n int
			if test.Signed {
				n = format.EncodeSigned(bytes, test.Samples)
			} else {
				n = format.EncodeUnsigned(bytes, test.Samples)
			}
			assert.Equal(t, len(test.Bytes), n)
			assert.Equal(t, test.Bytes, bytes)
		})
	}
}

func TestFormatEncodeDecode(t *testing.T) {
	formats := make(chan beep.Format)
	go func() {
		defer close(formats)
		for _, sampleRate := range []beep.SampleRate{100, 2347, 44100, 48000} {
			for _, numChannels := range []int{1, 2, 3, 4} {
				for _, precision := range []int{1, 2, 3, 4, 5, 6} {
					formats <- beep.Format{
						SampleRate:  sampleRate,
						NumChannels: numChannels,
						Precision:   precision,
					}
				}
			}
		}
	}()

	for format := range formats {
		for i := 0; i < 20; i++ {
			deviation := 2.0 / (math.Pow(2, float64(format.Precision)*8) - 2)
			sample := [2]float64{rand.Float64()*2 - 1, rand.Float64()*2 - 1}

			tmp := make([]byte, format.Width())
			format.EncodeSigned(tmp, sample)
			decoded, _ := format.DecodeSigned(tmp)

			if format.NumChannels == 1 {
				if math.Abs((sample[0]+sample[1])/2-decoded[0]) > deviation || decoded[0] != decoded[1] {
					t.Fatalf("signed decoded sample is too different: %v -> %v (deviation: %v)", sample, decoded, deviation)
				}
			} else {
				if math.Abs(sample[0]-decoded[0]) > deviation || math.Abs(sample[1]-decoded[1]) > deviation {
					t.Fatalf("signed decoded sample is too different: %v -> %v (deviation: %v)", sample, decoded, deviation)
				}
			}

			format.EncodeUnsigned(tmp, sample)
			decoded, _ = format.DecodeUnsigned(tmp)

			if format.NumChannels == 1 {
				if math.Abs((sample[0]+sample[1])/2-decoded[0]) > deviation || decoded[0] != decoded[1] {
					t.Fatalf("unsigned decoded sample is too different: %v -> %v (deviation: %v)", sample, decoded, deviation)
				}
			} else {
				if math.Abs(sample[0]-decoded[0]) > deviation || math.Abs(sample[1]-decoded[1]) > deviation {
					t.Fatalf("unsigned decoded sample is too different: %v -> %v (deviation: %v)", sample, decoded, deviation)
				}
			}
		}
	}
}

func TestBufferAppendPop(t *testing.T) {
	formats := make(chan beep.Format)
	go func() {
		defer close(formats)
		for _, numChannels := range []int{1, 2, 3, 4} {
			formats <- beep.Format{
				SampleRate:  44100,
				NumChannels: numChannels,
				Precision:   2,
			}
		}
	}()

	for format := range formats {
		b := beep.NewBuffer(format)
		b.Append(generators.Silence(768))
		if b.Len() != 768 {
			t.Fatalf("buffer length isn't equal to appended stream length: expected: %v, actual: %v (NumChannels: %v)", 768, b.Len(), format.NumChannels)
		}
		b.Pop(512)
		if b.Len() != 768-512 {
			t.Fatalf("buffer length isn't as expected after Pop: expected: %v, actual: %v (NumChannels: %v)", 768-512, b.Len(), format.NumChannels)
		}
	}
}
