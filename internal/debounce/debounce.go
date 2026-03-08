package debounce

import (
	"sync"
	"time"
)

// Debouncer delays execution of a function until after a quiet period.
// If triggered multiple times within the interval, only the last trigger fires.
type Debouncer struct {
	interval time.Duration
	mu       sync.Mutex
	timer    *time.Timer
}

// New creates a Debouncer with the given quiet interval.
func New(interval time.Duration) *Debouncer {
	return &Debouncer{interval: interval}
}

// Trigger resets the debounce timer. fn will be called after the quiet interval.
// If Trigger is called again before the interval elapses, the timer resets.
func (d *Debouncer) Trigger(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.interval, fn)
}

// Stop cancels any pending debounced call.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
