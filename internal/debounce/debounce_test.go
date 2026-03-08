package debounce_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/lokeshreddygoli/hotreload/internal/debounce"
)

func TestDebouncerFiresOnce(t *testing.T) {
	db := debounce.New(50 * time.Millisecond)
	defer db.Stop()

	var count atomic.Int32
	for i := 0; i < 10; i++ {
		db.Trigger(func() { count.Add(1) })
	}

	time.Sleep(150 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestDebouncerRespectsInterval(t *testing.T) {
	db := debounce.New(50 * time.Millisecond)
	defer db.Stop()

	var count atomic.Int32
	db.Trigger(func() { count.Add(1) })

	time.Sleep(10 * time.Millisecond)
	if got := count.Load(); got != 0 {
		t.Errorf("expected 0 calls before interval, got %d", got)
	}

	time.Sleep(100 * time.Millisecond)
	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 call after interval, got %d", got)
	}
}

func TestDebouncerResetsOnEachTrigger(t *testing.T) {
	db := debounce.New(60 * time.Millisecond)
	defer db.Stop()

	var count atomic.Int32
	for i := 0; i < 3; i++ {
		db.Trigger(func() { count.Add(1) })
		time.Sleep(40 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestDebouncerStopCancelsPending(t *testing.T) {
	db := debounce.New(100 * time.Millisecond)

	var count atomic.Int32
	db.Trigger(func() { count.Add(1) })
	db.Stop()

	time.Sleep(200 * time.Millisecond)

	if got := count.Load(); got != 0 {
		t.Errorf("expected 0 calls after Stop, got %d", got)
	}
}

func TestDebouncerMultipleBursts(t *testing.T) {
	db := debounce.New(50 * time.Millisecond)
	defer db.Stop()

	var count atomic.Int32
	fn := func() { count.Add(1) }

	for i := 0; i < 5; i++ {
		db.Trigger(fn)
	}
	time.Sleep(100 * time.Millisecond)
	if got := count.Load(); got != 1 {
		t.Errorf("after first burst: expected 1, got %d", got)
	}

	for i := 0; i < 5; i++ {
		db.Trigger(fn)
	}
	time.Sleep(100 * time.Millisecond)
	if got := count.Load(); got != 2 {
		t.Errorf("after second burst: expected 2, got %d", got)
	}
}
