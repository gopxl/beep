package effects_test

import (
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/generators"
	"github.com/gopxl/beep/speaker"
)

// Cross-fade between two sine tones.
func ExampleTransition() {
	sampleRate := beep.SampleRate(44100)

	s1, err := generators.SineTone(sampleRate, 261.63)
	if err != nil {
		panic(err)
	}
	s2, err := generators.SineTone(sampleRate, 329.628)
	if err != nil {
		panic(err)
	}

	crossFades := beep.Seq(
		// Play s1 normally for 3 seconds
		beep.Take(sampleRate.N(time.Second*3), s1),
		// Play s1 and s2 together. s1 transitions from a gain of 1.0 (normal volume)
		// to 0.0 (silent) whereas s2 does the opposite. The equal power transition
		// function helps keep the overall volume constant.
		beep.Mix(
			effects.Transition(
				beep.Take(sampleRate.N(time.Second*2), s1),
				sampleRate.N(time.Second*2),
				1.0,
				0.0,
				effects.TransitionEqualPower,
			),
			effects.Transition(
				beep.Take(sampleRate.N(time.Second*2), s2),
				sampleRate.N(time.Second*2),
				0.0,
				1.0,
				effects.TransitionEqualPower,
			),
		),
		// Play the rest of s2 normally for 3 seconds
		beep.Take(sampleRate.N(time.Second*3), s2),
	)

	err = speaker.Init(sampleRate, sampleRate.N(time.Second/30))
	if err != nil {
		panic(err)
	}

	done := make(chan struct{})
	speaker.Play(beep.Seq(
		crossFades,
		beep.Callback(func() {
			done <- struct{}{}
		}),
	))
	<-done
}
