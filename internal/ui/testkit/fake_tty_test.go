package testkit

import (
	"sync"
	"testing"
)

func TestFakeTTY_EnterExit(t *testing.T) {
	t.Parallel()
	tty := NewFakeTTY(80, 24)

	if tty.InRawMode() {
		t.Error("should not be in raw mode initially")
	}

	err := tty.Enter()
	if err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	if !tty.InRawMode() {
		t.Error("should be in raw mode after Enter()")
	}

	err = tty.Exit()
	if err != nil {
		t.Fatalf("Exit() error = %v", err)
	}

	if tty.InRawMode() {
		t.Error("should not be in raw mode after Exit()")
	}
}

func TestFakeTTY_Size(t *testing.T) {
	t.Parallel()
	tty := NewFakeTTY(120, 40)

	w, h := tty.Size()
	if w != 120 {
		t.Errorf("Size() width = %d, want 120", w)
	}

	if h != 40 {
		t.Errorf("Size() height = %d, want 40", h)
	}
}

func TestFakeTTY_OnResize(t *testing.T) {
	t.Parallel()
	tty := NewFakeTTY(80, 24)

	var (
		called     bool
		gotW, gotH int
		mu         sync.Mutex
	)

	tty.OnResize(func(w, h int) {
		mu.Lock()
		defer mu.Unlock()

		called = true
		gotW, gotH = w, h
	})

	tty.SetSize(120, 40)

	mu.Lock()
	defer mu.Unlock()

	if !called {
		t.Error("OnResize callback was not invoked")
	}

	if gotW != 120 || gotH != 40 {
		t.Errorf("callback received (%d, %d), want (120, 40)", gotW, gotH)
	}
}

func TestFakeTTY_ConcurrentResize(t *testing.T) {
	t.Parallel()
	tty := NewFakeTTY(80, 24)

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Read size repeatedly.
	go func() {
		defer wg.Done()

		for range 100 {
			_, _ = tty.Size()
		}
	}()

	// Goroutine 2: Resize.
	go func() {
		defer wg.Done()

		for i := range 100 {
			tty.SetSize(80+i, 24+i)
		}
	}()

	wg.Wait()
}
