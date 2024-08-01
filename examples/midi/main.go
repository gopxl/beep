package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/midi"
	"github.com/gopxl/beep/speaker"
)

func main() {
	var sampleRate beep.SampleRate = 44100

	err := speaker.Init(sampleRate, sampleRate.N(time.Second/30))
	if err != nil {
		log.Fatal(err)
	}

	// Load a soundfont.
	soundFontFile, err := os.Open("Florestan-Basic-GM-GS-by-Nando-Florestan(Public-Domain).sf2")
	if err != nil {
		log.Fatal(err)
	}
	defer soundFontFile.Close()

	soundFont, err := midi.NewSoundFont(soundFontFile)
	if err != nil {
		log.Fatal(err)
	}

	// Load a midi track.
	midiFile, err := os.Open("Buy to the Beat - V2.mid")
	if err != nil {
		log.Fatal(err)
	}
	defer midiFile.Close()

	s, format, err := midi.Decode(midiFile, soundFont, sampleRate)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Song duration: %v\n", format.SampleRate.D(s.Len()))
	speaker.PlayAndWait(s)
}
