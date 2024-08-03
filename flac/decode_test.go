package flac_test

import (
	"io"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/internal/testtools"
	"github.com/gopxl/beep/v2/wav"

	mewkiz_flac "github.com/mewkiz/flac"
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

func TestDecoder_Seek(t *testing.T) {
	flacFile, err := os.Open(testtools.TestFilePath("valid_44100hz_22050_samples.flac"))
	assert.NoError(t, err)
	defer flacFile.Close()

	// Use WAV file as reference
	wavFile, err := os.Open(testtools.TestFilePath("valid_44100hz_22050_samples.wav"))
	assert.NoError(t, err)
	defer wavFile.Close()

	// Get the frame numbers from the FLAC files manually, so that we
	// can explicitly test difficult Seek positions.
	frameStarts, err := getFlacFrameStartPositions(flacFile)
	assert.NoError(t, err)
	_, err = flacFile.Seek(0, io.SeekStart)
	assert.NoError(t, err)

	flacStream, _, err := flac.Decode(flacFile)
	assert.NoError(t, err)

	wavStream, _, err := wav.Decode(wavFile)
	assert.NoError(t, err)

	assert.Equal(t, wavStream.Len(), flacStream.Len())

	// Test start of 2nd frame
	seekPos := int(frameStarts[1])
	err = wavStream.Seek(seekPos)
	assert.NoError(t, err)
	assert.Equal(t, seekPos, wavStream.Position())
	err = flacStream.Seek(seekPos)
	assert.NoError(t, err)
	assert.Equal(t, seekPos, flacStream.Position())

	wavSamples := testtools.CollectNum(100, wavStream)
	flacSamples := testtools.CollectNum(100, flacStream)
	assert.Equal(t, wavSamples, flacSamples)

	// Test middle of 2nd frame
	seekPos = (int(frameStarts[1]) + int(frameStarts[2])) / 2
	err = wavStream.Seek(seekPos)
	assert.NoError(t, err)
	assert.Equal(t, seekPos, wavStream.Position())
	err = flacStream.Seek(seekPos)
	assert.NoError(t, err)
	assert.Equal(t, seekPos, flacStream.Position())

	wavSamples = testtools.CollectNum(100, wavStream)
	flacSamples = testtools.CollectNum(100, flacStream)
	assert.Equal(t, wavSamples, flacSamples)

	// Test end of 2nd frame
	seekPos = int(frameStarts[2]) - 1
	err = wavStream.Seek(seekPos)
	assert.NoError(t, err)
	assert.Equal(t, seekPos, wavStream.Position())
	err = flacStream.Seek(seekPos)
	assert.NoError(t, err)
	assert.Equal(t, seekPos, flacStream.Position())

	wavSamples = testtools.CollectNum(100, wavStream)
	flacSamples = testtools.CollectNum(100, flacStream)
	assert.Equal(t, wavSamples, flacSamples)
}

func getFlacFrameStartPositions(r io.Reader) ([]uint64, error) {
	stream, err := mewkiz_flac.New(r)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	var frameStarts []uint64
	for {
		frame, err := stream.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		frameStarts = append(frameStarts, frame.SampleNumber())
	}

	return frameStarts, nil
}
