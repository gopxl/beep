package wav

import (
	"bytes"
	"github.com/gopxl/beep"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecode(t *testing.T) {
	wav := []byte{
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
	}

	r := bytes.NewReader(wav)

	s, f, err := Decode(r)
	if err != nil {
		t.Fatalf("failed to decode the WAV file: %v", err)
	}

	assert.Equal(t, beep.Format{
		SampleRate:  44100,
		NumChannels: 2,
		Precision:   2,
	}, f)

	assert.NoError(t, s.Err())
	assert.Equal(t, 5, s.Len())
	assert.Equal(t, 0, s.Position())

	samples := make([][2]float64, 3)
	// Stream first few bytes
	n, ok := s.Stream(samples)
	assert.Equal(t, 3, n)
	assert.Truef(t, ok, "the decoder failed to stream the samples")
	assert.Equal(t, 3, s.Position())
	assert.NoError(t, s.Err())
	// Drain the streamer
	n, ok = s.Stream(samples)
	assert.Equal(t, 2, n)
	assert.Truef(t, ok, "the decoder failed to stream the samples")
	assert.Equal(t, 5, s.Position())
	assert.NoError(t, s.Err())
	// Drain the streamer some more
	n, ok = s.Stream(samples)
	assert.Equal(t, 0, n)
	assert.Equal(t, 5, s.Position())
	assert.Falsef(t, ok, "expected the decoder to return false after it was fully drained")
	assert.NoError(t, s.Err())

	d, ok := s.(*decoder)
	if !ok {
		t.Fatal("Streamer is not a decoder")
	}

	assert.Equal(t, header{
		RiffMark:      [4]byte{'R', 'I', 'F', 'F'},
		FileSize:      56, // without the riff mark and file size
		WaveMark:      [4]byte{'W', 'A', 'V', 'E'},
		FmtMark:       [4]byte{'f', 'm', 't', ' '},
		FormatSize:    16,
		FormatType:    1, // 1 = PCM
		NumChans:      2,
		SampleRate:    44100,
		ByteRate:      176400, // 44100 * 2 bytes/sample precision * 2 channels = 176400 bytes/sec
		BytesPerFrame: 4,      // 2 bytes/sample precision * 2 channels = 4 bytes/frame
		BitsPerSample: 16,     // 2 bytes/sample precision = 16 bits/sample
		DataMark:      [4]byte{'d', 'a', 't', 'a'},
		DataSize:      20, // 5 samples * 2 bytes/sample precision * 2 channels = 20 bytes
	}, d.h)
}
