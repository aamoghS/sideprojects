package sync

import "minstd/atomic"

type Mutex struct {
	locked uint32
}

func (m *Mutex) Lock() {
	for !atomic.CompareAndSwapUint32(&m.locked, 0, 1) {
	}
}

func (m *Mutex) Unlock() {
	atomic.CompareAndSwapUint32(&m.locked, 1, 0)
}
