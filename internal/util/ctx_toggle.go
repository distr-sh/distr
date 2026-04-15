package util

import (
	"context"
	"sync"
)

type ToggleableGoroutine struct {
	fn       func(context.Context)
	cancelFn func()
	mut      sync.Mutex
}

func NewToggleableGoroutine(fn func(context.Context)) *ToggleableGoroutine {
	return &ToggleableGoroutine{fn: fn}
}

func (t *ToggleableGoroutine) GoOrCancel(ctx context.Context, v bool) string {
	t.mut.Lock()
	defer t.mut.Unlock()
	if v && t.cancelFn == nil {
		t.goNoLock(ctx)
		return "started"
	} else if !v && t.cancelFn != nil {
		t.cancelNoLock()
		return "stopped"
	}

	return "unchanged"
}

func (t *ToggleableGoroutine) Go(ctx context.Context) {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.goNoLock(ctx)
}

func (t *ToggleableGoroutine) goNoLock(ctx context.Context) {
	t.cancelNoLock()
	ctx, cancel := context.WithCancel(ctx)
	t.cancelFn = cancel
	go func() {
		defer t.Cancel()
		t.fn(ctx)
	}()
}

func (t *ToggleableGoroutine) Cancel() {
	t.mut.Lock()
	defer t.mut.Unlock()
	t.cancelNoLock()
}

func (t *ToggleableGoroutine) cancelNoLock() {
	if t.cancelFn != nil {
		t.cancelFn()
		t.cancelFn = nil
	}
}
