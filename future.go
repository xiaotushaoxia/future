package future

import (
	"context"
	"sync"
	"sync/atomic"
)

func New[T any](rf ResultFunc[T]) *Future[T] {
	f := newFutureInner(rf)
	go f.run()
	return f
}

func NewWithPool[T any](rf ResultFunc[T], pool Pool) *Future[T] {
	f := newFutureInner(rf)
	pool.Go(f.run)
	return f
}

func NewResultError[T any](rf ResultErrorFunc[T]) *ResultErrorFuture[T] {
	f2 := newResultErrorFuture(rf)
	go f2.run()
	return f2
}

func NewResultErrorWithPool[T any](rf ResultErrorFunc[T], pool Pool) *ResultErrorFuture[T] {
	f2 := newResultErrorFuture(rf)
	pool.Go(f2.run)
	return f2
}

type Future[T any] struct {
	result atomic.Pointer[result[T]]

	doneCloseOnce sync.Once
	done          chan struct{}

	state atomicInt32

	resultFunc ResultFunc[T]
	ctx        context.Context
	cancel     context.CancelFunc

	cancelOrFinishOnce sync.Once

	// close when child goroutine exit
	finish chan struct{}
}

// Get
// Waits if necessary for the computation to complete, and then retrieves its result, if available.
// Returns:
// the computed result and nil, or T.Zero and the error causing retrieve result failed
// Note:
// panic if child goroutine panic
func (f *Future[T]) Get() (T, error) {
	return f.GetContext(context.Background())
}

// GetContext
// Waits if necessary for the computation to complete or ctx is Done, and then retrieves its result, if available.
// Returns:
// the computed result and nil, or T.Zero and the error causing retrieve result failed
// Note:
// panic if child goroutine  panic
func (f *Future[T]) GetContext(ctx context.Context) (t T, err error) {
	select {
	case <-ctx.Done():
		return t, ctx.Err()
	case <-f.ctx.Done():
		return t, Canceled
	case <-f.done:
		select {
		case <-f.ctx.Done(): // select is random if more than 1 case readable
			return t, Canceled
		default:
			return f.get(), nil
		}
	}
}

// Cancel
// Attempts to cancel execution of this task.
// This attempt will fail if the task has already completed, has already been cancelled, or could not be cancelled for some other reason.
// If successful
// 1. This task has not started when cancel is called, this task should never run.
// 2. If the task has already started, call f.Cancel to tell the child goroutine to exit.
//
// After this method returns, subsequent calls to IsDone() will always return true.
// Subsequent calls to IsCancelled() will always return true if this method returned true.
// Returns:
// false if the task could not be cancelled, typically because it has already completed normally; true otherwise
func (f *Future[T]) Cancel() bool {
	select {
	case <-f.done:
		return false
	case <-f.ctx.Done():
		return false
	default:
		callCancel := false
		f.cancelOrFinishOnce.Do(func() {
			// this is the only way to cancel f.ctx
			f.state.StoreAndDo(int32(StateCanceling), f.cancel, f.closeDone)
			callCancel = true
		})
		return callCancel
	}
}

// IsCanceled
// Returns true if this task was cancelled before it completed normally.
// Returns:
// true if this task was cancelled before it completed
func (f *Future[T]) IsCanceled() bool {
	select {
	case <-f.ctx.Done():
		return true
	default:
		return false
	}
}

// IsDone
// Returns true if this task completed.
// Completion may be due to normal termination, an exception, or cancellation -- in all of these cases, this method will return true.
// When return true, call Get can return immediately
// Returns:
// true if this task completed
func (f *Future[T]) IsDone() bool {
	select {
	case <-f.done:
		return true
	default:
		return false
	}
}

// State
// Possible state transition diagram.
// StatePending -> StateRunning => StateDone/StatePanicDone
// StatePending -> StateRunning -> (call f.Cancel() and return true) => StateCanceling => StateCanceled
// StatePending -> (call f.Cancel() and return true) => StateCanceling => StateCanceled
// StatePending, StateRunning means Future is not Done
// Returns:
// StatePending, StateRunning, StateCanceling, StateCanceled, StateDone, StatePanicDone
func (f *Future[T]) State() State {
	return State(f.state.Load())
}

