package beep_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/internal/testtools"
)

func TestMixer_MixesSamples(t *testing.T) {
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

func TestMixer_DrainedStreamersAreRemoved(t *testing.T) {
	s1, _ := testtools.RandomDataStreamer(50)
	s2, _ := testtools.RandomDataStreamer(65)

	m := beep.Mixer{}
	m.Add(s1)
	m.Add(s2)

	// Almost drain s1
	samples := testtools.CollectNum(50, &m)
	assert.Len(t, samples, 50)
	assert.Equal(t, 2, m.Len())

	// Drain s1 (s1 returns !ok, n == 0)
	samples = testtools.CollectNum(10, &m)
	assert.Len(t, samples, 10)
	assert.Equal(t, 1, m.Len())

	// Drain s2 (s2 returns ok, n < len(samples))
	samples = testtools.CollectNum(10, &m)
	assert.Len(t, samples, 10)
	assert.Equal(t, 0, m.Len())
}

func TestMixer_PlaysSilenceWhenNoStreamersProduceSamples(t *testing.T) {
	m := beep.Mixer{}

	// Test silence before streamers are added.
	samples := testtools.CollectNum(10, &m)
	assert.Len(t, samples, 10)
	assert.Equal(t, make([][2]float64, 10), samples)

	// Test silence after streamer has only streamed part of the requested samples.
	s, _ := testtools.RandomDataStreamer(50)
	m.Add(s)

	samples = testtools.CollectNum(100, &m)
	assert.Len(t, samples, 100)
	assert.Equal(t, 0, m.Len())
	assert.Equal(t, make([][2]float64, 50), samples[50:])

	// Test silence after streamers have been drained & removed.
	samples = testtools.CollectNum(10, &m)
	assert.Len(t, samples, 10)
	assert.Equal(t, make([][2]float64, 10), samples)
}

func BenchmarkMixer_MultipleStreams(b *testing.B) {
	s1, _ := testtools.RandomDataStreamer(b.N)
	s2, _ := testtools.RandomDataStreamer(b.N)

	m := beep.Mixer{}
	m.Add(s1)
	m.Add(s2)

	b.StartTimer()

	testtools.CollectNum(b.N, &m)
}

func BenchmarkMixer_OneStream(b *testing.B) {
	s, _ := testtools.RandomDataStreamer(b.N)

	m := beep.Mixer{}
	m.Add(s)

	b.StartTimer()
	testtools.CollectNum(b.N, &m)
}

func BenchmarkMixer_Silence(b *testing.B) {
	m := beep.Mixer{}
	// Don't add any streamers

	b.StartTimer()
	testtools.CollectNum(b.N, &m)
}
