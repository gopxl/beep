package flac_test

import (
	"github.com/gopxl/beep/flac"
	"github.com/gopxl/beep/internal/testtools"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDecoder_Stream(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_22050_samples.flac"))
	assert.NoError(t, err)
	defer f.Close()

	streamer, _, err := flac.Decode(f)
	assert.NoError(t, err)

	// Case 1: return ok with all requested samples
	buf := testtools.CollectNum(22000, streamer)
	assert.Lenf(t, buf, 22000, "streamer quit prematurely; expected %d samples, got %d", 22000, len(buf))
	assert.NoError(t, streamer.Err())

	buf = make([][2]float64, 512)

	// Case 2: return ok with 0 < n < 512 samples
	n, ok := streamer.Stream(buf[:])
	assert.True(t, ok)
	assert.Equal(t, 50, n)

	// Case 3: return !ok with n == 0
	n, ok = streamer.Stream(buf[:])
	assert.False(t, ok)
	assert.Equal(t, 0, n)
}
