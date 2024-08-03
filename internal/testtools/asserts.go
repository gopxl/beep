package testtools

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/v2"
)

// AssertStreamerHasCorrectReturnBehaviour tests whether the return values returned
// by the streamer s adhere to the description on the Streamer interface.
func AssertStreamerHasCorrectReturnBehaviour(t *testing.T, s beep.Streamer, expectedSamples int) {
	const leaveUnreadInFirstCase = 50

	if expectedSamples < leaveUnreadInFirstCase+1 {
		panic(fmt.Sprintf("AssertStreamerHasCorrectReturnBehaviour must be called with at least %d samples.", leaveUnreadInFirstCase+1))
	}

	// 1. n == len(samples) && ok
	buf := make([][2]float64, 512)
	samplesLeft := expectedSamples - leaveUnreadInFirstCase
	for samplesLeft > 0 {
		toRead := min(samplesLeft, len(buf))
		n, ok := s.Stream(buf[:toRead])
		if !ok {
			t.Fatalf("streamer returned !ok before it was expected to be drained")
		}
		if n < toRead {
			t.Fatalf("streamer didn't return all requested samples before it was expected to be drained")
		}
		if s.Err() != nil {
			t.Fatalf("unexpected error in streamer: %v", s.Err())
		}
		samplesLeft -= n
	}

	// 2. 0 < n && n < len(samples) && ok
	n, ok := s.Stream(buf)
	assert.True(t, ok)
	assert.Equal(t, leaveUnreadInFirstCase, n)
	assert.NoError(t, s.Err())

	// 3. n == 0 && !ok
	n, ok = s.Stream(buf)
	assert.False(t, ok)
	assert.Equal(t, 0, n)
	assert.NoError(t, s.Err())

	// Repeat calls after case 3 must return the same result.
	n, ok = s.Stream(buf)
	assert.False(t, ok)
	assert.Equal(t, 0, n)
	assert.NoError(t, s.Err())
}
