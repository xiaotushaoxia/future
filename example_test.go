package future

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func f1(ctx context.Context) int {
	t := time.NewTimer(time.Second)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return 0
	case <-t.C:
		return 1
	}
}

func tryGetEvenNumber(ctx context.Context) (int, error) {
	t := time.NewTimer(time.Second)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-t.C:
		intn := rand.Intn(10) + 1 // avoid 0
		if intn%2 == 0 {
			return intn, nil
		} else {
			return 0, fmt.Errorf("not even number")
		}
	}
}

func ExampleNew() {
	future1 := New(f1)
	fmt.Println(future1.Get()) // output:  1, nil

	future2 := New(f1)
	future2.Cancel()
	fmt.Println(future2.Get()) // output:  0, future is canceled

	future3 := New(f1)
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Millisecond*100)
	fmt.Println(future3.GetContext(ctx)) // output:  0, context deadline exceeded
	cancelFunc()
}

func ExampleNewResultError() {
	future1 := NewResultError(tryGetEvenNumber)
	fmt.Println(future1.Get()) // output:  `2, nil`  or `0, not even number`

	future2 := NewResultError(tryGetEvenNumber)
	future2.Cancel()
	fmt.Println(future2.Get()) // output:   0, future is canceled
}

func ExampleWrappedRecovered() {
	future1 := New(forTestPanic)
	//future1 := NewResultError(forTestPanic2)

	defer func() {
		if p := recover(); p != nil {
			if recovered, ok := p.(*WrappedRecovered); ok {
				fmt.Println(recovered.Value)
				fmt.Println(string(recovered.Stack))
				// output:
				// test panic
				// goroutine 7 [running]:
				// panic({0xf47180?, 0xfa8880?})
				// 	/Go1.22.4/src/runtime/panic.go:770 +0x132
				// future.pp({0x0?, 0x0?})
				// 	/src/future/example_test.go:78 +0x25
				// future.(*Future[...]).run(0x0)
				//	/src/future/future.go:189 +0x9a
			} else {
				// should not happen. Get only panic *WrappedRecovered
			}
		}
	}()
	time.Sleep(time.Second * 3)
	fmt.Println(future1.Get()) // output:  1, nil
}

func forTestPanic(ctx context.Context) int {
	panic("test panic")
}

func forTestPanic2(ctx context.Context) (int, error) {
	panic("test panic")
	return 1, nil
}

func Test_ExampleNew(t *testing.T) {
	ExampleNew()
}

func Test_ExampleNewResultError(t *testing.T) {
	ExampleNewResultError()
}

func Test_ExampleWrappedRecovered(t *testing.T) {
	ExampleWrappedRecovered()
}
