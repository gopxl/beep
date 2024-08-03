package beep_test

import (
	"math/rand"
	"reflect"
	"testing"

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

	if !reflect.DeepEqual(want, got) {
		t.Error("Mix not working correctly")
	}
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
