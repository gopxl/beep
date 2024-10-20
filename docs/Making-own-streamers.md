Beep offers a lot of pre-made streamers, but sometimes that's not enough. Fortunately, making new ones isn't very hard and in this part, we'll learn just that.

So, what's a streamer? It's this interface:

```go
type Streamer interface {
    Stream(samples [][2]float64) (n int, ok bool)
    Err() error
}
```

Read [the docs](https://godoc.org/github.com/gopxl/beep#Streamer) for more details.

> **Why does `Stream` return a `bool` and error handling is moved to a separate `Err` method?** The main reason is to prevent one faulty streamer from ruining your whole audio pipeline, yet make it possible to catch the error and handle it somehow.
>
> How would a single faulty streamer ruin your whole pipeline? For example, there's a streamer called [`beep.Mixer`](https://godoc.org/github.com/gopxl/beep#Mixer), which mixes multiple streamers together and makes it possible to add streamers dynamically to it. The [`speaker`](https://godoc.org/github.com/gopxl/beep/speaker) package uses `beep.Mixer` under the hood. The mixer works by gathering samples from all of the streamers added to it and adding those together. If the `Stream` method returned an error, what should the mixer's `Stream` method return if one of its streamers errored? There'd be two choices: either it returns the error but halts its own playback, or it doesn't return it, thereby making the error inaccessible. Neither choice is good and that's why the `Streamer` interface is designed as it is.

To make our very own streamer, all that's needed is implementing that interface. Let's get to it!

## Noise generator

This will probably be the simplest streamer ever. It'll stream completely random samples, resulting in a noise. To implement any interface, we need to make a type. The noise generator requires no state, so it'll be an empty struct:

```go
type Noise struct{}
```

Now we need to implement the `Stream` method. It receives a slice and it should fill it will samples. Then it should return how many samples it filled and a `bool` depending on whether it was already drained or not. The noise generator will stream forever, so it will always fully fill the slice and return `true`.

The samples are expected to be values between -1 and +1 (including). We fill the slice using a simple for-loop:

```go
func (no Noise) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		samples[i][0] = rand.Float64()*2 - 1
		samples[i][1] = rand.Float64()*2 - 1
	}
	return len(samples), true
}
```

The last thing remaining is the `Err` method. The noise generator can never malfunction, so `Err` always returns `nil`:

```go
func (no Noise) Err() error {
	return nil
}
```

Now it's done and we can use it in a program:

```go
func main() {
	sr := beep.SampleRate(44100)
	speaker.Init(sr, sr.N(time.Second/10))
	speaker.Play(Noise{})
	select {}
}
```

This will play noise indefinitely. Or, if we only want to play it for a certain period of time, we can use [`beep.Take`](https://godoc.org/github.com/gopxl/beep#Take):

```go
func main() {
	sr := beep.SampleRate(44100)
	speaker.Init(sr, sr.N(time.Second/10))

	done := make(chan bool)
	speaker.Play(beep.Seq(beep.Take(sr.N(5*time.Second), Noise{}), beep.Callback(func() {
		done <- true
	})))
	<-done
}
```

This will play noise for 5 seconds.

Since streamers that never fail are fairly common, Beep provides a helper type called [`beep.StreamerFunc`](https://godoc.org/github.com/gopxl/beep#StreamerFunc), which is defined like this:

```go
type StreamerFunc func(samples [][2]float64) (n int, ok bool)
```

It implements the `Streamer` interface by calling itself from the `Stream` method and always returning `nil` from the `Err` method. As you can surely see, we can simplify our `Noise` streamer definition by getting rid of the custom type and using `beep.StreamerFunc` instead:

```go
func Noise() beep.Streamer {
	return beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		for i := range samples {
			samples[i][0] = rand.Float64()*2 - 1
			samples[i][1] = rand.Float64()*2 - 1
		}
		return len(samples), true
	})
}
```

We've changed the streamer from a struct to a function, so we need to replace all `Noise{}` with `Noise()`, but other than that, everything will be the same.

## Queue

Well, that was simple. How about something more complex?

Remember [`beep.Seq`](https://godoc.org/github.com/gopxl/beep#Seq)? It takes a bunch of streamers and streams them one by one. Here, we'll do something similar, but dynamic. We'll make a queue.

Initially, it'll be empty and it'll stream silence. But, we'll be able to use its `Add` method to add streamers to it. It'll add them to the queue and play them one by one. We will be able to call `Add` at any time and add more songs to the queue. They'll start playing when all the previous songs finish.

Let's get to it! The `Queue` type needs just one thing to remember: the streamers left to play.

```go
type Queue struct {
	streamers []beep.Streamer
}
```

We need a method to add new streamers to the queue:

```go
func (q *Queue) Add(streamers ...beep.Streamer) {
	q.streamers = append(q.streamers, streamers...)
}
```

Now, all that's remaining is to implement the streamer interface. Here's the `Stream` method with comments for understanding:

```go
func (q *Queue) Stream(samples [][2]float64) (n int, ok bool) {
	// We use the filled variable to track how many samples we've
	// successfully filled already. We loop until all samples are filled.
	filled := 0
	for filled < len(samples) {
		// There are no streamers in the queue, so we stream silence.
		if len(q.streamers) == 0 {
			for i := range samples[filled:] {
				samples[i][0] = 0
				samples[i][1] = 0
			}
			break
		}

		// We stream from the first streamer in the queue.
		n, ok := q.streamers[0].Stream(samples[filled:])
		// If it's drained, we pop it from the queue, thus continuing with
		// the next streamer.
		if !ok {
			q.streamers = q.streamers[1:]
		}
		// We update the number of filled samples.
		filled += n
	}
	return len(samples), true
}
```

And here's the trivial `Err` method:

```go
func (q *Queue) Err() error {
	return nil
}
```

Alright! Now we've gotta use the queue somehow. Here's how we're gonna use it: we'll let the user type the name of a file on the command line and we'll load the file and add it to the queue. If there were no songs in the queue before, it'll start playing immediately. Of course, it'll be a little cumbersome, because there will be no tab-completion, but whatever, it'll be something.

Here's how it's done (again, with comments):

```go
func main() {
	sr := beep.SampleRate(44100)
	speaker.Init(sr, sr.N(time.Second/10))

	// A zero Queue is an empty Queue.
	var queue Queue
	speaker.Play(&queue)

	for {
		var name string
		fmt.Print("Type an MP3 file name: ")
		fmt.Scanln(&name)

		// Open the file on the disk.
		f, err := os.Open(name)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Decode it.
		streamer, format, err := mp3.Decode(f)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// The speaker's sample rate is fixed at 44100. Therefore, we need to
		// resample the file in case it's in a different sample rate.
		resampled := beep.Resample(4, format.SampleRate, sr, streamer)

		// And finally, we add the song to the queue.
		speaker.Lock()
		queue.Add(resampled)
		speaker.Unlock()
	}
}
```

And that's it!

We've learned a lot today. _Now, go, brave programmer, make new streamers, make new music players, make new artificial sound generators, whatever, go make the world a better place!_

> **Why isn't the `Queue` type implemented in Beep?** So that I could make this tutorial.