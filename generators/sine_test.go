package generators_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/generators"
	"github.com/gopxl/beep/internal/testtools"
)

func TestSineTone(t *testing.T) {
	epsilon := 0.000001

	s, err := generators.SineTone(beep.SampleRate(8000), 400)
	assert.NoError(t, err)

	// Get a full single phase including the last sample.
	phaseLength := 8000 / 400
	samples := testtools.CollectNum(phaseLength+1, s)

	// The sine wave should be 0 at the start, half a phase and at the end of the phase.
	assert.InDelta(t, 0, samples[phaseLength*0][0], epsilon)
	assert.InDelta(t, 0, samples[phaseLength*0][1], epsilon)
	assert.InDelta(t, 0, samples[phaseLength*1/2][0], epsilon)
	assert.InDelta(t, 0, samples[phaseLength*1/2][1], epsilon)
	assert.InDelta(t, 0, samples[phaseLength*1][0], epsilon)
	assert.InDelta(t, 0, samples[phaseLength*1][1], epsilon)

	// The sine wave should be in a peak and trough at 1/4th and 3/4th in the phase respectively.
	assert.InDelta(t, 1, samples[phaseLength*1/4][0], epsilon)
	assert.InDelta(t, 1, samples[phaseLength*1/4][1], epsilon)
	assert.InDelta(t, -1, samples[phaseLength*3/4][0], epsilon)
	assert.InDelta(t, -1, samples[phaseLength*3/4][1], epsilon)
}
