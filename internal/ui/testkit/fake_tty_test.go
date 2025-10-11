package testkit

import (
	"sync"
	"testing"
	"time"
)

func TestFakeTTY_New(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	if tty == nil {
		t.Fatal("NewFakeTTY() returned nil")
	}

	w, h := tty.Size()
	if w != 80 || h != 24 {
		t.Errorf("Size() = (%d, %d), want (80, 24)", w, h)
	}
}

func TestFakeTTY_Enter(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	if err := tty.Enter(); err != nil {
		t.Errorf("Enter() error = %v", err)
	}

	if !tty.IsEntered() {
		t.Error("IsEntered() = false, want true after Enter()")
	}

	if tty.IsExited() {
		t.Error("IsExited() = true, want false after Enter()")
	}
}

func TestFakeTTY_EnterTwice(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	if err := tty.Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	// Second Enter should return error
	err := tty.Enter()
	if err == nil {
		t.Error("Enter() twice did not return error")
	}
}

func TestFakeTTY_Exit(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	tty.Enter()
	if err := tty.Exit(); err != nil {
		t.Errorf("Exit() error = %v", err)
	}

	if tty.IsEntered() {
		t.Error("IsEntered() = true, want false after Exit()")
	}

	if !tty.IsExited() {
		t.Error("IsExited() = false, want true after Exit()")
	}
}

func TestFakeTTY_ExitIdempotent(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	tty.Enter()

	// Multiple exits should not error
	for i := 0; i < 3; i++ {
		if err := tty.Exit(); err != nil {
			t.Errorf("Exit() call %d error = %v", i+1, err)
		}
	}

	if !tty.IsExited() {
		t.Error("IsExited() = false after multiple Exit()")
	}
}

func TestFakeTTY_ExitWithoutEnter(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	// Exit without Enter should not error (idempotent)
	if err := tty.Exit(); err != nil {
		t.Errorf("Exit() without Enter() error = %v", err)
	}

	if !tty.IsExited() {
		t.Error("IsExited() = false after Exit()")
	}
}

func TestFakeTTY_Size(t *testing.T) {
	tests := []struct {
		name string
		w, h int
	}{
		{"standard", 80, 24},
		{"wide", 120, 40},
		{"narrow", 40, 20},
		{"tiny", 10, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tty := NewFakeTTY(tt.w, tt.h)

			w, h := tty.Size()
			if w != tt.w || h != tt.h {
				t.Errorf("Size() = (%d, %d), want (%d, %d)", w, h, tt.w, tt.h)
			}
		})
	}
}

func TestFakeTTY_SetSize(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	tty.SetSize(120, 40)

	w, h := tty.Size()
	if w != 120 || h != 40 {
		t.Errorf("Size() after SetSize() = (%d, %d), want (120, 40)", w, h)
	}
}

func TestFakeTTY_OnResize(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	called := false
	var gotW, gotH int

	tty.OnResize(func(w, h int) {
		called = true
		gotW, gotH = w, h
	})

	tty.SetSize(120, 40)

	if !called {
		t.Error("OnResize callback was not invoked")
	}

	if gotW != 120 || gotH != 40 {
		t.Errorf("OnResize callback received (%d, %d), want (120, 40)", gotW, gotH)
	}
}

func TestFakeTTY_OnResizeMultiple(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	count := 0

	// Register multiple callbacks
	tty.OnResize(func(w, h int) { count++ })
	tty.OnResize(func(w, h int) { count++ })
	tty.OnResize(func(w, h int) { count++ })

	tty.SetSize(100, 30)

	if count != 3 {
		t.Errorf("OnResize callbacks invoked %d times, want 3", count)
	}
}

func TestFakeTTY_OnResizeOrder(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	var order []int

	tty.OnResize(func(w, h int) { order = append(order, 1) })
	tty.OnResize(func(w, h int) { order = append(order, 2) })
	tty.OnResize(func(w, h int) { order = append(order, 3) })

	tty.SetSize(100, 30)

	want := []int{1, 2, 3}
	if len(order) != len(want) {
		t.Errorf("OnResize order = %v, want %v", order, want)
		return
	}

	for i, v := range want {
		if order[i] != v {
			t.Errorf("OnResize order[%d] = %d, want %d", i, order[i], v)
		}
	}
}

func TestFakeTTY_Reset(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	tty.Enter()
	tty.OnResize(func(w, h int) {})

	tty.Reset()

	if tty.IsEntered() {
		t.Error("IsEntered() = true after Reset(), want false")
	}

	if tty.IsExited() {
		t.Error("IsExited() = true after Reset(), want false")
	}

	// Verify callbacks cleared (no panic on SetSize)
	tty.SetSize(100, 30)
}

func TestFakeTTY_ConcurrentSize(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent Size() readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = tty.Size()
			}
		}()
	}

	wg.Wait()
}

func TestFakeTTY_ConcurrentSetSize(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent SetSize() writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tty.SetSize(80+id, 24+id)
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state (any size is valid)
	w, h := tty.Size()
	if w < 80 || w >= 90 || h < 24 || h >= 34 {
		t.Errorf("Size() = (%d, %d), out of expected range", w, h)
	}
}

func TestFakeTTY_ConcurrentResize(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	var callCount int
	var mu sync.Mutex

	tty.OnResize(func(w, h int) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent SetSize() triggering callbacks
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tty.SetSize(80+id, 24+id)
			}
		}(i)
	}

	wg.Wait()

	// Verify callbacks invoked
	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != goroutines*10 {
		t.Errorf("OnResize callback invoked %d times, want %d", count, goroutines*10)
	}
}

func TestFakeTTY_ConcurrentEnterExit(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Concurrent Enter/Exit
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				tty.Enter()
				time.Sleep(1 * time.Millisecond)
				tty.Exit()
			}
		}()
	}

	wg.Wait()

	// Final state: exited
	if !tty.IsExited() {
		t.Error("IsExited() = false after concurrent Enter/Exit, want true")
	}
}

func TestFakeTTY_SizeAfterResize(t *testing.T) {
	tty := NewFakeTTY(80, 24)

	sizes := [][2]int{
		{100, 30},
		{120, 40},
		{80, 24},
		{40, 20},
	}

	for _, size := range sizes {
		tty.SetSize(size[0], size[1])

		w, h := tty.Size()
		if w != size[0] || h != size[1] {
			t.Errorf("Size() after SetSize(%d, %d) = (%d, %d)", size[0], size[1], w, h)
		}
	}
}
