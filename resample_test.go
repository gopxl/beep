package beep_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/internal/testtools"
)

func TestResample(t *testing.T) {
	for _, numSamples := range []int{8, 100, 500, 1000, 50000} {
		for _, old := range []beep.SampleRate{100, 2000, 44100, 48000} {
			for _, new := range []beep.SampleRate{100, 2000, 44100, 48000} {
				if numSamples/int(old)*int(new) > 1e6 {
					continue // skip too expensive combinations
				}

				t.Run(fmt.Sprintf("numSamples_%d_old_%d_new_%d", numSamples, old, new), func(t *testing.T) {
					s, data := testtools.RandomDataStreamer(numSamples)

					want := resampleCorrect(3, old, new, data)

					got := testtools.Collect(beep.Resample(3, old, new, s))

					if !reflect.DeepEqual(want, got) {
						t.Fatal("Resample not working correctly")
					}
				})
			}
		}
	}
}

func resampleCorrect(quality int, old, new beep.SampleRate, p [][2]float64) [][2]float64 {
	ratio := float64(old) / float64(new)
	pts := make([]point, quality*2)
	var resampled [][2]float64
	for i := 0; ; i++ {
		j := float64(i) * ratio
		if int(j) >= len(p) {
			break
		}
		var sample [2]float64
		for c := range sample {
			for k := range pts {
				l := int(j) + k - (len(pts)-1)/2
				if l >= 0 && l < len(p) {
					pts[k] = point{X: float64(l), Y: p[l][c]}
				} else {
					pts[k] = point{X: float64(l), Y: 0}
				}
			}

			startK := 0
			for k, pt := range pts {
				if pt.X >= 0 {
					startK = k
					break
				}
			}
			endK := 0
			for k, pt := range pts {
				if pt.X < float64(len(p)) {
					endK = k + 1
				}
			}

			y := lagrange(pts[startK:endK], j)
			sample[c] = y
		}
		resampled = append(resampled, sample)
	}
	return resampled
}

func lagrange(pts []point, x float64) (y float64) {
	y = 0.0
	for j := range pts {
		l := 1.0
		for m := range pts {
			if j == m {
				continue
			}
			l *= (x - pts[m].X) / (pts[j].X - pts[m].X)
		}
		y += pts[j].Y * l
	}
	return y
}

type point struct {
	X, Y float64
}

func FuzzResampler_SetRatio(f *testing.F) {
	f.Add(44100, 48000, 0.5, 1.0, 8.0)
	f.Fuzz(func(t *testing.T, original, desired int, r1, r2, r3 float64) {
		if original <= 0 || desired <= 0 || r1 <= 0 || r2 <= 0 || r3 <= 0 {
			t.Skip()
		}

		s, _ := testtools.RandomDataStreamer(1e4)
		r := beep.Resample(4, beep.SampleRate(original), beep.SampleRate(desired), s)
		testtools.CollectNum(1024, r)
		r.SetRatio(r1)
		testtools.CollectNum(1024, r)
		r.SetRatio(r2)
		testtools.CollectNum(1024, r)
		r.SetRatio(r3)
		testtools.CollectNum(1024, r)
	})
}
