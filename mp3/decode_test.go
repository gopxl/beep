package mp3_test

import (
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
	// TestDecoder_TrimsGaplessSamples tests the length of the streamer.

	testtools.AssertStreamerHasCorrectReturnBehaviour(t, s, s.Len())
}

func TestDecoder_TrimsGaplessSamples(t *testing.T) {
	// The file has a LAME tag with an encoder delay of 576 and a padding of 1566.
	// Without them it holds the same 22050 samples as the other test files.
	const want = 22050

	f, err := os.Open(testtools.TestFilePath("valid_44100hz_x_padded_samples.mp3"))
	assert.NoError(t, err)
	s, _, err := mp3.Decode(f)
	assert.NoError(t, err)
	defer s.Close()

	assert.Equal(t, want, s.Len())
	seekable := testtools.Collect(s)
	assert.Len(t, seekable, want)
	assert.Equal(t, want, s.Position())

	// A non-seekable source has no known length, so only the start is trimmed.
	f, err = os.Open(testtools.TestFilePath("valid_44100hz_x_padded_samples.mp3"))
	assert.NoError(t, err)
	s, _, err = mp3.Decode(nonSeekable{f})
	assert.NoError(t, err)
	defer s.Close()

	piped := testtools.Collect(s)
	assert.Len(t, piped, want+1566-529)
	assert.Equal(t, seekable, piped[:want])
}

func TestDecoder_SeekTrimmed(t *testing.T) {
	f, err := os.Open(testtools.TestFilePath("valid_44100hz_x_padded_samples.mp3"))
	assert.NoError(t, err)
	s, _, err := mp3.Decode(f)
	assert.NoError(t, err)
	defer s.Close()

	assert.NoError(t, s.Seek(5000))
	assert.Equal(t, 5000, s.Position())
	assert.Len(t, testtools.Collect(s), s.Len()-5000)

	assert.NoError(t, s.Seek(s.Len()))
	n, ok := s.Stream(make([][2]float64, 10))
	assert.Equal(t, 0, n)
	assert.False(t, ok)
	assert.Error(t, s.Seek(s.Len()+1))
}

// nonSeekable hides the Seek method of an *os.File.
type nonSeekable struct{ f *os.File }

func (n nonSeekable) Read(p []byte) (int, error) { return n.f.Read(p) }
func (n nonSeekable) Close() error               { return n.f.Close() }
