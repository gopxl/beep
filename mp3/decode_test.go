package mp3_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/internal/testtools"
	"github.com/gopxl/beep/mp3"
)

func TestDecoder_ReturnBehaviour(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_x_padded_samples.mp3"))
	assert.NoError(t, err)
	defer f.Close()

	s, _, err := mp3.DecodeReadSeeker(f)
	assert.NoError(t, err)
	// The length of the streamer isn't tested because mp3 files have
	// a different padding depending on the decoder used.
	// https://superuser.com/a/1393775

	testtools.AssertStreamerHasCorrectReturnBehaviour(t, s, s.Len())
}
