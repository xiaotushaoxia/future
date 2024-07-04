package future

import (
	"bufio"
	"bytes"
	"io"
	"runtime/debug"
	"sync"
)

type stringError string

func (s stringError) Error() string {
	return string(s)
}

func stack(n int) []byte {
	lines := bytes2lines(debug.Stack())
	if n == 0 || len(lines) <= n*2+1 { // if n is too large. not skip
		return bytes.Join(lines, []byte{})
	}
	lines = append([][]byte{lines[0]}, lines[2*n+1:]...)
	return bytes.Join(lines, []byte{})
}

func bytes2lines(bs []byte) [][]byte {
	reader := bufio.NewReader(bytes.NewReader(bs))
	var lines = make([][]byte, 0)
	for {
		readBytes, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			// todo not sure what to do... maybe just ignore it
			break
		}
		lines = append(lines, readBytes)
	}
	return lines
}

type atomicInt32 struct {
	m sync.Mutex
	v int32
}

func (i *atomicInt32) Load() int32 {
	i.m.Lock()
	defer i.m.Unlock()
	return i.v
}

func (i *atomicInt32) Store(v int32) {
	i.m.Lock()
	defer i.m.Unlock()
	i.v = v
}

func (i *atomicInt32) StoreAndDo(v int32, fs ...func()) {
	i.m.Lock()
	defer i.m.Unlock()
	i.v = v
	for _, f := range fs {
		f()
	}
}

func (i *atomicInt32) CompareAndSwap(oldV int32, newV int32) bool {
	i.m.Lock()
	defer i.m.Unlock()
	if i.v == oldV {
		i.v = newV
		return true
	}
	return false
}
