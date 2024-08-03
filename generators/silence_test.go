package generators_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/v2/generators"
	"github.com/gopxl/beep/v2/internal/testtools"
)

func TestSilence_StreamsFiniteSamples(t *testing.T) {
	s := generators.Silence(100)

	got := testtools.CollectNum(200, s)
	assert.Equal(t, make([][2]float64, 100), got)

	got = testtools.CollectNum(200, s)
	assert.Len(t, got, 0)
}

func TestSilence_StreamsInfiniteSamples(t *testing.T) {
	s := generators.Silence(-1)

	got := testtools.CollectNum(200, s)
	assert.Equal(t, make([][2]float64, 200), got)

	got = testtools.CollectNum(200, s)
	assert.Equal(t, make([][2]float64, 200), got)
}
