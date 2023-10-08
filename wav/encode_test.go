package wav

import (
	"github.com/faiface/beep"
	"github.com/orcaman/writerseeker"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncode(t *testing.T) {
	var f = beep.Format{
		SampleRate:  44100,
		NumChannels: 2,
		Precision:   2,
	}
	var w writerseeker.WriterSeeker
	var s = beep.Silence(5)

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
