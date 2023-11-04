package mp3_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/internal/testtools"
	"github.com/gopxl/beep/mp3"
)

func TestDecoder_ReturnBehaviour(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_22050_samples.mp3"))
	assert.NoError(t, err)
	defer f.Close()

	s, _, err := mp3.Decode(f)
	assert.NoError(t, err)
	//assert.Equal(t, 22050, s.Len()) // todo: mp3 seems to return more samples than there are in the file. Uncomment this when fixed.

	testtools.AssertStreamerHasCorrectReturnBehaviour(t, s, s.Len())
}