// Done
// Returns:
// chan who will be closed when Future is Done (see IsDone)
func (f *Future[T]) Done() <-chan struct{} {
	return f.done
}

// IsFinished
// Returns:
// true if child goroutine is exited.
func (f *Future[T]) IsFinished() bool {
	select {
	case <-f.finish:
		return true
	default:
		return false
	}
}

// Finished
// maybe kind of confused. Done? Finished?
// Done means Get can return immediately, Finished means child goroutine exits
// Returns:
// chan who will be closed when child goroutine exits
func (f *Future[T]) Finished() <-chan struct{} {
	return f.finish
}

func (f *Future[T]) get() T {
	load := f.result.Load()
	if load.p != nil {
		panic(load.p)
	}
	return load.t
}

func (f *Future[T]) run() {
	var t T
	defer func() {
		f.handleExit(t, recover())
	}()
	if f.state.CompareAndSwap(int32(StatePending), int32(StateRunning)) {
		t = f.resultFunc(f.ctx)
	} // else means f.Cancel is called
}

func (f *Future[T]) handleExit(t T, recovered any) {
	var callFinish bool
	f.cancelOrFinishOnce.Do(func() {
		callFinish = true
		temp := &result[T]{}
		if recovered != nil {
			// 7 means skip debug.Stack, stack, do func, once.doSlow, once.Do, handleExit, defer func
			temp.p = &WrappedRecovered{Value: recovered, Stack: stack(7)}
		} else {
			temp.t = t
		}
		f.result.Store(temp)
		f.state.StoreAndDo(int32(temp.exitState()), f.closeDone)
	})
	if !callFinish {
		f.state.StoreAndDo(int32(StateCanceled), f.closeDone)
	}

	// f.state.StoreAndDo make Store and closeDone operations atomic
	// for example:
	// <-f.Done()
	// s :=f.State()
	// make sure `s` won't be StateRunning or StatePending or StateCanceling

	close(f.finish)
}

func (f *Future[T]) closeDone() {
	f.doneCloseOnce.Do(func() {
		close(f.done)
	})
}

// ResultErrorFuture
// Syntactic sugar, same as Future[*ResultWithError[T]]
type ResultErrorFuture[T any] struct {
	*Future[*ResultWithError[T]]
}

func (f *ResultErrorFuture[T]) Get() (T, error) {
	return f.GetContext(context.Background())
}

func (f *ResultErrorFuture[T]) GetContext(ctx context.Context) (t T, err error) {
	// there are two bad choices.
	// Option 1: `return t, err, getErr`, which is too ugly.
	// Option 2: `return t, err`, which mixes errors together.
	// For the following reasons, I choose option 2.
	// 1. User **maybe** not care why Get return error. err just means Get failed.
	// 2. By calling `f.IsDone() and f.IsCanceled()` or `ctx.Err()`, user can determine why Get() returned an error.
	te, er := f.Future.GetContext(ctx)
	if er != nil {
		return t, er
	}
	if te.Error != nil {
		return t, te.Error
	}
	return te.Result, nil
}

func newFutureInner[T any](rf ResultFunc[T]) *Future[T] {
	f := Future[T]{done: make(chan struct{}), resultFunc: rf, finish: make(chan struct{})}
	// use Background as root, no goroutine will leak even if f.cancel not called
	f.ctx, f.cancel = context.WithCancel(context.Background())
	return &f
}

func newResultErrorFuture[T any](rf ResultErrorFunc[T]) *ResultErrorFuture[T] {
	return &ResultErrorFuture[T]{
		Future: newFutureInner(
			func(ctx context.Context) *ResultWithError[T] {
				r, e := rf(ctx)
				return &ResultWithError[T]{Error: e, Result: r}
			},
		),
	}
}
