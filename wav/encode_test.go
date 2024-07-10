package wav

import (
	"fmt"
	"math"
	"testing"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/generators"
	"github.com/gopxl/beep/internal/testtools"

	"github.com/orcaman/writerseeker"
	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	var f = beep.Format{
		SampleRate:  44100,
		NumChannels: 2,
		Precision:   2,
	}
	var w writerseeker.WriterSeeker
	var s = generators.Silence(5)

	err := Encode(&w, s, f)
	if err != nil {
		t.Fatalf("encoding failed with error: %v", err)
	}

	r := w.BytesReader()
	expectedWrittenSize := 44 /* header length */ + 5*f.Precision*f.NumChannels /* number of samples * bytes per sample * number of channels */
	assert.Equal(t, expectedWrittenSize, r.Len(), "the encoded file doesn't have the right size")

	encoded := make([]byte, r.Len())
	_, err = w.Reader().Read(encoded)
	if err != nil {
		t.Fatalf("failed reading the buffer: %v", err)
	}

	// Everything is encoded using little endian.
	assert.Equal(t, []byte{
		// Riff mark
		'R', 'I', 'F', 'F',
		// File size without riff mark and file size
		0x38, 0x00, 0x00, 0x00, // 56 bytes
		// Wave mark
		'W', 'A', 'V', 'E',

		// Fmt mark
		'f', 'm', 't', ' ',
		// Format chunk size
		0x10, 0x00, 0x00, 0x00, // 16 bytes
		// Format type
		0x01, 0x00, // 1 = PCM
		// Number of channels,
		0x02, 0x00,
		// Sample rate
		0x44, 0xAC, 0x00, 0x00, // 44100 samples/sec
		// Byte rate
		0x10, 0xB1, 0x02, 0x00, // 44100 * 2 bytes/sample precision * 2 channels = 176400 bytes/sec
		// Bytes per frame
		0x04, 0x00, // 2 bytes/sample precision * 2 channels = 4 bytes/frame
		// Bits per sample
		0x10, 0x00, // 2 bytes/sample precision = 16 bits/sample

		// Data mark
		'd', 'a', 't', 'a',
		// Data size
		0x14, 0x00, 0x00, 0x00, // 5 samples * 2 bytes/sample precision * 2 channels = 20 bytes
		// Data
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}, encoded, "the encoded file isn't formatted as expected")
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	numChannelsS := []int{1, 2}
	precisions := []int{1, 2, 3}

	for _, numChannels := range numChannelsS {
		for _, precision := range precisions {
			name := fmt.Sprintf("%d_channel(s)_%d_precision", numChannels, precision)
			t.Run(name, func(t *testing.T) {
				var s beep.Streamer
				s, data := testtools.RandomDataStreamer(1000)

				if numChannels == 1 {
					s = effects.Mono(s)
					for i := range data {
						mix := (data[i][0] + data[i][1]) / 2
						data[i][0] = mix
						data[i][1] = mix
					}
				}

				var w writerseeker.WriterSeeker

				format := beep.Format{SampleRate: 44100, NumChannels: numChannels, Precision: precision}

				err := Encode(&w, s, format)
				assert.NoError(t, err)

				s, decodedFormat, err := Decode(w.Reader())
				assert.NoError(t, err)
				assert.Equal(t, format, decodedFormat)

				actual := testtools.Collect(s)
				assert.Len(t, actual, 1000)

				// Delta is determined as follows:
				// The float values range from -1 to 1, which difference is 2.0.
				// For each byte of precision, there are 8 bits -> 2^(precision*8) different possible values.
				// So, fitting 2^(precision*8) values into a range of 2.0, each "step" must not
				// be bigger than 2.0 / math.Exp2(float64(precision*8)).
				delta := 2.0 / math.Exp2(float64(precision*8))
				for i := range actual {
					// Adjust for clipping.
					if data[i][0] >= 1.0 {
						data[i][0] = 1.0 - 1.0/(math.Exp2(float64(precision)*8-1))
					}
					if data[i][1] >= 1.0 {
						data[i][1] = 1.0 - 1.0/(math.Exp2(float64(precision)*8-1))
					}

					if actual[i][0] <= data[i][0]-delta || actual[i][0] >= data[i][0]+delta {
						t.Fatalf("encoded & decoded sample doesn't match orginal. expected: %v, actual: %v", data[i][0], actual[i][0])
					}
					if actual[i][1] <= data[i][1]-delta || actual[i][1] >= data[i][1]+delta {
						t.Fatalf("encoded & decoded sample doesn't match orginal. expected: %v, actual: %v", data[i][1], actual[i][1])
					}
				}
			})
		}
	}
}
