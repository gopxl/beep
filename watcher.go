package beep

import (
	"container/heap"
)

// Watcher is a mechanism for executing callbacks at specific positions in the audio stream.
//
// Watcher supports two types:
//   - Asynchronous watchers: These are called from a separate goroutine after the Stream() function completes.
//     They do not block the Stream() operation.
//   - Synchronous watchers: These are executed precisely at the specified position in the stream. The source streamer
//     is consumed up to the watch position before triggering the callback. Only after this point, the rest of
//     the source stream will be consumed.
//
// While synchronous watchers offer better accuracy and performance for the overall application, they come with limitations:
//   - Synchronous watchers run within Stream(), so they must not take excessive time to process. Lengthy operations
//     could lead to playback glitches, especially when used in conjunction with the speaker. Slow synchronous watchers may
//     adversely affect the performance of the Streamer pipeline.
//   - When played through the speaker, the speaker remains in a locked state. Attempting to lock the speaker again
//     (e.g., when performed in speaker.Add()) can result in a deadlock.
type Watcher struct {
	s                  Streamer
	pos                int
	streamSyncEvents   eventQueue
	streamAsyncEvents  eventQueue
	drainedSyncEvents  []Trigger
	drainedAsyncEvents []Trigger
}

// Trigger is a callback function invoked by a Watcher when a specific position in the audio stream is reached.
// The 'pos' parameter represents the watched position. For asynchronous watchers, be aware that the stream may
// have progressed beyond 'pos' in the meantime.
type Trigger func(pos int)

// Watch wraps Streamer s in a Watcher.
func Watch(s Streamer) *Watcher {
	return &Watcher{s: s}
}

// Stream populates the 'samples' slice with audio samples from the source streamer,
// checking registered watchers to trigger associated triggers on relevant events.
func (w *Watcher) Stream(samples [][2]float64) (n int, ok bool) {
	defer func() {
		var evts []event
		for len(w.streamAsyncEvents) > 0 {
			evt := w.streamAsyncEvents[0]
			if evt.time > w.pos {
				break
			}
			heap.Pop(&w.streamAsyncEvents)
			evts = append(evts, evt)
		}
		if !ok {
			for _, cb := range w.drainedAsyncEvents {
				evts = append(evts, event{
					time:     w.pos,
					callback: cb,
				})
			}
		}
		if len(evts) > 0 {
			go func() {
				for _, evt := range evts {
					evt.callback(evt.time)
				}
			}()
		}
	}()

	for len(samples) > 0 {
		want := len(samples)

		if len(w.streamSyncEvents) > 0 {
			evt := w.streamSyncEvents[0]
			if evt.time <= w.pos+want {
				want = evt.time - w.pos
			}
		}

		var sn int
		sn, ok = w.s.Stream(samples[:want])
		n += sn
		w.pos += sn

		for len(w.streamSyncEvents) > 0 {
			evt := w.streamSyncEvents[0]
			if evt.time > w.pos {
				break
			}
			heap.Pop(&w.streamSyncEvents)
			evt.callback(w.pos)
		}

		if !ok {
			for _, callback := range w.drainedSyncEvents {
				callback(w.pos)
			}
			w.drainedSyncEvents = nil
		}
		if !ok || sn < want {
			return
		}

		samples = samples[sn:]
	}
	return
}

// Position returns the current playback position.
func (w *Watcher) Position() int {
	return w.pos
}

// Err propagates the original Streamer's errors.
func (w *Watcher) Err() error {
	return w.s.Err()
}

// AtAsync registers a callback to be invoked when the Streamer reaches the specified position 'pos'.
//
// See Watcher for distinctions between synchronous and asynchronous triggers.
func (w *Watcher) AtAsync(pos int, callback Trigger) {
	heap.Push(&w.streamAsyncEvents, event{
		time:     pos,
		callback: callback,
	})
}

// AtSync registers a callback to be invoked when the Streamer reaches the specified position 'pos'.
//
// See Watcher for distinctions between synchronous and asynchronous triggers.
func (w *Watcher) AtSync(pos int, callback Trigger) {
	heap.Push(&w.streamSyncEvents, event{
		time:     pos,
		callback: callback,
	})
}

// StartedAsync registers a callback to be invoked when the Streamer begins streaming.
//
// See Watcher for distinctions between synchronous and asynchronous triggers.
func (w *Watcher) StartedAsync(callback Trigger) {
	w.AtAsync(0, callback)
}

// StartedSync registers a callback to be invoked when the Streamer begins streaming.
//
// See Watcher for distinctions between synchronous and asynchronous triggers.
func (w *Watcher) StartedSync(callback Trigger) {
	w.AtSync(0, callback)
}

// EndedAsync registers callback to be invoked when the source Streamer is fully drained (ok == false).
//
// See Watcher for the difference between synchronous and asynchronous triggers.
func (w *Watcher) EndedAsync(callback Trigger) {
	w.drainedAsyncEvents = append(w.drainedAsyncEvents, callback)
}

// EndedSync registers callback to be invoked when the source Streamer is fully drained (ok == false).
//
// See Watcher for distinctions between synchronous and asynchronous triggers.
func (w *Watcher) EndedSync(callback Trigger) {
	w.drainedSyncEvents = append(w.drainedSyncEvents, callback)
}

type event struct {
	time     int
	callback Trigger
}

// eventQueue can be used as a priority queue. It implements heap.Interface.
type eventQueue []event

func (eq eventQueue) Len() int {
	return len(eq)
}

func (eq eventQueue) Less(i, j int) bool {
	return eq[i].time < eq[j].time
}

func (eq eventQueue) Swap(i, j int) {
	eq[i], eq[j] = eq[j], eq[i]
}

func (eq *eventQueue) Push(x any) {
	item := x.(event)
	*eq = append(*eq, item)
}

func (eq *eventQueue) Pop() any {
	last := len(*eq) - 1
	item := (*eq)[last]
	(*eq)[last] = event{}
	*eq = (*eq)[:last]
	return item
}
