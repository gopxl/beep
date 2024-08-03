package flac_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/internal/testtools"
)

func TestDecoder_ReturnBehaviour(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_22050_samples.flac"))
	assert.NoError(t, err)
	defer f.Close()

	s, _, err := flac.Decode(f)
	assert.NoError(t, err)
	assert.Equal(t, 22050, s.Len())

	testtools.AssertStreamerHasCorrectReturnBehaviour(t, s, s.Len())
}
