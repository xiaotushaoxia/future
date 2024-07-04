package future

import (
	"context"
	"fmt"
)

type (
	ResultFunc[T any]      func(ctx context.Context) T
	ResultErrorFunc[T any] func(ctx context.Context) (T, error)
	// State
	// 0 -> 1 -> 4/5
	// 0 -> 1 -> (call Cancel) -> 2 -> 3
	// 0 -> (call Cancel) -> 3
	State int32
)

const (
	StatePending   State = 0
	StateRunning   State = 1
	StateCanceling State = 2
	StateCanceled  State = 3
	StateDone      State = 4
	StatePanicDone State = 5

	Canceled = stringError("future is canceled")
)

type ResultWithError[T any] struct {
	Error  error
	Result T
}

type Pool interface {
	Go(func())
}

// WrappedRecovered
// When a child goroutine panics, the stack trace is lost. Use this to obtain the stack trace of the child goroutine.
type WrappedRecovered struct {
	Value any
	Stack []byte
}

func (p *WrappedRecovered) String() string {
	return fmt.Sprintf("panic: %v\nstack:\n%s\n", p.Value, p.Stack)
}

type result[T any] struct {
	t T
	p any
}

func (r *result[T]) exitState() State {
	if r.p == nil {
		return StatePanicDone
	}
	return StateDone
}
