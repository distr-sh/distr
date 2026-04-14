package util

import "context"

type ToggleableGoroutine struct {
	fn       func(context.Context)
	cancelFn func()
}

func NewToggleableGoroutine(fn func(context.Context)) *ToggleableGoroutine {
	return &ToggleableGoroutine{fn: fn}
}

func (t *ToggleableGoroutine) GoOrCancel(ctx context.Context, v bool) string {
	if v && t.cancelFn == nil {
		t.Go(ctx)
		return "started"
	} else if !v && t.cancelFn != nil {
		t.Cancel()
		return "stopped"
	}

	return "unchanged"
}

func (t *ToggleableGoroutine) Go(ctx context.Context) {
	t.Cancel()
	ctx, cancel := context.WithCancel(ctx)
	t.cancelFn = cancel
	go t.fn(ctx)
}

func (t *ToggleableGoroutine) Cancel() {
	if t.cancelFn != nil {
		t.cancelFn()
		t.cancelFn = nil
	}
}
