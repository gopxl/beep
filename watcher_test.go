package beep_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/internal/testtools"
	"github.com/gopxl/beep/speaker"
)

func TestWatcher_Stream_StreamsWithoutEvents(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	l := beep.Watch(s)

	data := testtools.Collect(l)

	assert.Equal(t, 100, len(data))
}

func TestWatcher_Stream_ReturnsCorrectSampleCount(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(75)

	l := beep.Watch(s)
	l.AtSync(25, func(pos int) {})

	var buf [50][2]float64
	n, ok := l.Stream(buf[:])
	assert.True(t, ok)
	assert.Equal(t, 50, n)

	n, ok = l.Stream(buf[:])
	assert.True(t, ok)
	assert.Equal(t, 25, n)
}

func TestWatcher_AtSync(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	numCalled := 0

	l := beep.Watch(s)
	l.AtSync(0, func(pos int) {
		numCalled++
		assert.Equal(t, 0, pos)
		assert.Equal(t, 0, s.Position())
		assert.Equal(t, 0, l.Position())
	})
	l.AtSync(50, func(pos int) {
		numCalled++
		assert.Equal(t, 50, pos)
		assert.Equal(t, 50, s.Position())
		assert.Equal(t, 50, l.Position())
	})
	l.AtSync(100, func(pos int) {
		numCalled++
		assert.Equal(t, 100, pos)
		assert.Equal(t, 100, s.Position())
		assert.Equal(t, 100, l.Position())
	})
	l.AtSync(101, func(pos int) {
		assert.FailNow(t, "event after end of streamer was triggered unexpectedly")
	})

	testtools.Collect(l)

	assert.Equal(t, 3, numCalled)
}

func TestWatcher_AtAsync(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	numCalled := 0

	l := beep.Watch(s)
	l.AtAsync(0, func(pos int) {
		numCalled++
		assert.Equal(t, 0, pos)
		assert.Equal(t, 100, s.Position())
		assert.Equal(t, 100, l.Position())
	})
	l.AtAsync(50, func(pos int) {
		numCalled++
		assert.Equal(t, 50, pos)
		assert.Equal(t, 100, s.Position())
		assert.Equal(t, 100, l.Position())
	})
	l.AtAsync(100, func(pos int) {
		numCalled++
		assert.Equal(t, 100, pos)
		assert.Equal(t, 100, s.Position())
		assert.Equal(t, 100, l.Position())
	})
	l.AtAsync(101, func(pos int) {
		assert.FailNow(t, "event after end of streamer was triggered unexpectedly")
	})

	testtools.Collect(l)

	// Wait for goroutines to finish. Increase the time if test is flaky.
	time.Sleep(time.Millisecond)

	assert.Equal(t, 3, numCalled)
}

func TestWatcher_AtAsync_DoesntDeadlockWithSpeaker(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	numCalled := 0

	l := beep.Watch(s)
	l.AtAsync(50, func(pos int) {
		speaker.Lock()
		numCalled++
		speaker.Unlock()
	})

	// Emulate speaker behaviour by locking the speaker while consuming samples.
	speaker.Lock()
	testtools.Collect(l)
	speaker.Unlock()

	// Wait for goroutines to finish. Increase the time if test is flaky.
	time.Sleep(time.Millisecond)

	assert.Equal(t, 1, numCalled)
}

func TestWatcher_EndedSync(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	numCalled := 0

	l := beep.Watch(s)
	l.EndedSync(func(pos int) {
		numCalled++
		assert.Equal(t, 100, pos)
		assert.Equal(t, 100, s.Position())
		assert.Equal(t, 100, l.Position())
	})

	testtools.Collect(l)

	assert.Equal(t, 1, numCalled)
}

func TestWatcher_EndedAsync(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	numCalled := 0

	l := beep.Watch(s)
	l.EndedAsync(func(pos int) {
		numCalled++
		assert.Equal(t, 100, pos)
		assert.Equal(t, 100, s.Position())
		assert.Equal(t, 100, l.Position())
	})

	testtools.Collect(l)

	// Wait for goroutines to finish. Increase the time if test is flaky.
	time.Sleep(time.Millisecond)

	assert.Equal(t, 1, numCalled)
}

func TestWatcher_EndedAsync_DoesntDeadlockWithSpeaker(t *testing.T) {
	s, _ := testtools.RandomDataStreamer(100)

	numCalled := 0

	l := beep.Watch(s)
	l.EndedAsync(func(pos int) {
		speaker.Lock()
		numCalled++
		speaker.Unlock()
	})

	// Emulate speaker behaviour by locking the speaker while consuming samples.
	speaker.Lock()
	testtools.Collect(l)
	speaker.Unlock()

	// Wait for goroutines to finish. Increase the time if test is flaky.
	time.Sleep(time.Millisecond)

	assert.Equal(t, 1, numCalled)
}
