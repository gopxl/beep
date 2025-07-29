package mp3_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/v2/internal/testtools"
	"github.com/gopxl/beep/v2/mp3"
)

func TestDecoder_ReturnBehaviour(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_x_padded_samples.mp3"))
	assert.NoError(t, err)
	defer f.Close()

	s, _, err := mp3.Decode(f)
	assert.NoError(t, err)
	// The length of the streamer isn't tested because mp3 files have
	// a different padding depending on the decoder used.
	// https://superuser.com/a/1393775

	testtools.AssertStreamerHasCorrectReturnBehaviour(t, s, s.Len())
}

func TestMP3DecodePanicCase(t *testing.T) {
	r := io.NopCloser(bytes.NewReader([]byte("\xff\xf2000000000000000001\xb3000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")))

	streamer, _, err := mp3.Decode(r)
	if err != nil {
		t.Logf("Expected error caught: %v", err)
		return
	}
	defer streamer.Close()
}
