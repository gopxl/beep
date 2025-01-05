package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
)

func main() {
	f, err := os.Open("../Miami_Slice_-_04_-_Step_Into_Me.mp3")
	if err != nil {
		log.Fatal(err)
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	defer streamer.Close()

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	loopStreamer, err := beep.Loop2(streamer, beep.LoopTimes(2))
	if err != nil {
		log.Fatal(err)
	}
	fast := beep.ResampleRatio(4, 5, loopStreamer)

	done := make(chan bool)
	speaker.Play(beep.Seq(fast, beep.Callback(func() {
		done <- true
	})))

	for {
		select {
		case <-done:
			return
		case <-time.After(time.Second):
			speaker.Lock()
			fmt.Println(format.SampleRate.D(streamer.Position()).Round(time.Second))
			speaker.Unlock()
		}
	}
}
