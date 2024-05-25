package workerdsdk

import "sync"

type FreeIdx struct {
	freeIdx []uint16
	lock    sync.Mutex
}

func newFreeIdx() *FreeIdx {
	return &FreeIdx{freeIdx: make([]uint16, 0), lock: sync.Mutex{}}
}

func (f *FreeIdx) push(value uint16) {
	f.lock.Lock()
	defer f.lock.Unlock()

	f.freeIdx = append(f.freeIdx, value)
}

func (f *FreeIdx) pop() uint16 {
	if len(f.freeIdx) < 1 {
		panic("FreeIdx is empty")
	}

	length := len(f.freeIdx)
	v := f.freeIdx[length-1]
	f.freeIdx = f.freeIdx[0 : length-1]
	return v
}

func (f *FreeIdx) length() int {
	return len(f.freeIdx)
}
