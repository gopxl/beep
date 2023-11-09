package vorbis_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/internal/testtools"
	"github.com/gopxl/beep/vorbis"
)

func TestDecoder_ReturnBehaviour(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_22050_samples.ogg"))
	assert.NoError(t, err)
	defer f.Close()

	s, _, err := vorbis.Decode(f)
	assert.NoError(t, err)
	assert.Equal(t, 22050, s.Len())

	testtools.AssertStreamerHasCorrectReturnBehaviour(t, s, s.Len())
}
