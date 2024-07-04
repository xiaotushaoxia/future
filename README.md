# future

`future` implementation Java Future

# reference docs
[Java Future](https://docs.oracle.com/javase/8/docs/api/index.html?java/util/concurrent/Future.html)

# usage
example_test.go
```go
package future

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

func f1(ctx context.Context) int {
	select {
	case <-ctx.Done():
		return 0
	case <-time.After(time.Second):
		return 1
	}
}

func tryGetEvenNumber(ctx context.Context) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(time.Second):
		intn := rand.Intn(10) + 1 // avoid 0
		if intn%2 == 1 {
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

}

func ExampleNewResultError() {
	future1 := NewResultError(tryGetEvenNumber)
	fmt.Println(future1.Get()) // output:  `2, nil`  or `0, not even number`

	future2 := NewResultError(tryGetEvenNumber)
	future2.Cancel()
	fmt.Println(future2.Get()) // output:   0, future is canceled
}

```

