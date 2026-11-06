package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// loop runs timers only. I/O is synchronous, so waiting on timers is all it does.
// Each callback is called from Go so that goja runs pending promise jobs after it.
type loop struct {
	vm        *goja.Runtime
	inspect   func(goja.Value) string
	timers    []*timer
	nextID    int
	unhandled []*goja.Promise
}

type timer struct {
	id   int
	at   time.Time
	fn   goja.Callable
	args []goja.Value
}

func newLoop(vm *goja.Runtime, inspect func(goja.Value) string) *loop {
	l := &loop{vm: vm, inspect: inspect}
	vm.SetPromiseRejectionTracker(func(p *goja.Promise, op goja.PromiseRejectionOperation) {
		if op == goja.PromiseRejectionReject {
			l.unhandled = append(l.unhandled, p)
		} else {
			l.unhandled = slices.DeleteFunc(l.unhandled, func(q *goja.Promise) bool { return q == p })
		}
	})
	vm.Set("setTimeout", func(fn goja.Callable, ms float64, args ...goja.Value) int {
		l.nextID++
		l.timers = append(l.timers, &timer{l.nextID, time.Now().Add(time.Duration(ms * float64(time.Millisecond))), fn, args})
		return l.nextID
	})
	vm.Set("clearTimeout", func(id int) {
		l.timers = slices.DeleteFunc(l.timers, func(t *timer) bool { return t.id == id })
	})
	return l
}

// run executes the script, then fires timers in order until none remain.
func (l *loop) run(code string) error {
	if _, err := l.vm.RunString(code); err != nil {
		return err
	}
	for {
		if err := l.checkRejections(); err != nil {
			return err
		}
		if len(l.timers) == 0 {
			return nil
		}
		// Earliest first; ties keep insertion order.
		t := slices.MinFunc(l.timers, func(a, b *timer) int { return a.at.Compare(b.at) })
		l.timers = slices.DeleteFunc(l.timers, func(x *timer) bool { return x == t })
		time.Sleep(time.Until(t.at))
		if _, err := t.fn(goja.Undefined(), t.args...); err != nil {
			return err
		}
	}
}

func (l *loop) checkRejections() error {
	if len(l.unhandled) == 0 {
		return nil
	}
	return fmt.Errorf("Uncaught (in promise) %s", strings.TrimSpace(l.inspect(l.unhandled[0].Result())))
}
