package beep_test

import (
	"testing"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/internal/testtools"
)

func TestMixer(t *testing.T) {
	epsilon := 0.000001

	s1, data1 := testtools.RandomDataStreamer(200)
	s2, data2 := testtools.RandomDataStreamer(200)

	m := beep.Mixer{}
	m.Add(s1)
	m.Add(s2)

	samples := testtools.CollectNum(100, &m)
	for i, s := range samples {
		wantL := data1[i][0] + data2[i][0]
		wantR := data1[i][1] + data2[i][1]

		if s[0] < wantL-epsilon || s[0] > wantL+epsilon {
			t.Fatalf("unexpected value for mixed samples; expected: %f, got: %f", wantL, s[0])
		}
		if s[1] < wantR-epsilon || s[1] > wantR+epsilon {
			t.Fatalf("unexpected value for mixed samples; expected: %f, got: %f", wantR, s[1])
		}
	}

	s3, data3 := testtools.RandomDataStreamer(100)
	m.Add(s3)

	samples = testtools.CollectNum(100, &m)
	for i, s := range samples {
		wantL := data1[100+i][0] + data2[100+i][0] + data3[i][0]
		wantR := data1[100+i][1] + data2[100+i][1] + data3[i][1]

		if s[0] < wantL-epsilon || s[0] > wantL+epsilon {
			t.Fatalf("unexpected value for mixed samples; expected: %f, got: %f", wantL, s[0])
		}
		if s[1] < wantR-epsilon || s[1] > wantR+epsilon {
			t.Fatalf("unexpected value for mixed samples; expected: %f, got: %f", wantR, s[1])
		}
	}
}

func BenchmarkMixer(b *testing.B) {
	s1, _ := testtools.RandomDataStreamer(b.N)
	s2, _ := testtools.RandomDataStreamer(b.N)

	m := beep.Mixer{}
	m.Add(s1)
	m.Add(s2)

	b.StartTimer()

	testtools.CollectNum(b.N, &m)
}
