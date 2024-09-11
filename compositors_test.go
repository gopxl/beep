package beep_test

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/internal/testtools"
)

func TestTake(t *testing.T) {
	for i := 0; i < 7; i++ {
		total := rand.Intn(1e5) + 1e4
		s, data := testtools.RandomDataStreamer(total)
		take := rand.Intn(total)

		want := data[:take]
		got := testtools.Collect(beep.Take(take, s))

		if !reflect.DeepEqual(want, got) {
			t.Error("Take not working correctly")
		}
	}
}

func TestLoop(t *testing.T) {
	for i := 0; i < 7; i++ {
		for n := 0; n < 5; n++ {
			s, data := testtools.RandomDataStreamer(10)

			var want [][2]float64
			for j := 0; j < n; j++ {
				want = append(want, data...)
			}
			got := testtools.Collect(beep.Loop(n, s))

			if !reflect.DeepEqual(want, got) {
				t.Error("Loop not working correctly")
			}
		}
	}
}

func TestLoop2(t *testing.T) {
	// LoopStart is bigger than s.Len()
	s, _ := testtools.NewSequentialDataStreamer(5)
	l, err := beep.Loop2(s, beep.LoopStart(5))
	assert.EqualError(t, err, "invalid argument to Loop2; start position 5 must be smaller than the source streamer length 5")

	// LoopStart is bigger than LoopEnd
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopBetween(4, 4))
	assert.EqualError(t, err, "invalid argument to Loop2; start position 4 must be smaller than the end position 4")

	// Loop indefinitely (no options).
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s)
	assert.NoError(t, err)
	got := testtools.CollectNum(16, l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {0, 0}}, got)

	// Test no loop.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(0))
	assert.NoError(t, err)
	got = testtools.Collect(l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}}, got)

	// Test loop once.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(1))
	assert.NoError(t, err)
	got = testtools.Collect(l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}}, got)

	// Test loop twice.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(2))
	assert.NoError(t, err)
	got = testtools.Collect(l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}}, got)

	// Test loop from start position.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(2), beep.LoopStart(2))
	assert.NoError(t, err)
	got = testtools.Collect(l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {2, 2}, {3, 3}, {4, 4}, {2, 2}, {3, 3}, {4, 4}}, got)

	// Test loop with end position.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(2), beep.LoopEnd(4))
	assert.NoError(t, err)
	got = testtools.Collect(l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}}, got)

	// Test loop with start and end position.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(2), beep.LoopBetween(2, 4))
	assert.NoError(t, err)
	got = testtools.Collect(l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {2, 2}, {3, 3}, {2, 2}, {3, 3}, {4, 4}}, got)

	// Loop indefinitely with both start and end position.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopBetween(2, 4))
	assert.NoError(t, err)
	got = testtools.CollectNum(10, l)
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {2, 2}, {3, 3}, {2, 2}, {3, 3}, {2, 2}, {3, 3}}, got)

	//// Test streaming from the middle of the loops.
	s, _ = testtools.NewSequentialDataStreamer(5)
	l, err = beep.Loop2(s, beep.LoopTimes(2), beep.LoopBetween(2, 4)) // 0, 1, 2, 3, 2, 3, 2, 3
	assert.NoError(t, err)
	// First stream to the middle of a loop.
	buf := make([][2]float64, 3)
	if n, ok := l.Stream(buf); n != 3 || !ok {
		t.Fatalf("want n %d got %d, want ok %t got %t", 3, n, true, ok)
	}
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}}, buf)
	// Then stream starting at the middle of the loop.
	if n, ok := l.Stream(buf); n != 3 || !ok {
		t.Fatalf("want n %d got %d, want ok %t got %t", 3, n, true, ok)
	}
	assert.Equal(t, [][2]float64{{3, 3}, {2, 2}, {3, 3}}, buf)

	// Test error handling in middle of loop.
	expectedErr := errors.New("expected error")
	s, _ = testtools.NewSequentialDataStreamer(5)
	s = testtools.NewDelayedErrorStreamer(s, 5, expectedErr)
	l, err = beep.Loop2(s, beep.LoopTimes(3), beep.LoopBetween(2, 4)) // 0, 1, 2, 3, 2, 3, 2, 3
	assert.NoError(t, err)
	buf = make([][2]float64, 10)
	if n, ok := l.Stream(buf); n != 5 || !ok {
		t.Fatalf("want n %d got %d, want ok %t got %t", 5, n, true, ok)
	}
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {2, 2}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}}, buf)
	assert.Equal(t, expectedErr, l.Err())
	if n, ok := l.Stream(buf); n != 0 || ok {
		t.Fatalf("want n %d got %d, want ok %t got %t", 0, n, false, ok)
	}
	assert.Equal(t, expectedErr, l.Err())

	// Test error handling during call to Seek().
	s, _ = testtools.NewSequentialDataStreamer(5)
	s = testtools.NewSeekErrorStreamer(s, expectedErr)
	l, err = beep.Loop2(s, beep.LoopTimes(3), beep.LoopBetween(2, 4)) // 0, 1, 2, 3, [error]
	assert.NoError(t, err)
	buf = make([][2]float64, 10)
	if n, ok := l.Stream(buf); n != 4 || !ok {
		t.Fatalf("want n %d got %d, want ok %t got %t", 4, n, true, ok)
	}
	assert.Equal(t, [][2]float64{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}}, buf)
	assert.Equal(t, expectedErr, l.Err())
	if n, ok := l.Stream(buf); n != 0 || ok {
		t.Fatalf("want n %d got %d, want ok %t got %t", 0, n, false, ok)
	}
	assert.Equal(t, expectedErr, l.Err())
}

func TestSeq(t *testing.T) {
	var (
		n    = 7
		s    = make([]beep.Streamer, n)
		data = make([][][2]float64, n)
	)
	for i := range s {
		s[i], data[i] = testtools.RandomDataStreamer(rand.Intn(1e5) + 1e4)
	}

	var want [][2]float64
	for _, d := range data {
		want = append(want, d...)
	}

	got := testtools.Collect(beep.Seq(s...))

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Seq not working properly")
	}
}

func TestMix(t *testing.T) {
	var (
		n    = 7
		s    = make([]beep.Streamer, n)
		data = make([][][2]float64, n)
	)
	for i := range s {
		s[i], data[i] = testtools.RandomDataStreamer(rand.Intn(1e5) + 1e4)
	}

	maxLen := 0
	for _, d := range data {
		maxLen = max(maxLen, len(d))
	}

	want := make([][2]float64, maxLen)
	for _, d := range data {
		for i := range d {
			want[i][0] += d[i][0]
			want[i][1] += d[i][1]
		}
	}

	got := testtools.Collect(beep.Mix(s...))

	testtools.AssertSamplesEqual(t, want, got)
}

func TestDup(t *testing.T) {
	for i := 0; i < 7; i++ {
		s, data := testtools.RandomDataStreamer(rand.Intn(1e5) + 1e4)
		st, su := beep.Dup(s)

		var tData, uData [][2]float64
		for {
			buf := make([][2]float64, rand.Intn(1e4))
			tn, tok := st.Stream(buf)
			tData = append(tData, buf[:tn]...)

			buf = make([][2]float64, rand.Intn(1e4))
			un, uok := su.Stream(buf)
			uData = append(uData, buf[:un]...)

			if !tok && !uok {
				break
			}
		}

		if !reflect.DeepEqual(data, tData) || !reflect.DeepEqual(data, uData) {
			t.Error("Dup not working correctly")
		}
	}
}
