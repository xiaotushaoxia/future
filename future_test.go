package future

import (
	"fmt"
	"testing"
)

func TestFuture_run(t *testing.T) {
	var k int
	defer ccc(k)
	k = 10
	fmt.Println(k)
}

func ccc(i int) {
	fmt.Println(i)
}
