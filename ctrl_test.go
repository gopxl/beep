package beep_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/internal/testtools"
)

func TestCtrl_CanBePausedAndUnpaused(t *testing.T) {
	s, data := testtools.RandomDataStreamer(20)

	ctrl := beep.Ctrl{
		Streamer: s,
		Paused:   false,
	}

	got := testtools.CollectNum(10, &ctrl)
	assert.Equal(t, data[:10], got)

	ctrl.Paused = true
	got = testtools.CollectNum(10, &ctrl)
	assert.Equal(t, make([][2]float64, 10), got)

	ctrl.Paused = false
	got = testtools.CollectNum(10, &ctrl)
	assert.Equal(t, data[10:20], got)
}

func TestCtrl_DoesNotStreamFromNilStreamer(t *testing.T) {
	ctrl := beep.Ctrl{
		Streamer: nil,
		Paused:   false,
	}

	buf := make([][2]float64, 10)
	n, ok := ctrl.Stream(buf)
	assert.Equal(t, 0, n)
	assert.False(t, ok)
}

func TestCtrl_PropagatesErrors(t *testing.T) {
	ctrl := beep.Ctrl{}

	assert.NoError(t, ctrl.Err())

	err := errors.New("oh no")
	ctrl.Streamer = testtools.ErrorStreamer{Error: err}
	assert.Equal(t, err, ctrl.Err())
}
